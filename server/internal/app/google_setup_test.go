package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/oauth2"
)

func TestGoogleSetupFromEmptyToImported(t *testing.T) {
	useTestKey(t)
	c := &connections{dir: filepath.Join(t.TempDir(), "data")}
	out, err := c.googleSetup(t.Context(), googleSetupInput{})
	if err != nil || out["state"] != "client_json_required" {
		t.Fatalf("empty setup: %v %v", out, err)
	}
	if _, err := os.Stat(c.dir); !os.IsNotExist(err) {
		t.Fatal("setup inspection created storage")
	}
	missing := filepath.Join(t.TempDir(), "not-downloaded.json")
	out, err = c.googleSetup(t.Context(), googleSetupInput{CredentialsPath: missing})
	if err != nil || out["state"] != "client_json_required" {
		t.Fatalf("missing download: %v %v", out, err)
	}
	path := filepath.Join(t.TempDir(), "client.json")
	raw := []byte(`{"installed":{"client_id":"test-id","client_secret":"secret-not-for-output","project_id":"test-project"}}`)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	out, err = c.googleSetup(t.Context(), googleSetupInput{CredentialsPath: path})
	if err != nil || out["state"] != "import_required" || out["project_id"] != "test-project" {
		t.Fatalf("inspect client: %v %v", out, err)
	}
	if strings.Contains(fmt.Sprint(out), "secret-not-for-output") {
		t.Fatal("secret disclosed")
	}
	addTestConnection(t, c, "ms")
	in := googleImportInput{ConnectionID: "google", Label: "Google", CredentialsPath: path}
	if _, err := c.importGoogle(in); err == nil {
		t.Fatal("import without user intent")
	}
	in.Confirm = true
	if _, err := c.importGoogle(in); err != nil {
		t.Fatal(err)
	}
	if _, err := c.importGoogle(in); err == nil {
		t.Fatal("existing connection overwritten")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(after, raw) {
		t.Fatal("original JSON changed")
	}
	stored, _ := os.ReadFile(c.path("google", ".client"))
	if bytes.Contains(stored, []byte("secret-not-for-output")) {
		t.Fatal("client stored in plaintext")
	}
	restarted := &connections{dir: c.dir}
	out, err = restarted.googleSetup(t.Context(), googleSetupInput{ConnectionID: "google"})
	if err != nil || out["state"] != "login_required" || out["project_id"] != "test-project" {
		t.Fatalf("restart: %v %v", out, err)
	}
	out, err = restarted.googleSetup(t.Context(), googleSetupInput{})
	if err != nil || out["state"] != "select_connection" {
		t.Fatalf("existing not reused: %v %v", out, err)
	}
	if _, err := restarted.loginGoogle(t.Context(), googleLoginInput{ConnectionID: "google"}); err == nil {
		t.Fatal("login without consent")
	}
	if _, err := restarted.loginGoogle(t.Context(), googleLoginInput{ConnectionID: "ms", Confirm: true}); err == nil {
		t.Fatal("wrong provider opened browser")
	}
	if _, err := restarted.load("ms"); err != nil {
		t.Fatal("Google setup modified Microsoft", err)
	}
}

func TestGoogleSetupLegacyProjectInspection(t *testing.T) {
	useTestKey(t)
	c := &connections{dir: t.TempDir()}
	if err := c.add(connection{ID: "old", Provider: googleProvider, Label: "Old"}, &googleClient{ClientID: "old-id", ClientSecret: "fake-secret"}); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(c.path("old", ".client"))
	path := filepath.Join(t.TempDir(), "source.json")
	if err := os.WriteFile(path, []byte(`{"installed":{"client_id":"old-id","client_secret":"fake-secret","project_id":"original-project"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	out, err := c.googleSetup(t.Context(), googleSetupInput{ConnectionID: "old", CredentialsPath: path})
	if err != nil || out["project_id"] != "original-project" {
		t.Fatalf("legacy inspection: %v %v", out, err)
	}
	after, _ := os.ReadFile(c.path("old", ".client"))
	if !bytes.Equal(before, after) {
		t.Fatal("inspection replaced saved client")
	}
	if err := os.WriteFile(path, []byte(`{"installed":{"client_id":"different-id","client_secret":"fake","project_id":"wrong-project"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := c.googleSetup(t.Context(), googleSetupInput{ConnectionID: "old", CredentialsPath: path}); err == nil {
		t.Fatal("mismatched project accepted")
	}
}

func TestGoogleSetupAPIEnablementResumesWithoutLogin(t *testing.T) {
	useTestKey(t)
	a := newGoogleAuth(googleClient{ClientID: "fake-id", ClientSecret: "fake-secret", ProjectID: "own-project"}, filepath.Join(t.TempDir(), "token.secret"))
	if err := a.save(googleSession{Subject: "same-account", Token: &oauth2.Token{AccessToken: "private-token", Expiry: time.Now().Add(time.Hour)}}); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(a.path)
	status, body, calls := http.StatusForbidden, `{"error":{"details":[{"reason":"SERVICE_DISABLED"}]}}`, 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodGet || r.URL.Path != googleListsPath || r.URL.Query().Get("maxResults") != "1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}))
	defer srv.Close()
	g := newGoogleTasks(a.token)
	g.base = srv.URL
	out, err := googleSetupAccess(t.Context(), a, g, false)
	if err != nil || out["state"] != "verification_required" || calls != 0 {
		t.Fatalf("inspection contacted API: %v %v", out, err)
	}
	for _, tc := range []struct {
		status      int
		body, state string
	}{
		{http.StatusForbidden, body, "api_enable_required"},
		{http.StatusOK, `{"items":[]}`, "ready"},
		{http.StatusForbidden, `{"error":{"details":[{"reason":"ACCESS_TOKEN_SCOPE_INSUFFICIENT"}]}}`, "consent_required"},
		{http.StatusForbidden, `{"error":{"message":"private-token"}}`, "check_failed"},
		{http.StatusTooManyRequests, `{"error":{"message":"private-token"}}`, "check_failed"},
	} {
		status, body = tc.status, tc.body
		out, err = googleSetupAccess(t.Context(), a, g, true)
		if err != nil || out["state"] != tc.state {
			t.Fatalf("%s: %v %v", tc.state, out, err)
		}
		if !strings.HasSuffix(out["console"].(object)["tasks_api"].(string), "?project=own-project") {
			t.Fatal("wrong project activation URL")
		}
		if strings.Contains(fmt.Sprint(out), "private-token") {
			t.Fatal("token disclosed")
		}
	}
	if calls != 5 {
		t.Fatal("unexpected retry or request", calls)
	}
	after, _ := os.ReadFile(a.path)
	if !bytes.Equal(before, after) {
		t.Fatal("activation check rewrote valid credentials")
	}
}

func TestGoogleSetupMCP(t *testing.T) {
	useTestKey(t)
	c := &connections{dir: t.TempDir()}
	st, ct := mcp.NewInMemoryTransports()
	ss, err := server(c).Connect(t.Context(), st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "setup-test", Version: "1"}, nil).Connect(t.Context(), ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	call := func(name string, args object, want string) {
		t.Helper()
		r, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil || r.IsError {
			t.Fatalf("tool %s: %v %v", name, r, err)
		}
		b, err := json.Marshal(r.StructuredContent)
		var out object
		if err != nil || json.Unmarshal(b, &out) != nil || out["state"] != want {
			t.Fatalf("tool result: %s %v", b, err)
		}
	}
	call(googleSetupTool, object{}, "client_json_required")
	path := filepath.Join(t.TempDir(), "download.json")
	if err := os.WriteFile(path, []byte(`{"installed":{"client_id":"mcp-fake","client_secret":"mcp-secret","project_id":"mcp-project"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	call(googleSetupTool, object{"credentials_path": path}, "import_required")
	call(googleImportTool, object{"connection_id": "test", "label": "Test", "credentials_path": path, "confirm": true}, "login_required")
	call(googleSetupTool, object{"connection_id": "test"}, "login_required")
}
