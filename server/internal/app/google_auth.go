package app

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

const googleProvider = "google"
const googleTasksScope = "https://www.googleapis.com/auth/tasks"
const googleAuthURL = "https://accounts.google.com/o/oauth2/v2/auth"
const googleTokenURL = "https://oauth2.googleapis.com/token"
const googleUserInfoURL = "https://openidconnect.googleapis.com/v1/userinfo"

type googleClient struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	ProjectID    string `json:"project_id,omitempty"`
}

func readGoogleClient(path string) (*googleClient, error) {
	var source io.Reader = os.Stdin
	if path != "-" {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		source = f
	}
	var document struct {
		Installed *googleClient `json:"installed"`
	}
	if err := json.NewDecoder(io.LimitReader(source, 1024*1024)).Decode(&document); err != nil {
		return nil, errors.New("invalid Google desktop-client JSON; contents were not logged")
	}
	if document.Installed == nil || document.Installed.ClientID == "" || document.Installed.ClientSecret == "" {
		return nil, errors.New("download an OAuth Desktop app client JSON containing installed.client_id and installed.client_secret; web clients and API keys are not supported")
	}
	// Endpoints from the imported file are not used; only Google's fixed endpoints are trusted.
	return document.Installed, nil
}

type googleSession struct {
	Token   *oauth2.Token `json:"token"`
	Subject string        `json:"subject"`
}

type googleAuth struct {
	projectID     string
	config        oauth2.Config
	path          string
	http          *http.Client
	userInfoURL   string
	listenAddress string
}

func newGoogleAuth(client googleClient, path string) *googleAuth {
	return &googleAuth{
		projectID: client.ProjectID,
		config:    oauth2.Config{ClientID: client.ClientID, ClientSecret: client.ClientSecret, Scopes: []string{googleTasksScope, "openid"}, Endpoint: oauth2.Endpoint{AuthURL: googleAuthURL, TokenURL: googleTokenURL, AuthStyle: oauth2.AuthStyleInParams}},
		path:      path, userInfoURL: googleUserInfoURL, listenAddress: "127.0.0.1:0",
		http: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	}
}

func (a *googleAuth) load() (googleSession, error) {
	var s googleSession
	b, err := os.ReadFile(a.path)
	if err != nil {
		return s, err
	}
	b, err = unprotect(b)
	if err != nil {
		return s, errors.New("could not decrypt Google credentials")
	}
	if json.Unmarshal(b, &s) != nil || s.Token == nil || s.Subject == "" {
		return s, errors.New("invalid Google credential cache")
	}
	return s, nil
}

func (a *googleAuth) save(s googleSession) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	b, err = protect(b)
	if err != nil {
		return err
	}
	return writePrivate(a.path, b)
}

// The connection's OS lock spans token refresh, persistence and the API request.
func (a *googleAuth) token(ctx context.Context) (string, error) {
	s, err := a.load()
	if errors.Is(err, os.ErrNotExist) {
		return "", errors.New("Google connection is not signed in; run todo-connect login CONNECTION_ID locally")
	}
	if err != nil {
		return "", err
	}
	if s.Token.Valid() {
		return s.Token.AccessToken, nil
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, a.http)
	token, err := a.config.TokenSource(ctx, s.Token).Token()
	if err != nil {
		return "", errors.New("Google token refresh failed; check connectivity or explicitly reauthenticate if consent expired; provider response withheld to protect credentials")
	}
	s.Token = token
	if err := a.save(s); err != nil {
		return "", err
	}
	return token.AccessToken, nil
}

func (a *googleAuth) subject(ctx context.Context, token *oauth2.Token) (string, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, a.userInfoURL, nil)
	if err != nil {
		return "", err
	}
	token.SetAuthHeader(r)
	resp, err := a.http.Do(r)
	if err != nil {
		return "", errors.New("Google account identity request failed")
	}
	defer resp.Body.Close()
	var identity struct {
		Subject string `json:"sub"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&identity) != nil || identity.Subject == "" {
		return "", errors.New("could not verify Google account identity; existing credentials were not changed")
	}
	return identity.Subject, nil
}

func (a *googleAuth) login(ctx context.Context, reauthenticate bool, openURL func(string) error) error {
	if _, err := protect([]byte("credential-storage-check")); err != nil {
		return err
	}
	previous, err := a.load()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err == nil && !reauthenticate {
		return errors.New("this connection already has credentials; check it first, or explicitly request --reauthenticate")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	listener, err := net.Listen("tcp4", a.listenAddress)
	if err != nil {
		return errors.New("could not open a loopback login callback")
	}
	defer listener.Close()
	config := a.config
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		return err
	}
	callbackHost := net.JoinHostPort("127.0.0.1", port)
	config.RedirectURL = "http://" + callbackHost + "/"
	state, verifier := rand.Text(), oauth2.GenerateVerifier()
	type callback struct {
		code string
		err  error
	}
	response := make(chan callback, 1)
	var once sync.Once
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		q := r.URL.Query()
		if r.Host != callbackHost || len(q["state"]) != 1 || subtle.ConstantTimeCompare([]byte(q.Get("state")), []byte(state)) != 1 {
			http.Error(w, "Invalid sign-in response.", http.StatusBadRequest)
			return
		}
		result := callback{code: q.Get("code")}
		if q.Get("error") != "" || len(q["code"]) != 1 || result.code == "" {
			result.err = errors.New("Google sign-in was declined or returned no authorization code")
		}
		once.Do(func() { response <- result })
		fmt.Fprint(w, "To Do Connect received the sign-in response. Return to the terminal to check the connection result. You can close this tab.")
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}
	var workers sync.WaitGroup
	workers.Go(func() { _ = srv.Serve(listener) })
	defer func() { _ = srv.Close(); workers.Wait() }()
	url := config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent select_account"), oauth2.S256ChallengeOption(verifier))
	if openURL(url) != nil {
		return errors.New("could not open the system browser; no authorization URL or code was logged")
	}
	var result callback
	select {
	case <-ctx.Done():
		return errors.New("Google sign-in canceled or timed out")
	case result = <-response:
	}
	if result.err != nil {
		return result.err
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, a.http)
	token, err := config.Exchange(ctx, result.code, oauth2.VerifierOption(verifier))
	if err != nil {
		return errors.New("Google authorization exchange failed; check Desktop client configuration and consent; no credentials were logged")
	}
	if token.RefreshToken == "" {
		return errors.New("Google did not grant offline access; existing credentials were not changed")
	}
	if granted, ok := token.Extra("scope").(string); ok && !slices.Contains(strings.Fields(granted), googleTasksScope) {
		return errors.New("Google Tasks permission was not granted; existing credentials were not changed")
	}
	subject, err := a.subject(ctx, token)
	if err != nil {
		return err
	}
	if previous.Subject != "" && previous.Subject != subject {
		return errors.New("a different Google account was selected; original credentials retained. Add another connection for that account")
	}
	return a.save(googleSession{Token: token, Subject: subject})
}
