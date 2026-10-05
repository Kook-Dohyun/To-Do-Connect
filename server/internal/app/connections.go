package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/pkg/browser"
)

const microsoftProvider = "microsoft"
const dataDirEnv = "TODO_CONNECT_DATA_DIR"

var connectionIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
var errConnectionBusy = errors.New("connection is busy in another request or process; wait for it to finish")

// Connection metadata contains no tokens or passwords. Credentials are in a separate encrypted file.
type connection struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Label    string `json:"label"`
	ClientID string `json:"client_id"`
}

type connections struct{ dir string }

func openConnections() (*connections, error) {
	dir := os.Getenv(dataDirEnv)
	if dir == "" {
		var err error
		if runtime.GOOS == "windows" {
			dir, err = os.UserCacheDir() // LOCALAPPDATA, not the roaming or Vault directory.
		} else {
			dir, err = os.UserConfigDir()
		}
		if err != nil {
			return nil, err
		}
		dir = filepath.Join(dir, "todo-connect")
	}
	if !filepath.IsAbs(dir) {
		return nil, errors.New(dataDirEnv + " must be an absolute directory outside source control and sync folders")
	}
	return &connections{dir: filepath.Clean(dir)}, nil
}

func (c *connections) path(id, suffix string) string {
	return filepath.Join(c.dir, id+suffix)
}

func (c *connections) lock(id string) (func(), error) {
	if !connectionIDPattern.MatchString(id) {
		return nil, errors.New("connection_id must be 1-64 lowercase letters, digits, hyphens or underscores, starting with a letter or digit")
	}
	if err := os.MkdirAll(c.dir, 0700); err != nil {
		return nil, err
	}
	return lockConnection(c.path(id, ".lock"))
}

func (c *connections) load(id string) (connection, error) {
	var v connection
	b, err := os.ReadFile(c.path(id, ".json"))
	if errors.Is(err, os.ErrNotExist) {
		return v, errors.New("connection not found; list connections or add one locally")
	}
	if err != nil {
		return v, err
	}
	if err = json.Unmarshal(b, &v); err != nil || v.ID != id {
		return v, errors.New("invalid connection metadata")
	}
	return v, nil
}

