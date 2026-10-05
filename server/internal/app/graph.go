package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const graphBase = "https://graph.microsoft.com/v1.0"
const listsPath = "/me/todo/lists"
const maxFile = 25 * 1024 * 1024

type object = map[string]any
type graph struct {
	base  string
	http  *http.Client
	token func(context.Context) (string, error)
}

func newGraph(token func(context.Context) (string, error)) *graph {
	return &graph{graphBase, &http.Client{Timeout: 90 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, token}
}
func segment(s string) (string, error) {
	if s == "" || s == "." || s == ".." || strings.ContainsAny(s, "/\\\x00\r\n") {
		return "", errors.New("non-empty opaque resource ID required (not a URL or path)")
	}
	return url.PathEscape(s), nil
}
func listPath(id string) (string, error) { s, e := segment(id); return listsPath + "/" + s, e }
func taskPath(l, t string) (string, error) {
	p, e := listPath(l)
	if e != nil {
		return "", e
	}
	s, e := segment(t)
	return p + "/tasks/" + s, e
}

// Absolute URLs are accepted only when issued by Graph and confined to this API.
func (g *graph) absolute(raw string) (string, error) {
	b, _ := url.Parse(g.base)
	u, err := url.Parse(raw)
	if err != nil {
		return "", errors.New("invalid Graph URL")
	}
	if !u.IsAbs() {
		if !strings.HasPrefix(raw, listsPath) {
			return "", errors.New("outside To Do scope")
		}
		u, err = url.Parse(g.base + raw)
		if err != nil {
			return "", err
		}
	}
	if u.Scheme != b.Scheme || u.Host != b.Host || u.User != nil || u.Fragment != "" {
		return "", errors.New("Graph URL origin mismatch")
	}
	if !strings.HasPrefix(u.EscapedPath(), b.EscapedPath()+"/") {
		return "", errors.New("outside Graph v1.0")
	}
	rel := strings.TrimPrefix(u.EscapedPath(), b.EscapedPath())
	if rel != listsPath && !strings.HasPrefix(rel, listsPath+"/") && !(strings.HasPrefix(rel, "/users/") && strings.Contains(rel, "/todo/lists/")) {
		return "", errors.New("outside To Do scope")
	}
	for _, part := range strings.Split(u.Path, "/") {
		if part == ".." || part == "." {
			return "", errors.New("path traversal is not allowed")
		}
	}
	return u.String(), nil
}
func (g *graph) request(ctx context.Context, method, path string, body []byte, headers map[string]string) (object, error) {
	addr, err := g.absolute(path)
	if err != nil {
		return nil, err
	}
	tok, err := g.token(ctx)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, addr, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := g.http.Do(req)
	if err != nil {
		return nil, errors.New("Graph transport failed; write outcome may be unknown; inspect before retrying")
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 40*1024*1024+1))
	if err != nil {
		return nil, errors.New("Graph response interrupted; inspect before retrying")
	}
	if len(b) > 40*1024*1024 {
		return nil, errors.New("Graph response exceeds 40 MiB")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var v struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		_ = json.Unmarshal(b, &v)
		return nil, fmt.Errorf("Graph HTTP %d (%s); request-id=%s; retry-after=%s", resp.StatusCode, v.Error.Code, resp.Header.Get("request-id"), resp.Header.Get("Retry-After"))
	}
	if strings.HasSuffix(req.URL.Path, "/$value") {
		return object{"contentBytes": base64.StdEncoding.EncodeToString(b), "contentType": resp.Header.Get("Content-Type")}, nil
	}
	out := object{}
	if len(b) > 0 {
		if err = json.Unmarshal(b, &out); err != nil {
			return nil, errors.New("Graph returned invalid JSON")
		}
	}
	if loc := resp.Header.Get("Location"); loc != "" {
		out["location"] = loc
	}
	out["httpStatus"] = resp.StatusCode
	return out, nil
}
func (g *graph) json(ctx context.Context, method, path string, body object) (object, error) {
	var b []byte
	var err error
	if body != nil {
		b, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	return g.request(ctx, method, path, b, nil)
}
func (g *graph) all(ctx context.Context, path string) ([]object, error) {
	var items []object
	for path != "" {
		r, e := g.json(ctx, http.MethodGet, path, nil)
		if e != nil {
			return nil, e
		}
		a, ok := r["value"].([]any)
		if !ok {
			return nil, errors.New("Graph collection missing value")
		}
		for _, v := range a {
			o, ok := v.(map[string]any)
			if !ok {
				return nil, errors.New("invalid collection item")
			}
			items = append(items, o)
		}
		path, _ = r["@odata.nextLink"].(string)
	}
	return items, nil
}
