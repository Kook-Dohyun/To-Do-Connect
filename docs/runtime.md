# Runtime and storage

To Do Connect is one local executable with a stdio MCP server and setup CLI. It connects directly to provider APIs. Native packages do not require a developer server or Docker.

## Account data

| OS | Default directory | Default encryption |
| --- | --- | --- |
| Windows | `%LOCALAPPDATA%\todo-connect` | Current user's DPAPI |
| macOS | `~/Library/Application Support/todo-connect` | OS Keychain key with AES-GCM |
| Linux | `${XDG_CONFIG_HOME:-$HOME/.config}/todo-connect` | Secret Service key with AES-GCM |

Program installation and plugin caches are separate from account data. Windows packaged applications can redirect AppData access. An absolute `TODO_CONNECT_DATA_DIR` overrides the default; CLI and MCP must receive the same directory and encryption settings.

Each connection stores nonsecret metadata in `.json`, encrypted tokens in `.secret`, and, for Google, encrypted imported client settings in `.client`. A `.lock` coordinates processes. The original downloaded client JSON remains at its download location.

Linux needs an available Secret Service session. If the key store is unavailable, the program does not silently switch to plaintext. For a headless host, select key-file storage explicitly.

## Key-file storage

Create one key outside the repository and package:

```text
PROGRAM keygen --out ABSOLUTE_PRIVATE_KEY_PATH
```

Set `TODO_CONNECT_KEY_FILE` to that path, not to the key contents. Use the same setting for setup and MCP. Keep permissions private and persist the original key alongside a recoverable backup strategy for encrypted data.

Losing the original key makes the encrypted data unreadable. Do not replace a key to repair a login error. Changing the data directory or encryption mode is not an automatic migration. Windows DPAPI caches cannot be reused directly in a Linux container.

## Docker

Docker is an optional alternative. Use a published image archive when available, or build locally:

```text
docker image load --input todo-connect-VERSION-linux-ARCH.tar.gz
docker run --rm todo-connect:VERSION version
```

```text
docker build -f packaging/docker/Dockerfile -t todo-connect:local .
```

The following examples use `todo-connect:local`; replace it with the loaded image tag as appropriate. The runtime uses UID/GID 10001 and includes license notices under `/usr/share/licenses/todo-connect/`.

### Persistent storage

```text
docker volume create todo-connect-data
docker volume create todo-connect-keys
docker run --rm -v todo-connect-data:/data -v todo-connect-keys:/keys todo-connect:local keygen --out /keys/todo-connect.key
```

Create the key only once. Reuse both volumes; do not recreate storage on startup.

### Microsoft sign-in

```text
docker run --rm -v todo-connect-data:/data -v todo-connect-keys:/keys:ro todo-connect:local connections add --provider microsoft --id personal --label "Personal Microsoft"
docker run --rm -it -v todo-connect-data:/data -v todo-connect-keys:/keys:ro todo-connect:local login personal --device
docker run --rm -v todo-connect-data:/data -v todo-connect-keys:/keys:ro todo-connect:local check personal
```

Complete the device-login instructions in your terminal and browser. No callback port is required. Do not paste login codes into chat.

### Google sign-in

Prepare your own Cloud project, enabled Tasks API and Desktop OAuth client with the bundled Google skill. Import the downloaded JSON via stdin without printing it.

PowerShell:

```powershell
Get-Content -Raw -LiteralPath 'ABSOLUTE_DESKTOP_CLIENT_JSON_PATH' | docker run --rm -i -v todo-connect-data:/data -v todo-connect-keys:/keys:ro todo-connect:local connections add --provider google --id google-personal --label 'Personal Google' --credentials -
```

Unix shell:

```sh
docker run --rm -i -v todo-connect-data:/data -v todo-connect-keys:/keys:ro todo-connect:local connections add --provider google --id google-personal --label 'Personal Google' --credentials - < '/absolute/private/desktop-client.json'
```

Publish a loopback callback only for login and open the browser on the Docker host:

```text
docker run --rm -it -p 127.0.0.1:8765:8765 -v todo-connect-data:/data -v todo-connect-keys:/keys:ro todo-connect:local login google-personal --headless --container-port 8765
docker run --rm -v todo-connect-data:/data -v todo-connect-keys:/keys:ro todo-connect:local check google-personal
```

If needed, choose another port and use it consistently on both sides of the mapping and in `--container-port`. Do not expose the callback publicly.

### MCP configuration

```json
{
  "mcpServers": {
    "todo-connect": {
      "command": "docker",
      "args": ["run", "--rm", "-i", "-v", "todo-connect-data:/data", "-v", "todo-connect-keys:/keys:ro", "todo-connect:local", "serve"]
    }
  }
}
```

MCP uses stdin/stdout, no TTY and no published port. Use the same data/key volumes as setup.

## Disconnect and uninstall

`PROGRAM connections remove --id ID --confirm` or the `disconnect_connection` tool removes only that local connection. Remote tasks and service-side consent remain.

Uninstalling a program or plugin is not account disconnection. Keep the original key and data directory during updates. See [installation](https://github.com/Kook-Dohyun/To-Do-Connect/blob/main/docs/installation.md) and [privacy](https://github.com/Kook-Dohyun/To-Do-Connect/blob/main/docs/privacy.md).
