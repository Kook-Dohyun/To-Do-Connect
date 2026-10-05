package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestGoogleAllRoutes(t *testing.T) {
	if len(googleOperations()) != 14 {
		t.Fatal("API coverage changed")
	}
	for _, op := range googleOperations() {
		t.Run(op.Name, func(t *testing.T) {
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				expected := strings.NewReplacer("{list}", "list one", "{task}", "task?two").Replace(op.Path)
				if r.Method != op.Method || r.URL.Path != "/tasks/v1"+expected {
					t.Errorf("wrong route %s %s", r.Method, r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer fake-google" {
					t.Error("missing token")
				}
				data, _ := io.ReadAll(r.Body)
				if op.body {
					var body object
					if json.Unmarshal(data, &body) != nil || body["title"] != "한글 작업" {
						t.Errorf("body %s", data)
					}
				} else if len(data) != 0 {
					t.Error("unexpected body")
				}
				if op.Name == "move_task" && r.URL.Query().Get("destinationTasklist") != "destination" {
					t.Error("cross-list move missing")
				}
				fmt.Fprint(w, `{"id":"returned","title":"한글 작업","nextPageToken":"page-two"}`)
			}))
			defer srv.Close()
			g := newGoogleTasks(func(context.Context) (string, error) { return "fake-google", nil })
			g.base = srv.URL + "/tasks/v1"
			in := googleInput{ListID: "list one", TaskID: "task?two", Confirm: true}
			if op.body {
				in.Body = object{"title": "한글 작업"}
			}
			if op.Name == "move_task" {
				in.Query = map[string]string{"destinationTasklist": "destination", "parent": "parent", "previous": "before"}
			}
			out, err := g.execute(t.Context(), op, in)
			if err != nil || out["title"] != "한글 작업" || out["nextPageToken"] != "page-two" || calls != 1 {
				t.Fatalf("route result %v %v calls=%d", out, err, calls)
			}
			if op.confirm {
				in.Confirm = false
				if _, err := g.execute(t.Context(), op, in); err == nil {
					t.Error("unconfirmed mutation")
				}
				in.Confirm = true
			}
			in.Query = map[string]string{"access_token": "must-never-send"}
			if _, err := g.execute(t.Context(), op, in); err == nil {
				t.Error("credentials query accepted")
			}
			if calls != 1 {
				t.Error("invalid input reached Google")
			}
		})
	}
}

func TestGooglePaginationRedirectAndErrors(t *testing.T) {
	hits := 0
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++; t.Error("redirect followed") }))
	defer foreign.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("pageToken") {
		case "next":
			fmt.Fprint(w, `{"items":[{"id":"second"}]}`)
		case "redirect":
			http.Redirect(w, r, foreign.URL, http.StatusFound)
		case "quota":
			w.Header().Set("Retry-After", "30")
			w.WriteHeader(429)
			fmt.Fprint(w, `{"error":"secret-must-not-be-printed"}`)
		default:
			fmt.Fprint(w, `{"items":[{"id":"first"}],"nextPageToken":"next"}`)
		}
	}))
	defer srv.Close()
	g := newGoogleTasks(func(context.Context) (string, error) { return "secret-must-not-be-printed", nil })
	g.base = srv.URL
	op := googleOperations()[0]
	r, err := g.execute(t.Context(), op, googleInput{})
	if err != nil {
		t.Fatal(err)
	}
	r, err = g.execute(t.Context(), op, googleInput{Query: map[string]string{"pageToken": r["nextPageToken"].(string)}})
	if err != nil || r["items"].([]any)[0].(map[string]any)["id"] != "second" {
		t.Fatalf("pagination %v %v", r, err)
	}
	for _, cursor := range []string{"redirect", "quota"} {
		_, err = g.execute(t.Context(), op, googleInput{Query: map[string]string{"pageToken": cursor}})
		if err == nil || strings.Contains(err.Error(), "secret-must-not-be-printed") {
			t.Fatalf("unsafe failure %v", err)
		}
	}
	if hits != 0 {
		t.Fatal("external resource contacted")
	}
}

func TestGoogleMCPTools(t *testing.T) {
	s := mcp.NewServer(&mcp.Implementation{Name: "google-test", Version: "1"}, nil)
	called := false
	addGoogleTools(s, func(_ context.Context, name string, in googleInput) (object, error) {
		called = true
		if name != "create_task" || in.ConnectionID != "google-work" || in.Body["notes"] != "메모" || in.ListID != "list" {
			t.Errorf("schema lost fields: %s %#v", name, in)
		}
		return object{"id": "new-task"}, nil
	})
	st, ct := mcp.NewInMemoryTransports()
	ss, err := s.Connect(t.Context(), st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	cs, err := client.Connect(t.Context(), ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	tools, err := cs.ListTools(t.Context(), nil)
	if err != nil || len(tools.Tools) != 14 {
		t.Fatalf("tools %v %v", tools, err)
	}
	r, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: googleToolPrefix + "create_task", Arguments: object{"list_id": "list", "body": object{"notes": "메모"}}})
	if err != nil || !r.IsError || called {
		t.Fatalf("connection ID not required %v %v", r, err)
	}
	r, err = cs.CallTool(t.Context(), &mcp.CallToolParams{Name: googleToolPrefix + "create_task", Arguments: object{"connection_id": "google-work", "list_id": "list", "body": object{"notes": "메모"}}})
	if err != nil || r.IsError || !called {
		t.Fatalf("MCP call %v %v", r, err)
	}
}

func TestGoogleSafeSetupDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
	}{
		{"disabled", `{"error":{"message":"private-project-and-token","details":[{"reason":"SERVICE_DISABLED","metadata":{"consumer":"private-project"}}]}}`, "API_DISABLED"},
		{"legacy-disabled", `{"error":{"errors":[{"reason":"accessNotConfigured","message":"private-project-and-token"}]}}`, "API_DISABLED"},
		{"scope", `{"error":{"details":[{"reason":"ACCESS_TOKEN_SCOPE_INSUFFICIENT"}]}}`, "TASKS_PERMISSION_MISSING"},
		{"legacy-scope", `{"error":{"errors":[{"reason":"insufficientPermissions"}]}}`, "TASKS_PERMISSION_MISSING"},
		{"unknown", `{"error":{"message":"private-project-and-token","details":[{"reason":"private-project-and-token"}]}}`, "response body withheld"},
		{"invalid", `private-project-and-token`, "response body withheld"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprint(w, tc.body)
			}))
			defer srv.Close()
			g := newGoogleTasks(func(context.Context) (string, error) { return "private-project-and-token", nil })
			g.base = srv.URL
			_, err := g.execute(t.Context(), googleOperations()[0], googleInput{})
			if err == nil || !strings.Contains(err.Error(), tc.want) || strings.Contains(err.Error(), "private-project") {
				t.Fatalf("unexpected diagnostic: %v", err)
			}
		})
	}
}
