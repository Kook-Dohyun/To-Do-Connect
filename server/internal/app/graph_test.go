package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func testGraph(t *testing.T, h http.HandlerFunc) *graph {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	g := newGraph(func(context.Context) (string, error) { return "test-token", nil })
	g.base = srv.URL + "/v1.0"
	return g
}
func TestEveryOperationRoute(t *testing.T) {
	if len(operations()) != 12 {
		t.Fatalf("coverage changed: %d operations", len(operations()))
	}
	for _, op := range operations() {
		t.Run(op.Name, func(t *testing.T) {
			called := false
			in := input{ListID: "list=", TaskID: "task=", ItemID: "item=", Body: object{"title": "한글 작업"}, Confirm: true}
			expected, e := renderPath(op, in)
			if e != nil {
				t.Fatal(e)
			}
			g := testGraph(t, func(w http.ResponseWriter, r *http.Request) {
				called = true
				if r.Method != op.Method || r.URL.EscapedPath() != "/v1.0"+expected {
					t.Errorf("wrong route %s %s", r.Method, r.URL.EscapedPath())
				}
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Error("missing auth")
				}
				if op.Method == http.MethodDelete {
					w.WriteHeader(204)
					return
				}
				if strings.HasSuffix(expected, "/$value") {
					w.Write([]byte("한국어 첨부"))
					return
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"id":"test","value":[]}`)
			})
			if _, e = g.execute(t.Context(), op, in); e != nil {
				t.Fatal(e)
			}
			if !called {
				t.Fatal("not called")
			}
		})
	}
}
func TestBoundaries(t *testing.T) {
	calls := 0
	g := testGraph(t, func(w http.ResponseWriter, r *http.Request) { calls++; fmt.Fprint(w, `{"value":[]}`) })
	for _, path := range []string{"https://evil.example/v1.0/me/todo/lists", g.base + "/me/messages", g.base + "/me/todo/listsEvil", g.base + "/me/todo/lists/../messages", "/me/todo/lists/../../messages", g.base + "/me/todo/lists#x"} {
		if _, e := g.json(t.Context(), "GET", path, nil); e == nil {
			t.Errorf("allowed %s", path)
		}
	}
	for _, op := range operations() {
		if op.Method == http.MethodDelete {
			if _, e := g.execute(t.Context(), op, input{ListID: "l", TaskID: "t", ItemID: "i"}); e == nil {
				t.Error("unconfirmed deletion")
			}
		}
	}
	if calls != 0 {
		t.Fatalf("unsafe request made %d calls", calls)
	}
	for _, s := range []string{"", "..", "a/b", "a\\b"} {
		if _, e := segment(s); e == nil {
			t.Errorf("unsafe ID: %s", s)
		}
	}
}
func TestCursorAndPagination(t *testing.T) {
	var g *graph
	calls := 0
	g = testGraph(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("$skiptoken") == "next" {
			fmt.Fprint(w, `{"value":[{"id":"b"}],"@odata.deltaLink":"done"}`)
		} else {
			json.NewEncoder(w).Encode(object{"value": []object{{"id": "a"}}, "@odata.nextLink": g.base + listsPath + "?$skiptoken=next"})
		}
	})
	v, e := g.all(t.Context(), listsPath)
	if e != nil || len(v) != 2 || calls != 2 {
		t.Fatalf("pagination: %v %v %d", v, e, calls)
	}
	op := operations()[0]
	_, e = g.execute(t.Context(), op, input{Cursor: g.base + listsPath + "?$skiptoken=next"})
	if e != nil {
		t.Fatal(e)
	}
	_, e = g.execute(t.Context(), op, input{Cursor: g.base + listsPath + "/other/tasks?$skiptoken=next"})
	if e == nil {
		t.Fatal("cross-collection cursor allowed")
	}
}
func TestErrorsAndRedirectDoNotLeak(t *testing.T) {
	g := testGraph(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(429)
		fmt.Fprint(w, `{"error":{"code":"TooManyRequests","message":"private contents and password"}}`)
	})
	_, e := g.json(t.Context(), "GET", listsPath, nil)
	if e == nil || strings.Contains(e.Error(), "password") || !strings.Contains(e.Error(), "retry-after=5") {
		t.Fatalf("error: %v", e)
	}
	reached := false
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true }))
	defer other.Close()
	g = testGraph(t, func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, other.URL, 302) })
	if _, e = g.json(t.Context(), "GET", listsPath, nil); e == nil || reached {
		t.Fatal("redirect followed")
	}
}
func TestMCPToolsAndCalls(t *testing.T) {
	g := testGraph(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"value":[{"id":"l","displayName":"업무"}]}`)
	})
	st, ct := mcp.NewInMemoryTransports()
	s := mcp.NewServer(&mcp.Implementation{Name: "microsoft-tools-test", Version: "1"}, nil)
	addMicrosoftTools(s, func(ctx context.Context, name string, in connectedInput) (object, error) {
		return g.call(ctx, name, in.input)
	})
	ss, e := s.Connect(t.Context(), st, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer ss.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	cs, e := client.Connect(t.Context(), ct, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer cs.Close()
	listed, e := cs.ListTools(t.Context(), nil)
	if e != nil || len(listed.Tools) != 12 {
		t.Fatalf("tools %v %v", listed, e)
	}
	for _, tool := range listed.Tools {
		if strings.HasPrefix(tool.Name, microsoftToolPrefix+"delete_") || tool.Name == microsoftToolPrefix+"move_task" {
			if tool.Annotations.ReadOnlyHint || tool.Annotations.DestructiveHint == nil || !*tool.Annotations.DestructiveHint {
				t.Errorf("wrong annotations %s", tool.Name)
			}
		}
	}
	r, e := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: microsoftToolPrefix + "list_lists", Arguments: object{"connection_id": "test"}})
	if e != nil || r.IsError {
		t.Fatalf("MCP call %v %v", r, e)
	}
	r, e = cs.CallTool(t.Context(), &mcp.CallToolParams{Name: microsoftToolPrefix + "delete_task", Arguments: object{"connection_id": "test", "list_id": "l", "task_id": "t"}})
	if e != nil || !r.IsError {
		t.Fatalf("expected tool error %v %v", r, e)
	}
}
