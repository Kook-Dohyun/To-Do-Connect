package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const googleToolPrefix = "google_"
const googleTasksBase = "https://tasks.googleapis.com/tasks/v1"
const googleListsPath = "/users/@me/lists"
const googleListTemplate = googleListsPath + "/{list}"
const googleTaskCollection = "/lists/{list}/tasks"
const googleTaskTemplate = googleTaskCollection + "/{task}"

var errGoogleAPIDisabled = errors.New("API_DISABLED")
var errGoogleTasksPermission = errors.New("TASKS_PERMISSION_MISSING")

type googleInput struct {
	ConnectionID string            `json:"connection_id" jsonschema:"Explicit Google connection ID from list_connections"`
	ListID       string            `json:"list_id,omitempty" jsonschema:"List ID from google_list_lists; not a name"`
	TaskID       string            `json:"task_id,omitempty" jsonschema:"Task ID from this connection and list"`
	Body         object            `json:"body,omitempty" jsonschema:"Google Tasks JSON body. Use notes for plain text, status=completed or needsAction, due as RFC3339 date (time is discarded). Parent and position are controlled by insert/move query arguments."`
	Query        map[string]string `json:"query,omitempty" jsonschema:"Endpoint query parameters including pageToken, maxResults and task filters. Move supports parent, previous, destinationTasklist. Never pass credentials or URLs."`
	Confirm      bool              `json:"confirm,omitzero" jsonschema:"True only when the user explicitly requested deletion, clearing completed tasks or moving this task"`
}

type googleOperation struct {
	operation
	query   string
	body    bool
	confirm bool
}

func googleOperations() []googleOperation {
	return []googleOperation{
		{operation{"list_lists", "GET", googleListsPath, "List Google task lists, one page; follow nextPageToken using query.pageToken."}, "maxResults pageToken fields", false, false},
		{operation{"create_list", "POST", googleListsPath, "Create a Google task list. body: title."}, "fields", true, false},
		{operation{"patch_list", "PATCH", googleListTemplate, "Partially update a Google task list. body: title."}, "fields", true, false},
		{operation{"delete_list", "DELETE", googleListTemplate, "Delete a Google task list and its tasks. Requires confirm=true."}, "", false, true},
		{operation{"list_tasks", "GET", googleTaskCollection, "List tasks, one page. Set showCompleted/showHidden explicitly when needed; follow nextPageToken."}, "maxResults pageToken completedMax completedMin dueMax dueMin showCompleted showDeleted showHidden showAssigned updatedMin fields", false, false},
		{operation{"create_task", "POST", googleTaskCollection, "Create a Google task. body: title, notes, status, due. Optional query.parent and query.previous position it."}, "parent previous fields", true, false},
		{operation{"patch_task", "PATCH", googleTaskTemplate, "Partially update task fields. Complete/reopen with status=completed/needsAction. due has date-only semantics."}, "fields", true, false},
		{operation{"delete_task", "DELETE", googleTaskTemplate, "Delete a task. Deleting an assigned task may also affect its originating Docs/Chat task. Requires confirm=true."}, "", false, true},
		{operation{"move_task", "POST", googleTaskTemplate + "/move", "Move/reorder using query.parent, previous, destinationTasklist. Same account only; API restrictions apply to assigned/repeating tasks. Requires confirm=true."}, "parent previous destinationTasklist fields", false, true},
	}
}

type googleTasks struct {
	base  string
	http  *http.Client
	token func(context.Context) (string, error)
}

