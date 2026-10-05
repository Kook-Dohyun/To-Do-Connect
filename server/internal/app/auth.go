package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
)

// Public application identifier, not a secret. User tokens remain in the encrypted cache.
const clientID = "dffc9edd-0513-4e7e-ac0e-70656932a965"

var scopes = []string{"https://graph.microsoft.com/Tasks.ReadWrite"}

type diskCache struct{ path string }

func (d diskCache) Replace(ctx context.Context, c cache.Unmarshaler, _ cache.ReplaceHints) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b, err := os.ReadFile(d.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	b, err = unprotect(b)
	if err != nil {
		return err
	}
	return c.Unmarshal(b)
}
func (d diskCache) Export(ctx context.Context, c cache.Marshaler, _ cache.ExportHints) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b, err := c.Marshal()
	if err != nil {
		return err
	}
	b, err = protect(b)
	if err != nil {
		return err
	}
	return writePrivate(d.path, b)
}

// Only encrypted bytes touch disk. Rename prevents a truncated cache after interruption.
func writePrivate(path string, b []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".auth-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

type auth struct {
	client public.Client
}

func newAuth(appID, path string) (*auth, error) {
	client, err := public.New(appID, public.WithAuthority("https://login.microsoftonline.com/consumers"), public.WithCache(diskCache{path}))
	return &auth{client: client}, err
}
func (a *auth) token(ctx context.Context) (string, error) {
	accounts, err := a.client.Accounts(ctx)
	if err != nil {
		return "", errors.New("could not load account cache")
	}
	if len(accounts) != 1 {
		return "", errors.New("this connection is not signed in; run todo-connect login CONNECTION_ID locally")
	}
	r, err := a.client.AcquireTokenSilent(ctx, scopes, public.WithSilentAccount(accounts[0]))
	if err != nil {
		return "", errors.New("Microsoft could not acquire a token; check connectivity and consent, then run todo-connect login CONNECTION_ID if reauthentication is needed")
	}
	return r.AccessToken, nil
}
func (a *auth) login(ctx context.Context, device, reauthenticate bool) error {
	// Fail before opening a browser if this build cannot persist credentials securely.
	if _, err := protect([]byte("credential-storage-check")); err != nil {
		return err
	}
	accounts, err := a.client.Accounts(ctx)
	if err != nil {
		return errors.New("could not load this connection's encrypted account cache")
	}
	if len(accounts) > 0 && !reauthenticate {
		return errors.New("this connection already has credentials; check it first, or explicitly request --reauthenticate")
	}
	if len(accounts) > 1 {
		return errors.New("connection cache has more than one account; disconnect and add a new connection")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	var r public.AuthResult
	if !device {
		fmt.Fprintln(os.Stderr, "Opening Microsoft sign-in in your browser. This app registration must allow the desktop redirect URI http://localhost.")
		r, err = a.client.AcquireTokenInteractive(ctx, scopes)
		if err != nil {
			return errors.New("browser sign-in did not complete; check network, consent and the app's desktop http://localhost redirect URI. Device login is available with --device")
		}
	} else {
		r, err = a.deviceLogin(ctx)
		if err != nil {
			return err
		}
	}
	if len(accounts) == 1 && accounts[0].HomeAccountID != r.Account.HomeAccountID {
		// Reauthentication is not an implicit switch to another person's task data.
		if err := a.client.RemoveAccount(ctx, r.Account); err != nil {
			return errors.New("wrong account selected; cleanup failed, disconnect this connection before use")
		}
		return errors.New("a different account was selected; original account retained. Add another connection for the new account")
	}
	fmt.Fprintln(os.Stderr, "Microsoft personal account connected.")
	return nil
}

func (a *auth) deviceLogin(ctx context.Context) (public.AuthResult, error) {
	code, err := a.client.AcquireTokenByDeviceCode(ctx, scopes)
	if err != nil {
		return public.AuthResult{}, errors.New("could not start device login; check app ID and public-client flow setting")
	}
	// Interactive terminal only; never returned by an MCP tool or written to a log.
	fmt.Fprintln(os.Stderr, code.Result.Message)
	r, err := code.AuthenticationResult(ctx)
	if err != nil {
		return public.AuthResult{}, errors.New("Microsoft login did not complete; no password or token has been printed")
	}
	return r, nil
}
