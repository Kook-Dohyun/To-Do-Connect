package app

import (
	"bytes"
	"fmt"
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

func TestGoogleAuthorizationFailuresDoNotReplaceCredentials(t *testing.T) {
	useTestKey(t)
	for _, mode := range []string{"declined", "exchange-error", "no-refresh", "missing-tasks-scope", "identity-error", "token-redirect"} {
		t.Run(mode, func(t *testing.T) {
			foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("OAuth credentials followed a redirect") }))
			defer foreign.Close()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/userinfo" {
					w.WriteHeader(http.StatusForbidden)
					fmt.Fprint(w, `{"error":"sensitive-body"}`)
					return
				}
				switch mode {
				case "token-redirect":
					http.Redirect(w, r, foreign.URL, http.StatusTemporaryRedirect)
				case "exchange-error":
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprint(w, `{"error":"invalid_grant","error_description":"sensitive-body"}`)
				case "no-refresh":
					fmt.Fprint(w, `{"access_token":"sensitive-body","token_type":"Bearer","expires_in":3600}`)
				case "missing-tasks-scope":
					fmt.Fprint(w, `{"access_token":"sensitive-body","refresh_token":"test-refresh","scope":"openid","token_type":"Bearer","expires_in":3600}`)
				default:
					fmt.Fprint(w, `{"access_token":"sensitive-body","refresh_token":"test-refresh","token_type":"Bearer","expires_in":3600}`)
				}
			}))
			defer srv.Close()
			a := newGoogleAuth(googleClient{ClientID: "test-id", ClientSecret: "test-secret"}, filepath.Join(t.TempDir(), "google.secret"))
			a.config.Endpoint.TokenURL = srv.URL + "/token"
			a.userInfoURL = srv.URL + "/userinfo"
			if err := a.save(googleSession{Subject: "old-account", Token: &oauth2.Token{AccessToken: "old-access", RefreshToken: "old-refresh", Expiry: time.Now().Add(time.Hour)}}); err != nil {
				t.Fatal(err)
			}
			before, _ := os.ReadFile(a.path)
			err := a.login(t.Context(), true, func(raw string) error {
				u, err := url.Parse(raw)
				if err != nil {
					return err
				}
				q := url.Values{"state": {u.Query().Get("state")}, "code": {"fake-code"}}
				if mode == "declined" {
					q.Del("code")
					q.Set("error", "access_denied")
				}
				resp, err := http.Get(u.Query().Get("redirect_uri") + "?" + q.Encode())
				if err == nil {
					resp.Body.Close()
				}
				return err
			})
			if err == nil || strings.Contains(err.Error(), "sensitive-body") {
				t.Fatal("unsafe authorization failure", err)
			}
			after, _ := os.ReadFile(a.path)
			if !bytes.Equal(before, after) {
				t.Fatal("authorization failure replaced previous session")
			}
		})
	}
}
