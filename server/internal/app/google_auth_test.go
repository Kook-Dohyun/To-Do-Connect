package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestGoogleBrowserLoginRefreshAndIdentity(t *testing.T) {
	useTestKey(t)
	const access = "fake-google-access"
	const refresh = "fake-google-refresh"
	subject := "account-one"
	refreshCalls := 0
	var authQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/userinfo" {
			if r.Header.Get("Authorization") != "Bearer "+access {
				t.Error("userinfo token")
			}
			fmt.Fprintf(w, `{"sub":%q}`, subject)
			return
		}
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		if r.Form.Get("client_id") != "desktop-id" || r.Form.Get("client_secret") != "fake-client-secret" {
			t.Error("wrong client")
		}
		switch r.Form.Get("grant_type") {
		case "authorization_code":
			if r.Form.Get("code") != "fake-code" || r.Form.Get("redirect_uri") != authQuery.Get("redirect_uri") {
				t.Error("bad exchange")
			}
			hash := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
			if base64.RawURLEncoding.EncodeToString(hash[:]) != authQuery.Get("code_challenge") {
				t.Error("PKCE verifier mismatch")
			}
			fmt.Fprintf(w, `{"access_token":%q,"refresh_token":%q,"token_type":"Bearer","expires_in":3600}`, access, refresh)
		case "refresh_token":
			refreshCalls++
			if r.Form.Get("refresh_token") != refresh {
				t.Error("refresh token lost")
			}
			// Google may omit refresh_token on refresh; oauth2 must preserve it.
			fmt.Fprintf(w, `{"access_token":%q,"token_type":"Bearer","expires_in":3600}`, access)
		default:
			t.Error("unknown grant")
		}
	}))
	defer srv.Close()
	a := newGoogleAuth(googleClient{ClientID: "desktop-id", ClientSecret: "fake-client-secret"}, filepath.Join(t.TempDir(), "google.secret"))
	a.config.Endpoint.TokenURL = srv.URL + "/token"
	a.userInfoURL = srv.URL + "/userinfo"
	open := func(raw string) error {
		u, err := url.Parse(raw)
		if err != nil {
			return err
		}
		authQuery = u.Query()
		if u.Scheme != "https" || u.Host != "accounts.google.com" || authQuery.Get("code_challenge_method") != "S256" || authQuery.Get("access_type") != "offline" || !strings.Contains(authQuery.Get("scope"), googleTasksScope) {
			t.Error("incorrect authorize request")
		}
		cb, err := url.Parse(authQuery.Get("redirect_uri"))
		if err != nil {
			return err
		}
		if cb.Hostname() != "127.0.0.1" {
			t.Error("non-loopback callback")
		}
		for _, valid := range []bool{false, true} {
			q := url.Values{"code": {"fake-code"}, "state": {"wrong-state"}}
			if valid {
				q.Set("state", authQuery.Get("state"))
			}
			cb.RawQuery = q.Encode()
			resp, err := http.Get(cb.String())
			if err != nil {
				return err
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if !valid && resp.StatusCode != http.StatusBadRequest {
				t.Error("wrong state accepted")
			}
			if valid && (resp.StatusCode != 200 || bytes.Contains(body, []byte("fake-code")) || resp.Header.Get("Cache-Control") != "no-store") {
				t.Error("callback unsafe or failed")
			}
		}
		return nil
	}
	if err := a.login(t.Context(), false, open); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(a.path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(before, []byte(access)) || bytes.Contains(before, []byte(refresh)) {
		t.Fatal("plaintext tokens")
	}
	if err := a.login(t.Context(), false, func(string) error { t.Fatal("login repeated"); return nil }); err == nil {
		t.Fatal("existing login not reused")
	}
	subject = "account-two"
	if err := a.login(t.Context(), true, open); err == nil {
		t.Fatal("reauth silently switched account")
	}
	after, _ := os.ReadFile(a.path)
	if !bytes.Equal(before, after) {
		t.Fatal("wrong-account attempt changed credentials")
	}
	s, err := a.load()
	if err != nil {
		t.Fatal(err)
	}
	s.Token.Expiry = time.Now().Add(-time.Hour)
	if err := a.save(s); err != nil {
		t.Fatal(err)
	}
	restarted := newGoogleAuth(googleClient{ClientID: "desktop-id", ClientSecret: "fake-client-secret"}, a.path)
	restarted.config.Endpoint.TokenURL = srv.URL + "/token"
	tok, err := restarted.token(t.Context())
	if err != nil || tok != access || refreshCalls != 1 {
		t.Fatalf("refresh %v count=%d", err, refreshCalls)
	}
	s, err = restarted.load()
	if err != nil || s.Token.RefreshToken != refresh || s.Subject != "account-one" {
		t.Fatal("refresh persistence failed", err)
	}
	if _, err := restarted.token(t.Context()); err != nil || refreshCalls != 1 {
		t.Fatal("unnecessary refresh", err)
	}
}

func TestGoogleLoginCancelAndFailurePreserveState(t *testing.T) {
	useTestKey(t)
	a := newGoogleAuth(googleClient{ClientID: "id", ClientSecret: "test"}, filepath.Join(t.TempDir(), "google.secret"))
	if err := a.login(t.Context(), false, func(string) error { return errors.New("browser cannot open secret-string") }); err == nil || strings.Contains(err.Error(), "secret-string") {
		t.Fatal("browser failure leaked", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	if err := a.login(ctx, false, func(string) error { cancel(); return nil }); err == nil {
		t.Fatal("canceled login accepted")
	}
	if _, err := os.Stat(a.path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("failed login persisted credentials")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		fmt.Fprint(w, `{"error":"invalid_grant","error_description":"secret-string"}`)
	}))
	defer srv.Close()
	a.config.Endpoint.TokenURL = srv.URL
	if err := a.save(googleSession{Subject: "one", Token: &oauth2.Token{AccessToken: "old", RefreshToken: "old-refresh", Expiry: time.Now().Add(-time.Hour)}}); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(a.path)
	if _, err := a.token(t.Context()); err == nil || strings.Contains(err.Error(), "secret-string") {
		t.Fatal("refresh failure unsafe", err)
	}
	after, _ := os.ReadFile(a.path)
	if !bytes.Equal(before, after) {
		t.Fatal("failed refresh erased old credentials")
	}
}

func TestGoogleAndMicrosoftConnectionsCoexist(t *testing.T) {
	useTestKey(t)
	c := &connections{dir: t.TempDir()}
	addTestConnection(t, c, "microsoft-personal")
	for _, id := range []string{"google-personal", "google-work"} {
		if err := c.add(connection{ID: id, Provider: googleProvider, Label: id}, &googleClient{ClientID: "client-" + id, ClientSecret: "secret-" + id}); err != nil {
			t.Fatal(err)
		}
		_, err := c.withGoogle(t.Context(), id, func(a *googleAuth) (object, error) {
			return nil, a.save(googleSession{Subject: id, Token: &oauth2.Token{AccessToken: "access-" + id, RefreshToken: "refresh-" + id, Expiry: time.Now().Add(time.Hour)}})
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		restarted := &connections{dir: c.dir}
		for _, id := range []string{"google-personal", "google-work"} {
			_, err := restarted.withGoogle(t.Context(), id, func(a *googleAuth) (object, error) {
				token, err := a.token(t.Context())
				if err != nil || token != "access-"+id || a.config.ClientID != "client-"+id {
					t.Fatal("account mixing", err)
				}
				return nil, nil
			})
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := c.callGoogle(t.Context(), "list_lists", googleInput{ConnectionID: "microsoft-personal"}); err == nil {
		t.Fatal("Microsoft used as Google")
	}
	if _, err := c.callMicrosoft(t.Context(), "list_lists", connectedInput{ConnectionID: "google-personal"}); err == nil {
		t.Fatal("Google used as Microsoft")
	}
	for _, suffix := range []string{".json", ".client", ".secret"} {
		data, err := os.ReadFile(c.path("google-personal", suffix))
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte("secret-google-personal")) || bytes.Contains(data, []byte("access-google-personal")) {
			t.Fatal("credential exposed on disk")
		}
	}
	if err := c.remove("google-personal", true); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{".json", ".client", ".secret"} {
		if _, err := os.Stat(c.path("google-personal", suffix)); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("disconnect retained credential", suffix)
		}
	}
	listed, err := c.list()
	if err != nil || len(listed["connections"].([]object)) != 2 {
		t.Fatalf("unrelated connection lost %v %v", listed, err)
	}
}

func TestGoogleClientImport(t *testing.T) {
	useTestKey(t)
	path := filepath.Join(t.TempDir(), "desktop.json")
	for _, src := range []string{`{"web":{"client_id":"id","client_secret":"secret"}}`, `{"installed":{"client_id":"id"}}`, `not json secret-string`} {
		if err := os.WriteFile(path, []byte(src), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := readGoogleClient(path); err == nil || strings.Contains(err.Error(), "secret-string") {
			t.Fatal("bad config accepted or logged", err)
		}
	}
	if err := os.WriteFile(path, []byte(`{"installed":{"client_id":"id","client_secret":"secret","auth_uri":"https://malicious.invalid","token_uri":"https://malicious.invalid"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	client, err := readGoogleClient(path)
	if err != nil {
		t.Fatal(err)
	}
	a := newGoogleAuth(*client, "")
	if a.config.Endpoint.AuthURL != googleAuthURL || a.config.Endpoint.TokenURL != googleTokenURL {
		t.Fatal("untrusted OAuth endpoints used")
	}
}
