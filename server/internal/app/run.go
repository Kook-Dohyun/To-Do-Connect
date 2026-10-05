package app

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Set by release builds through -ldflags -X. Unstamped local builds identify as dev.
var buildVersion = "dev"

const usage = `To Do Connect — local task service connections
  version
  connections list
  connections add --provider microsoft --id ID --label NAME [--client-id ID]
  connections add --provider google --id ID --label NAME --credentials DESKTOP_CLIENT_JSON
  connections remove --id ID --confirm
  connections import-microsoft --id ID --from ENCRYPTED_CACHE_PATH
  login ID [--device] [--reauthenticate]
  login GOOGLE_ID --headless --container-port 8765
  keygen --out ABSOLUTE_PRIVATE_KEY_FILE
  check ID
  serve
  catalog
  call TOOL < arguments.json
Credentials stay on this PC. Removing a connection never deletes remote tasks.
`

// Run executes the local CLI or stdio MCP server.
func Run(ctx context.Context, args []string) error {
	if len(args) == 1 && (args[0] == "version" || args[0] == "--version") {
		fmt.Fprintln(os.Stdout, buildVersion)
		return nil
	}
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		fmt.Fprint(os.Stdout, usage)
		return nil
	}
	c, err := openConnections()
	if err != nil {
		return err
	}
	switch args[0] {
	case "keygen":
		fs := flag.NewFlagSet("keygen", flag.ContinueOnError)
		out := fs.String("out", "", "absolute key-file path; never written to stdout")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 0 {
			return errors.New("unexpected keygen arguments")
		}
		if err := generateKeyFile(*out); err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, "Encryption key created. Keep it separate from data, source control and synchronized folders. Losing it makes its encrypted credentials unreadable.")
		return nil
	case "connections":
		return connectionsCommand(ctx, c, args[1:])
	case "login":
		if len(args) < 2 {
			return errors.New("login requires a connection ID; add a connection first")
		}
		fs := flag.NewFlagSet("login", flag.ContinueOnError)
		device := fs.Bool("device", false, "use terminal device-code flow instead of browser callback")
		reauthenticate := fs.Bool("reauthenticate", false, "explicitly reauthorize the existing account")
		headless := fs.Bool("headless", false, "Google only: show the login URL in the user's terminal instead of opening a browser")
		containerPort := fs.Int("container-port", 0, "Google only: container callback port; publish the same port on host 127.0.0.1 only")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if fs.NArg() != 0 {
			return errors.New("unexpected login arguments")
		}
		return c.login(ctx, args[1], loginOptions{Device: *device, Reauthenticate: *reauthenticate, Headless: *headless, ContainerPort: *containerPort})
	case "check":
		if len(args) != 2 {
			return errors.New("check requires a connection ID")
		}
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		out, err := c.check(ctx, args[1])
		return printJSON(out, err)
	case "serve":
		return server(c).Run(ctx, &mcp.StdioTransport{})
	case "catalog":
		// The CLI and MCP advertise the same tools and required arguments.
		st, ct := mcp.NewInMemoryTransports()
		ss, err := server(c).Connect(ctx, st, nil)
		if err != nil {
			return err
		}
		defer ss.Close()
		client := mcp.NewClient(&mcp.Implementation{Name: "catalog", Version: "1"}, nil)
		cs, err := client.Connect(ctx, ct, nil)
		if err != nil {
			return err
		}
		defer cs.Close()
		out, err := cs.ListTools(ctx, nil)
		return printJSON(out, err)
	case "call":
		if len(args) != 2 {
			return errors.New("call requires a tool name and JSON on stdin")
		}
		b, err := io.ReadAll(io.LimitReader(os.Stdin, 40*1024*1024+1))
		if err != nil {
			return err
		}
		if len(b) > 40*1024*1024 {
			return errors.New("input too large")
		}
		var in connectedInput
		if err := json.Unmarshal(b, &in); err != nil {
			return errors.New("expected JSON tool arguments")
		}
		var out object
		switch args[1] {
		case googleSetupTool:
			var setup googleSetupInput
			if err := json.Unmarshal(b, &setup); err != nil {
				return errors.New("expected Google setup arguments")
			}
			out, err = c.googleSetup(ctx, setup)
		case googleImportTool:
			var setup googleImportInput
			if err := json.Unmarshal(b, &setup); err != nil {
				return errors.New("expected Google import arguments")
			}
			out, err = c.importGoogle(setup)
		case googleLoginTool:
			var setup googleLoginInput
			if err := json.Unmarshal(b, &setup); err != nil {
				return errors.New("expected Google login arguments")
			}
			out, err = c.loginGoogle(ctx, setup)
		case "list_connections":
			out, err = c.list()
		case "check_connection":
			out, err = c.check(ctx, in.ConnectionID)
		case "disconnect_connection":
			err = c.remove(in.ConnectionID, in.Confirm)
			out = object{"connection_id": in.ConnectionID, "disconnected": err == nil, "remote_tasks_changed": false}
		default:
			if name, ok := strings.CutPrefix(args[1], googleToolPrefix); ok {
				var googleIn googleInput
				if json.Unmarshal(b, &googleIn) != nil {
					return errors.New("expected Google tool arguments")
				}
				out, err = c.callGoogle(ctx, name, googleIn)
				break
			}
			name, ok := strings.CutPrefix(args[1], microsoftToolPrefix)
			if !ok {
				return errors.New("unknown tool; see catalog")
			}
			out, err = c.callMicrosoft(ctx, name, in)
		}
		return printJSON(out, err)
	default:
		return errors.New("unknown command; run todo-connect help")
	}
}

