package app

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestFocusedProviderToolSchemas(t *testing.T) {
	s := mcp.NewServer(&mcp.Implementation{Name: "focused-tools", Version: "1"}, nil)
	calls := 0
	addMicrosoftTools(s, func(context.Context, string, connectedInput) (object, error) {
		calls++
		return object{}, nil
	})
	addGoogleTools(s, func(context.Context, string, googleInput) (object, error) {
		calls++
		return object{}, nil
	})
	st, ct := mcp.NewInMemoryTransports()
	ss, err := s.Connect(t.Context(), st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil).Connect(t.Context(), ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	listed, err := cs.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	// Exact surface, not merely a tool count: removed tools must not return.
	expected := map[string]string{
		"microsoft_list_lists":       "connection_id cursor query",
		"microsoft_create_list":      "body connection_id",
		"microsoft_update_list":      "body connection_id list_id",
		"microsoft_delete_list":      "confirm connection_id list_id",
		"microsoft_list_tasks":       "connection_id cursor list_id query",
		"microsoft_create_task":      "body connection_id list_id",
		"microsoft_update_task":      "body connection_id list_id task_id",
		"microsoft_delete_task":      "confirm connection_id list_id task_id",
		"microsoft_list_checklists":  "connection_id cursor list_id query task_id",
		"microsoft_create_checklist": "body connection_id list_id task_id",
		"microsoft_update_checklist": "body connection_id item_id list_id task_id",
		"microsoft_delete_checklist": "confirm connection_id item_id list_id task_id",
		"google_list_lists":          "connection_id query",
		"google_create_list":         "body connection_id query",
		"google_patch_list":          "body connection_id list_id query",
		"google_delete_list":         "confirm connection_id list_id",
		"google_list_tasks":          "connection_id list_id query",
		"google_create_task":         "body connection_id list_id query",
		"google_patch_task":          "body connection_id list_id query task_id",
		"google_delete_task":         "confirm connection_id list_id task_id",
		"google_move_task":           "confirm connection_id list_id query task_id",
	}
	if len(listed.Tools) != len(expected) {
		t.Fatalf("provider tools: %d", len(listed.Tools))
	}
	for _, tool := range listed.Tools {
		t.Run(tool.Name, func(t *testing.T) {
			schema := tool.InputSchema.(map[string]any)
			properties := schema["properties"].(map[string]any)
			want, ok := expected[tool.Name]
			if !ok || strings.Join(slices.Sorted(maps.Keys(properties)), " ") != want {
				t.Fatalf("unexpected fields: %v", properties)
			}
			args := object{}
			for _, field := range schema["required"].([]any) {
				name := field.(string)
				switch name {
				case "body":
					args[name] = object{"title": "test"}
				case "confirm":
					args[name] = true
				default:
					args[name] = "test-id"
				}
			}
			// Parent/position remain usable for Google subtasks and reordering.
			if tool.Name == "google_create_task" || tool.Name == "google_move_task" {
				args["query"] = object{"parent": "parent-id", "previous": "sibling-id"}
			}
			before := calls
			r, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: tool.Name, Arguments: args})
			if err != nil || r.IsError || calls != before+1 {
				t.Fatalf("valid request: %v %v", r, err)
			}
			for _, field := range schema["required"].([]any) {
				bad := maps.Clone(args)
				delete(bad, field.(string))
				r, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: tool.Name, Arguments: bad})
				if err != nil || !r.IsError || calls != before+1 {
					t.Fatalf("accepted missing %s: %v %v", field, r, err)
				}
			}
			args["session_url"] = "https://example.invalid"
			r, err = cs.CallTool(t.Context(), &mcp.CallToolParams{Name: tool.Name, Arguments: args})
			if err != nil || !r.IsError || calls != before+1 {
				t.Fatalf("accepted unused upload field: %v %v", r, err)
			}
		})
	}
	data, err := json.Marshal(listed)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("21 provider tools: %d JSON bytes", len(data))
}
