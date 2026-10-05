# To Do Connect

<img src="packaging/plugin/assets/icon.png" width="96" height="96" alt="To Do Connect">

**Connect your task services to AI.**

English · [한국어](README.ko.md)

Use Microsoft To Do and Google Tasks from your AI assistant. Connect one account or several, from either or both services, and keep using the task apps you already know.

The program runs on your computer and talks directly to task providers. There is no developer-hosted relay. Native packages include the program: no Go, Node.js or Docker installation is required.

> Development preview, not a stable release. Source and local development packages may be newer than [GitHub Releases](https://github.com/Kook-Dohyun/To-Do-Connect/releases). GitHub distribution and the OpenAI public plugin directory are separate.

## What you can do

| Service | Capabilities |
| --- | --- |
| Microsoft To Do | Create, read, edit, complete and delete lists, tasks and checklist steps |
| Google Tasks | Create, read, edit, complete and delete lists and tasks; create subtasks and move or reorder tasks |
| Connections | Reuse saved connections, add multiple accounts and disconnect accounts individually |

The current server exposes 27 tools. It does not synchronize or copy tasks between services.

## Install

Choose one path; you do not need all three.

- **Codex plugin ZIP:** download the plugin ZIP for your OS and CPU, then use **Plugins → Add → Upload plugin archive** in a supported desktop client.
- **GitHub installation with AI:** copy the request below into your local AI assistant.
- **Other MCP clients:** install the native package and configure its executable with `serve`. See [installation and MCP JSON](docs/installation.md).

```text
Install To Do Connect from https://github.com/Kook-Dohyun/To-Do-Connect.
Check my execution host's OS/CPU and existing installation first.
Use a published prebuilt package and verify its checksum; do not build from
source or install Go, Node.js or Docker. If only preview packages are available,
tell me the version. Register it in my chosen local MCP/plugin client.
Preserve saved connections, verify tool discovery, then ask which task service
and account I want to connect. Guide any unfinished setup.
```

If no suitable published package exists, use a provided development archive or wait for a release. Do not assume the source version has a downloadable asset.

## Connect your accounts

- **Microsoft To Do:** complete browser sign-in with a Microsoft personal account. The application ID is built in; no environment variable or app registration is needed.
- **Google Tasks:** use your own Google Cloud project, enable Tasks API and create a Desktop OAuth client. The bundled skill guides setup, imports the downloaded JSON locally and opens browser sign-in. Existing projects, clients and working connections are reused.

Passwords, MFA and consent stay in the provider's UI. Working connections do not require login on every request.

## Try it

```text
Show my task lists.
Create a list called Server-room checks.
Add “1. Check server identity” with steps “Record hostname” and “Record OS version”.
Show which tasks and steps I have completed.
Show tasks and checklist steps completed since 3 PM today in my time zone.
```

When multiple accounts or lists match, the assistant asks which one to use.

Completion reports use the provider's recorded completion timestamps, not chat memory. The assistant resolves your date/time zone and distinguishes tasks from steps or subtasks. Reports describe currently completed items; they are not a full history of deleted/reopened items, background monitoring or completion hooks.

## Keep using your other apps

To Do Connect can work with tasks that other apps synchronize into the same Microsoft To Do account and list. These are existing provider integrations, not additional providers implemented by this plugin.

| App or service | How it connects |
| --- | --- |
| Samsung Reminder on Galaxy | Enable **Sync with Microsoft To Do** in Reminder settings and use the same Microsoft account in this plugin. AI can then work with the synchronized To Do list while you use Reminder on your phone. [Official setup](https://support.microsoft.com/en-us/todo/sync-microsoft-to-do-with-the-samsung-reminder-app) |
| Outlook | Outlook tasks synchronize with To Do when you use the same Microsoft account. This plugin operates on the To Do tasks, not the email inbox or calendar. [Official overview](https://support.microsoft.com/en-us/outlook/how-can-i-manage-my-outlook-tasks-on-mobile) |
| Zapier | Its Microsoft To Do connector can create tasks and trigger workflows on task creation/completion. Connect it separately; this plugin can use the resulting To Do tasks. [Connector guide](https://help.zapier.com/hc/en-us/articles/8496034283533-How-to-get-started-with-Microsoft-To-Do-on-Zapier) |
| IFTTT | Its Microsoft To Do integration offers task creation and a task-completed trigger, with examples involving other apps. Applets are separate automations, not built-in two-way sync. [Integration](https://ifttt.com/microsoft_todo) |

For example: **Galaxy Reminder ↔ Microsoft To Do ↔ To Do Connect ↔ your AI assistant**. Choose the synchronized list when asking AI to add or inspect reminders. Microsoft currently documents one synchronized To Do list at a time in Samsung Reminder; its Samsung Cloud-only reminders and some features (including To Do steps) are not shared in the same way. Check the official setup for device/app requirements and feature differences. External synchronization may take time.

The current built-in Microsoft sign-in targets personal accounts. Third-party automation services require their own accounts/consent and may have paid plans or limits; they are optional and send data through those services. Their integrations have been checked against provider documentation, not end-to-end tested by this project. Connecting this plugin alone does not enable them or extend its API permissions.

## Platforms and privacy

Packages target Windows, macOS and Linux on x64 and ARM64. macOS/Linux need a working OS key store or an explicitly configured encryption key. Binaries are not currently code-signed or notarized. Docker is optional.

Credentials are encrypted in your user data directory, separately from the program and repository. Task results are sent to the AI application you use; local execution does not mean the AI service never receives task content.

[Installation](docs/installation.md) · [Runtime and Docker](docs/runtime.md) · [Development](docs/development.md) · [Packaging](docs/releases.md) · [Verification](docs/ci.md)

[Support](docs/support.md) · [Privacy](docs/privacy.md) · [Terms](docs/terms.md)

[MIT License](LICENSE). Microsoft To Do and Google Tasks belong to their respective owners. This is an independent community project, not an official Microsoft or Google product.