func printJSON(v any, err error) error {
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(v)
}

func connectionsCommand(ctx context.Context, c *connections, args []string) error {
	if len(args) == 0 {
		return errors.New("connections requires list, add, remove or import-microsoft")
	}
	if args[0] == "list" {
		out, err := c.list()
		return printJSON(out, err)
	}
	fs := flag.NewFlagSet("connections "+args[0], flag.ContinueOnError)
	id := fs.String("id", "", "connection ID")
	var err error
	switch args[0] {
	case "add":
		provider := fs.String("provider", "", "provider: microsoft or google")
		label := fs.String("label", "", "account label shown to you and your AI")
		appID := fs.String("client-id", clientID, "Microsoft public application ID (optional)")
		credentials := fs.String("credentials", "", "Google Desktop app client JSON path or - for stdin; never paste its contents into chat")
		if err = fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 0 {
			return errors.New("unexpected add arguments")
		}
		var googleConfig *googleClient
		if *provider == googleProvider {
			if *credentials == "" {
				return errors.New("Google requires --credentials DESKTOP_CLIENT_JSON")
			}
			googleConfig, err = readGoogleClient(*credentials)
			if err != nil {
				return err
			}
		} else if *credentials != "" {
			return errors.New("--credentials is for Google connections only")
		}
		err = c.add(connection{ID: *id, Provider: *provider, Label: *label, ClientID: *appID}, googleConfig)
		if err == nil {
			fmt.Fprintln(os.Stdout, "Connection created. Run todo-connect login", *id)
		}
	case "remove":
		confirm := fs.Bool("confirm", false, "remove this local connection, not remote tasks")
		if err = fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 0 {
			return errors.New("unexpected remove arguments")
		}
		err = c.remove(*id, *confirm)
		if err == nil {
			fmt.Fprintln(os.Stdout, "Local connection removed. Remote tasks and provider consent were not changed.")
		}
	case "import-microsoft":
		from := fs.String("from", "", "existing encrypted prototype cache path")
		if err = fs.Parse(args[1:]); err != nil {
			return err
		}
		if *from == "" || fs.NArg() != 0 {
			return errors.New("import-microsoft requires --id and --from; first add the destination connection with the same client ID")
		}
		err = c.importMicrosoft(ctx, *id, *from)
		if err == nil {
			fmt.Fprintln(os.Stdout, "Encrypted cache copied; original preserved. Run check to verify access.")
		}
	default:
		return errors.New("unknown connections command")
	}
	return err
}
