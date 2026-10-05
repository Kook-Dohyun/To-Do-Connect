# Privacy

Effective date: October 5, 2026

To Do Connect is an open-source local task connector maintained by [Kook-Dohyun](https://github.com/Kook-Dohyun). This notice describes the project, not modified versions or the AI applications using it.

## Data flow

The developer operates no task-data relay, analytics or telemetry service for To Do Connect and does not receive connected-account credentials.

The local program sends authenticated requests directly to the selected provider. Requests and responses can contain list/task names, notes, identifiers, dates, completion state, checklist steps or subtask relationships. Authentication uses the provider's sign-in and token services; passwords stay in the provider UI.

Tool results go to your AI application. Its own settings and policies govern prompts, task content, connection labels, results and errors. **Local execution does not mean your task data stays exclusively on your computer.** Microsoft, Google, your AI application and GitHub have their own policies.

## Local storage

Connection IDs, provider names, labels and OAuth client IDs are stored as local configuration. Tokens and imported Google Desktop OAuth client settings are encrypted in the user's separate application-data directory.

Windows uses DPAPI by default. macOS/Linux use an OS-held encryption key; explicit key-file storage is also supported. See [runtime and storage](runtime.md). The original downloaded Google JSON is not deleted on import.

The program does not maintain a separate persistent task database. Providers remain the source of task data; your AI host, saved files and backups may retain their own copies.

## Removal

Disconnecting an account removes its local configuration, encrypted token cache and imported client configuration. It does not delete remote tasks, revoke provider consent, erase original downloads/backups or clear AI chat history. Lock files and shared key-store entries may remain.

Uninstalling the plugin alone does not remove separately stored connections. Revoke consent through the provider's account settings and manage retained conversations through your AI application.

## Support

[GitHub Issues](https://github.com/Kook-Dohyun/To-Do-Connect/issues) is public. Do not post passwords, tokens, OAuth JSON, private tasks or unredacted screenshots. The project does not request credential-cache uploads.

Questions can be raised through [support](support.md) without private account information. Material changes to this notice will update its effective date.
