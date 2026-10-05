# Google Tasks tool fields

## Lists

| Action | Tool | Body |
| --- | --- | --- |
| Read | `google_list_lists` | None |
| Create | `google_create_list` | `{"title":"Server-room checks"}` |
| Rename | `google_patch_list` | `{"title":"New name"}` |
| Delete | `google_delete_list` | None; `confirm: true` |

All tools require `connection_id`. Rename/delete also require `list_id`.

## Tasks

Use `google_list_tasks`, `google_create_task`, `google_patch_task` and `google_delete_task` with `connection_id` and `list_id`. Patch/delete also require `task_id`. Delete requires `confirm: true`.

Example create body:

```json
{
  "title": "1. Check server identity",
  "notes": "Record the result for the evidence report."
}
```

Partial update bodies:

- Complete: `{"status":"completed"}`.
- Reopen: `{"status":"needsAction"}`.
- Due date: `{"due":"2026-12-01T00:00:00Z"}`. The API retains only the date, not a time-of-day deadline.
- Notes are plain text.

## Subtasks and ordering

Create a subtask with `google_create_task` and `query: {"parent":"PARENT_TASK_ID"}`. Optional `previous` positions it after a sibling.

Use `google_move_task` with `connection_id`, `list_id`, `task_id`, `confirm: true` and the intended `query.parent`, `query.previous` or `query.destinationTasklist`. IDs must belong to the selected account. Values in `query` are strings.

Assigned and repeating tasks have provider restrictions. Report a rejected move instead of approximating it by creating and deleting tasks. Deleting an assigned task can affect its originating Docs/Chat task.

## Read all pages

Collection tools return one page. Put `nextPageToken` into the next call's `query.pageToken` with the same connection and list.

When checking completed work, use `query: {"showCompleted":"true","showHidden":"true"}`; include `showAssigned` when assigned tasks are in scope. The listing's `parent` and `position` fields describe hierarchy and ordering.

## Provider reference

[Lists](https://developers.google.com/workspace/tasks/reference/rest/v1/tasklists) · [Tasks](https://developers.google.com/workspace/tasks/reference/rest/v1/tasks) · [Move](https://developers.google.com/workspace/tasks/reference/rest/v1/tasks/move)
