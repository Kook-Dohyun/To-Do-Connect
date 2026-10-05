package app

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const googleSetupTool = "get_google_setup"
const googleImportTool = "import_google_connection"
const googleLoginTool = "login_google_connection"

type googleSetupInput struct {
	ConnectionID    string `json:"connection_id,omitempty" jsonschema:"Existing Google connection ID; omit to discover existing connections before setup"`
	CredentialsPath string `json:"credentials_path,omitempty" jsonschema:"Absolute local Desktop OAuth JSON path, never JSON contents. Optional for pre-import inspection or identifying a legacy client's project."`
	Verify          bool   `json:"verify,omitzero" jsonschema:"Make a read-only Tasks request with stored credentials; may refresh the saved token, never opens login"`
}

type googleImportInput struct {
	ConnectionID    string `json:"connection_id" jsonschema:"New unique connection ID; never overwrites an existing connection"`
	Label           string `json:"label" jsonschema:"User-visible name for this Google account connection"`
	CredentialsPath string `json:"credentials_path" jsonschema:"Absolute local path to the downloaded Desktop OAuth JSON; never its contents"`
	Confirm         bool   `json:"confirm" jsonschema:"True only when the user requested adding this Google connection"`
}

type googleLoginInput struct {
	ConnectionID   string `json:"connection_id" jsonschema:"Existing Google connection ID to authorize in the user's system browser"`
	Confirm        bool   `json:"confirm" jsonschema:"True only when the user requested connecting or reauthorizing this account"`
	Reauthenticate bool   `json:"reauthenticate,omitzero" jsonschema:"Explicit same-account reauthorization; never set for API_DISABLED or a generic network error"`
}

// No client JSON, API key, token or Cloud-management scope is needed to inspect
// initial setup. Cloud Console actions belong to the bundled skill/browser.
func (c *connections) googleSetup(ctx context.Context, in googleSetupInput) (object, error) {
	var client *googleClient
	if in.CredentialsPath != "" {
		if !filepath.IsAbs(in.CredentialsPath) {
			return nil, errors.New("credentials_path must be an absolute local file path, not JSON or stdin")
		}
		var err error
		client, err = readGoogleClient(in.CredentialsPath)
		if errors.Is(err, os.ErrNotExist) {
			return googleSetupResult("client_json_required", "", "Continue the Google setup skill: reuse or create the user's project, enable Tasks API, configure consent, and download a Desktop OAuth client JSON to a private local folder."), nil
		}
		if err != nil {
			return nil, err
		}
	}
	if in.ConnectionID == "" {
		listed, err := c.list()
		if err != nil {
			return nil, err
		}
		state, action, project := "client_json_required", "Continue the Google setup skill in the user's browser. Missing credentials are a setup step, not a terminal failure.", ""
		for _, item := range listed["connections"].([]object) {
			if item["provider"] == googleProvider || item["state"] == "busy" {
				state, action = "select_connection", "Select an existing Google connection (or explicitly request an additional account); do not recreate setup. Wait for a busy connection before deciding."
				break
			}
		}
		if client != nil {
			project = client.ProjectID
			if state != "select_connection" {
				state, action = "import_required", "Use import_google_connection with this local path and a new connection ID. API enablement and read access are not yet verified."
			}
		}
		out := googleSetupResult(state, project, action)
		out["connections"] = listed["connections"]
		return out, nil
	}
	return c.withGoogle(ctx, in.ConnectionID, func(a *googleAuth) (object, error) {
		if client != nil {
			if client.ClientID != a.config.ClientID {
				return nil, errors.New("the supplied JSON belongs to a different OAuth client; existing connection was not changed")
			}
			// Older imports did not preserve project_id. Inspect the matching source
			// without replacing credentials or forcing another login.
			if a.projectID == "" {
				a.projectID = client.ProjectID
			}
		}
		out, err := googleSetupAccess(ctx, a, newGoogleTasks(a.token), in.Verify)
		if out != nil {
			out["connection_id"] = in.ConnectionID
		}
		return out, err
	})
}

