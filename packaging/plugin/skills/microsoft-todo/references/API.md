# Microsoft To Do tool fields

## Lists

| Action | Tool | Body |
| --- | --- | --- |
| Read | `microsoft_list_lists` | None |
| Create | `microsoft_create_list` | `{"displayName":"Server-room checks"}` |
| Rename | `microsoft_update_list` | `{"displayName":"New name"}` |
| Delete | `microsoft_delete_list` | None; `confirm: true` |

All tools require `connection_id`. Rename/delete also require `list_id`.

## Tasks

Use `microsoft_list_tasks`, `microsoft_create_task`, `microsoft_update_task` and `microsoft_delete_task` with `connection_id` and `list_id`. Update/delete also require `task_id`. Delete requires `confirm: true`.

Example create body:

```json
{
  "title": "1. Check server identity",
  "body": {"contentType": "text", "content": "Record the result for the evidence report."},
  "importance": "normal"
}
```

Partial update bodies:

- Complete: `{"status":"completed"}`.
- Reopen: `{"status":"notStarted"}`.
- Due date: `{"dueDateTime":{"dateTime":"2026-12-01T09:00:00","timeZone":"UTC"}}`. Use the user's intended timezone; this is only an example.
- Reminders use `isReminderOn` and `reminderDateTime`.

The tool's `body` is the Graph resource body; task notes are the nested `body` property. Use supported Graph fields rather than inventing generic fields.

## Checklist steps

Use `microsoft_list_checklists`, `microsoft_create_checklist`, `microsoft_update_checklist` and `microsoft_delete_checklist`.

All require `connection_id`, `list_id` and `task_id`. Update/delete also require `item_id`; deletion requires `confirm: true`.

Create: `{"displayName":"Record hostname","isChecked":false}`.
Complete/uncheck: `{"isChecked":true}` / `{"isChecked":false}`.

## Read all pages

Collection tools return one Graph page. Follow the exact returned `@odata.nextLink` in `cursor`, with the same connection and parent IDs. Omit `query` on cursor calls. Initial queries can use supported OData options such as `$top`, `$select` and `$filter`; endpoint support varies.

## Completion time window

Resolve “today at 3 PM” to an explicit date and user time zone; use the current time as the upper bound for “since.” Read all pages of the selected lists with `microsoft_list_tasks`. Compare `completedDateTime.dateTime` interpreted with its `timeZone`, and require `status: completed`. Do not substitute `lastModifiedDateTime`: edits are not completions. Report missing completion times as unknown, not as matches.

For requested steps, read each task's `microsoft_list_checklists` pages (including parents that are still incomplete), or use a complete returned `checklistItems` collection. Require `isChecked: true` and compare `checkedDateTime`, which includes a UTC offset. Do not count a checked step as completion of its parent. Include the parent title in step results.

State the exact time window and boundary interpretation. Deleted items, prior completion/uncompletion cycles and changes between observations cannot be reconstructed from this current-state query. Do not describe the result as an audit log or continuous monitoring.

## Provider reference

[Lists](https://learn.microsoft.com/en-us/graph/api/resources/todotasklist?view=graph-rest-1.0) · [Tasks](https://learn.microsoft.com/en-us/graph/api/resources/todotask?view=graph-rest-1.0) · [Checklist items](https://learn.microsoft.com/en-us/graph/api/resources/checklistitem?view=graph-rest-1.0)