func newGoogleTasks(token func(context.Context) (string, error)) *googleTasks {
	return &googleTasks{base: googleTasksBase, token: token, http: &http.Client{Timeout: 90 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

func (g *googleTasks) execute(ctx context.Context, op googleOperation, in googleInput) (object, error) {
	if op.confirm && !in.Confirm {
		return nil, errors.New("this action requires explicit user intent and confirm=true")
	}
	path, err := renderPath(op.operation, input{ListID: in.ListID, TaskID: in.TaskID})
	if err != nil {
		return nil, err
	}
	if op.body && len(in.Body) == 0 {
		return nil, errors.New("body is required")
	}
	if !op.body && len(in.Body) != 0 {
		return nil, errors.New("this endpoint requires an empty body")
	}
	q := url.Values{}
	for k, v := range in.Query {
		if !slices.Contains(strings.Fields(op.query), k) {
			return nil, fmt.Errorf("unsupported query parameter for this endpoint: %s", k)
		}
		q.Set(k, v)
	}
	var b []byte
	if op.body {
		b, err = json.Marshal(in.Body)
		if err != nil {
			return nil, err
		}
	}
	token, err := g.token(ctx)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequestWithContext(ctx, op.Method, g.base+path+"?"+q.Encode(), bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("Content-Type", "application/json")
	resp, err := g.http.Do(r)
	if err != nil {
		return nil, errors.New("Google Tasks transport failed; write outcome may be unknown, inspect before retrying")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Only recognized reason codes become diagnostics. Provider messages can
		// contain account, project or task data and must not be copied to logs.
		var failure struct {
			Error struct {
				Errors, Details []struct{ Reason string }
			}
		}
		if json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&failure) == nil {
			for _, detail := range append(failure.Error.Details, failure.Error.Errors...) {
				switch detail.Reason {
				case "SERVICE_DISABLED", "accessNotConfigured":
					return nil, fmt.Errorf("Google Tasks HTTP %d: %w; use get_google_setup and the Google setup skill to enable Tasks API in this client's project, then check again without a new login", resp.StatusCode, errGoogleAPIDisabled)
				case "ACCESS_TOKEN_SCOPE_INSUFFICIENT", "insufficientPermissions":
					return nil, fmt.Errorf("Google Tasks HTTP %d: %w; explicitly reauthenticate and select Google Tasks access on the consent screen", resp.StatusCode, errGoogleTasksPermission)
				}
			}
		}
		return nil, fmt.Errorf("Google Tasks HTTP %d; retry-after=%s; response body withheld", resp.StatusCode, resp.Header.Get("Retry-After"))
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 40*1024*1024+1))
	if err != nil {
		return nil, errors.New("Google Tasks response interrupted; inspect before retrying")
	}
	if len(data) > 40*1024*1024 {
		return nil, errors.New("Google Tasks response exceeds 40 MiB")
	}
	out := object{}
	if len(data) > 0 && json.Unmarshal(data, &out) != nil {
		return nil, errors.New("Google Tasks returned invalid JSON")
	}
	out["httpStatus"] = resp.StatusCode
	return out, nil
}

func (c *connections) callGoogle(ctx context.Context, name string, in googleInput) (object, error) {
	for _, op := range googleOperations() {
		if name == op.Name {
			return c.withGoogle(ctx, in.ConnectionID, func(a *googleAuth) (object, error) { return newGoogleTasks(a.token).execute(ctx, op, in) })
		}
	}
	return nil, errors.New("unknown Google tool")
}

func addGoogleTools(s *mcp.Server, call func(context.Context, string, googleInput) (object, error)) {
	for _, op := range googleOperations() {
		destructive := op.confirm || op.Method == http.MethodPatch
		queryProperties := object{}
		for key := range strings.FieldsSeq(op.query) {
			queryProperties[key] = object{"type": "string"}
		}
		var query object
		if len(queryProperties) > 0 {
			query = object{"type": "object", "properties": queryProperties, "additionalProperties": false}
		}
		body := ""
		if op.body {
			body = "Google Tasks JSON: title, notes, status (completed/needsAction), due (date only)."
		}
		mcp.AddTool(s, &mcp.Tool{InputSchema: operationInputSchema(op.operation, query, body, false, op.confirm), Name: googleToolPrefix + op.Name, Description: op.Description, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: op.Method == http.MethodGet, DestructiveHint: &destructive, IdempotentHint: op.Method != http.MethodPost}}, func(ctx context.Context, _ *mcp.CallToolRequest, in googleInput) (*mcp.CallToolResult, object, error) {
			return result(call(ctx, op.Name, in))
		})
	}
}