func googleSetupAccess(ctx context.Context, a *googleAuth, tasks *googleTasks, verify bool) (object, error) {
	_, err := a.load()
	if errors.Is(err, os.ErrNotExist) {
		return googleSetupResult("login_required", a.projectID, "Use login_google_connection with user confirmation. Password, MFA and Tasks consent remain in Google's browser UI."), nil
	}
	if err != nil {
		return nil, err
	}
	if !verify {
		return googleSetupResult("verification_required", a.projectID, "Call get_google_setup with verify=true. Stored credentials alone do not prove Tasks access; do not repeat login."), nil
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_, err = tasks.execute(ctx, googleOperations()[0], googleInput{Query: map[string]string{"maxResults": "1"}})
	switch {
	case err == nil:
		out := googleSetupResult("ready", a.projectID, "Reuse this connection for the requested task; no Console setup, client import or login is needed.")
		out["read_access"] = "verified"
		return out, nil
	case errors.Is(err, errGoogleAPIDisabled):
		return googleSetupResult("api_enable_required", a.projectID, "Continue the skill's API enablement step in the OAuth client's project, then repeat get_google_setup with verify=true. Keep the existing client and token; do not log in again."), nil
	case errors.Is(err, errGoogleTasksPermission):
		return googleSetupResult("consent_required", a.projectID, "Explain the missing Tasks permission and request same-account reauthorization with Tasks access selected. Do not recreate the project/client."), nil
	default:
		out := googleSetupResult("check_failed", a.projectID, "Read access is unverified. Diagnose connectivity, quota or expired/revoked consent; a generic failure is not evidence that API enablement or a new project/login is required.")
		out["diagnostic"] = err.Error()
		return out, nil
	}
}

func googleSetupResult(state, project, action string) object {
	query := ""
	if project != "" {
		query = "?project=" + url.QueryEscape(project)
	}
	return object{
		"state": state, "project_id": project, "next_action": action,
		"console": object{
			"projects":  "https://console.cloud.google.com/projectselector2/home/dashboard",
			"tasks_api": "https://console.cloud.google.com/apis/library/tasks.googleapis.com" + query,
			"branding":  "https://console.cloud.google.com/auth/branding" + query,
			"audience":  "https://console.cloud.google.com/auth/audience" + query,
			"clients":   "https://console.cloud.google.com/auth/clients" + query,
		},
		"project_check": "When project_id is empty, identify the project owning this OAuth client in Console or inspect the matching downloaded JSON; do not assume the currently selected project is correct.",
		"guide":         "google-tasks/references/connections.md",
	}
}

func (c *connections) importGoogle(in googleImportInput) (object, error) {
	if !in.Confirm {
		return nil, errors.New("adding a Google connection requires user intent and confirm=true")
	}
	if !filepath.IsAbs(in.CredentialsPath) {
		return nil, errors.New("credentials_path must be an absolute local file path")
	}
	client, err := readGoogleClient(in.CredentialsPath)
	if err != nil {
		return nil, err
	}
	if err := c.add(connection{ID: in.ConnectionID, Provider: googleProvider, Label: in.Label}, client); err != nil {
		return nil, err
	}
	out := googleSetupResult("login_required", client.ProjectID, "Client imported and encrypted. Continue with login_google_connection; original JSON was not changed.")
	out["connection_id"] = in.ConnectionID
	return out, nil
}

func (c *connections) loginGoogle(ctx context.Context, in googleLoginInput) (object, error) {
	if !in.Confirm {
		return nil, errors.New("browser login requires user intent and confirm=true")
	}
	// Check the provider under the same lock as login before opening a browser.
	if err := c.login(ctx, in.ConnectionID, loginOptions{GoogleOnly: true, Reauthenticate: in.Reauthenticate}); err != nil {
		return nil, err
	}
	return c.googleSetup(ctx, googleSetupInput{ConnectionID: in.ConnectionID, Verify: true})
}

func addGoogleSetupTools(s *mcp.Server, c *connections) {
	mcp.AddTool(s, &mcp.Tool{Name: googleSetupTool, Description: "Inspect Google setup without requiring credentials first. Returns the next setup action and Console links; optionally verifies existing Tasks access. Use the bundled Google skill/browser to complete missing Console steps, not just report them as blockers.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}}, func(ctx context.Context, _ *mcp.CallToolRequest, in googleSetupInput) (*mcp.CallToolResult, object, error) {
		return result(c.googleSetup(ctx, in))
	})
	no := false
	mcp.AddTool(s, &mcp.Tool{Name: googleImportTool, Description: "Import a downloaded Desktop OAuth JSON by local path into a new encrypted connection. No secret contents in tool arguments. Requires user request and confirm=true; never replaces an existing connection.", Annotations: &mcp.ToolAnnotations{DestructiveHint: &no}}, func(_ context.Context, _ *mcp.CallToolRequest, in googleImportInput) (*mcp.CallToolResult, object, error) {
		return result(c.importGoogle(in))
	})
	mcp.AddTool(s, &mcp.Tool{Name: googleLoginTool, Description: "Open the local system browser for a user-requested Google connection, wait for user sign-in/consent (up to 10 minutes), then verify read access. Password/MFA/consent stay with Google. Requires confirm=true. Not for headless Docker; use the bundled CLI procedure there.", Annotations: &mcp.ToolAnnotations{DestructiveHint: &no}}, func(ctx context.Context, _ *mcp.CallToolRequest, in googleLoginInput) (*mcp.CallToolResult, object, error) {
		return result(c.loginGoogle(ctx, in))
	})
}