func (c *connections) add(v connection, client *googleClient) error {
	if v.Provider != microsoftProvider && v.Provider != googleProvider {
		return errors.New("supported providers: microsoft, google")
	}
	var encryptedClient []byte
	if v.Provider == googleProvider {
		if client == nil || client.ClientID == "" || client.ClientSecret == "" {
			return errors.New("Google requires a Desktop app client JSON via --credentials")
		}
		v.ClientID = client.ClientID
		b, err := json.Marshal(client)
		if err != nil {
			return err
		}
		encryptedClient, err = protect(b)
		if err != nil {
			return err
		}
	} else if client != nil {
		return errors.New("Google credentials cannot be attached to a Microsoft connection")
	}
	if v.ClientID == "" || strings.TrimSpace(v.Label) == "" {
		return errors.New("client ID and connection label are required")
	}
	unlock, err := c.lock(v.ID)
	if err != nil {
		return err
	}
	defer unlock()
	// The connection lock makes the existence check and atomic write one local operation.
	_, err = os.Stat(c.path(v.ID, ".json"))
	if err == nil {
		return errors.New("connection already exists; use check or login, or choose another connection ID")
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if encryptedClient != nil {
		if err := writePrivate(c.path(v.ID, ".client"), encryptedClient); err != nil {
			return err
		}
	}
	if err := writePrivate(c.path(v.ID, ".json"), b); err != nil {
		if encryptedClient != nil {
			return errors.Join(err, os.Remove(c.path(v.ID, ".client")))
		}
		return err
	}
	return nil
}

func (c *connections) list() (object, error) {
	entries, err := os.ReadDir(c.dir)
	if errors.Is(err, os.ErrNotExist) {
		return object{"connections": []object{}}, nil
	}
	if err != nil {
		return nil, err
	}
	items := []object{}
	for _, entry := range entries {
		id, ok := strings.CutSuffix(entry.Name(), ".json")
		if !ok || entry.IsDir() || !connectionIDPattern.MatchString(id) {
			continue
		}
		unlock, err := c.lock(id)
		if err != nil {
			if !errors.Is(err, errConnectionBusy) {
				return nil, err
			}
			items = append(items, object{"id": id, "state": "busy"})
			continue
		}
		v, err := c.load(id)
		if err == nil {
			_, cacheErr := os.Stat(c.path(id, ".secret"))
			state := "not_connected"
			if cacheErr == nil {
				state = "credentials_stored_not_verified"
			} else if !errors.Is(cacheErr, os.ErrNotExist) {
				err = cacheErr
			}
			items = append(items, object{"id": v.ID, "provider": v.Provider, "label": v.Label, "state": state})
		}
		unlock()
		if err != nil {
			return nil, fmt.Errorf("connection %s: %w", id, err)
		}
	}
	return object{"connections": items}, nil
}

func (c *connections) withConnection(ctx context.Context, id string, fn func(connection) (object, error)) (object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	unlock, err := c.lock(id)
	if err != nil {
		return nil, err
	}
	defer unlock()
	v, err := c.load(id)
	if err != nil {
		return nil, err
	}
	return fn(v)
}

func (c *connections) withMicrosoft(ctx context.Context, id string, fn func(*auth, *graph) (object, error)) (object, error) {
	return c.withConnection(ctx, id, func(v connection) (object, error) {
		if v.Provider != microsoftProvider {
			return nil, errors.New("this tool requires a Microsoft connection")
		}
		a, err := newAuth(v.ClientID, c.path(id, ".secret"))
		if err != nil {
			return nil, err
		}
		return fn(a, newGraph(a.token))
	})
}

func (c *connections) googleAuth(v connection) (*googleAuth, error) {
	b, err := os.ReadFile(c.path(v.ID, ".client"))
	if err != nil {
		return nil, errors.New("could not read the Google client's encrypted configuration")
	}
	b, err = unprotect(b)
	if err != nil {
		return nil, errors.New("could not decrypt Google client configuration")
	}
	var client googleClient
	if json.Unmarshal(b, &client) != nil || client.ClientID != v.ClientID || client.ClientSecret == "" {
		return nil, errors.New("invalid Google client configuration")
	}
	return newGoogleAuth(client, c.path(v.ID, ".secret")), nil
}

func (c *connections) withGoogle(ctx context.Context, id string, fn func(*googleAuth) (object, error)) (object, error) {
	return c.withConnection(ctx, id, func(v connection) (object, error) {
		if v.Provider != googleProvider {
			return nil, errors.New("this tool requires a Google connection")
		}
		a, err := c.googleAuth(v)
		if err != nil {
			return nil, err
		}
		return fn(a)
	})
}

type loginOptions struct {
	GoogleOnly     bool
	Device         bool
	Reauthenticate bool
	Headless       bool
	ContainerPort  int
}

func (c *connections) login(ctx context.Context, id string, options loginOptions) error {
	if options.ContainerPort != 0 && (!options.Headless || options.ContainerPort < 1024 || options.ContainerPort > 65535) {
		return errors.New("--container-port requires --headless and a port between 1024 and 65535; publish it on host 127.0.0.1 only")
	}
	_, err := c.withConnection(ctx, id, func(v connection) (object, error) {
		if options.GoogleOnly && v.Provider != googleProvider {
			return nil, errors.New("this login tool requires a Google connection")
		}
		switch v.Provider {
		case microsoftProvider:
			if options.Headless || options.ContainerPort != 0 {
				return nil, errors.New("Microsoft container login uses --device, not Google headless callback flags")
			}
			a, err := newAuth(v.ClientID, c.path(id, ".secret"))
			if err != nil {
				return nil, err
			}
			return nil, a.login(ctx, options.Device, options.Reauthenticate)
		case googleProvider:
			if options.Device {
				return nil, errors.New("Google Tasks uses desktop browser login, not device-code login")
			}
			a, err := c.googleAuth(v)
			if err != nil {
				return nil, err
			}
			fmt.Fprintln(os.Stderr, "Google sign-in uses your system browser; password and consent stay with Google.")
			openURL := browser.OpenURL
			if options.Headless {
				openURL = func(url string) error {
					_, err := fmt.Fprintln(os.Stderr, "Open this sign-in URL in the browser on the Docker host (do not share or log it):\n"+url)
					return err
				}
			}
			if options.ContainerPort != 0 {
				a.listenAddress = fmt.Sprintf("0.0.0.0:%d", options.ContainerPort)
			}
			if err := a.login(ctx, options.Reauthenticate, openURL); err != nil {
				return nil, err
			}
			fmt.Fprintln(os.Stderr, "Google account connected.")
			return nil, nil
		default:
			return nil, errors.New("unsupported provider")
		}
	})
	return err
}

func (c *connections) check(ctx context.Context, id string) (object, error) {
	return c.withConnection(ctx, id, func(v connection) (object, error) {
		switch v.Provider {
		case microsoftProvider:
			a, err := newAuth(v.ClientID, c.path(id, ".secret"))
			if err != nil {
				return nil, err
			}
			if _, err := newGraph(a.token).json(ctx, "GET", listsPath+"?$top=1", nil); err != nil {
				return nil, err
			}
		case googleProvider:
			a, err := c.googleAuth(v)
			if err != nil {
				return nil, err
			}
			if _, err := newGoogleTasks(a.token).execute(ctx, googleOperations()[0], googleInput{Query: map[string]string{"maxResults": "1"}}); err != nil {
				return nil, err
			}
		default:
			return nil, errors.New("unsupported provider")
		}
		return object{"connection_id": id, "provider": v.Provider, "read_access": "verified"}, nil
	})
}

func (c *connections) remove(id string, confirm bool) error {
	if !confirm {
		return errors.New("disconnect requires explicit confirmation; no remote tasks will be deleted")
	}
	unlock, err := c.lock(id)
	if err != nil {
		return err
	}
	defer unlock()
	if _, err = c.load(id); err != nil {
		return err
	}
	if err = os.Remove(c.path(id, ".secret")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err = os.Remove(c.path(id, ".client")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	// The lock file remains so another process cannot lock a different inode during removal.
	return os.Remove(c.path(id, ".json"))
}

func (c *connections) importMicrosoft(ctx context.Context, id, source string) error {
	_, err := c.withMicrosoft(ctx, id, func(_ *auth, _ *graph) (object, error) {
		if _, err := os.Stat(c.path(id, ".secret")); !errors.Is(err, os.ErrNotExist) {
			return nil, errors.New("destination already has credentials or cannot be inspected")
		}
		// MSAL validates the existing encrypted cache locally; no token or task is printed.
		v, err := c.load(id)
		if err != nil {
			return nil, err
		}
		old, err := newAuth(v.ClientID, source)
		if err != nil {
			return nil, err
		}
		accounts, err := old.client.Accounts(ctx)
		if err != nil || len(accounts) != 1 {
			return nil, errors.New("source must be a readable encrypted MSAL cache with exactly one account")
		}
		b, err := os.ReadFile(source)
		if err != nil {
			return nil, err
		}
		// Copy encrypted bytes; leave the prototype's original cache intact.
		if err = writePrivate(c.path(id, ".secret"), b); err != nil {
			return nil, err
		}
		return object{"imported": true}, nil
	})
	return err
}
