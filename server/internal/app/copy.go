package app

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
)

var taskFields = []string{"title", "body", "categories", "completedDateTime", "dueDateTime", "importance", "isReminderOn", "recurrence", "reminderDateTime", "startDateTime", "status"}
var childKinds = []string{"checklistItems", "linkedResources", "extensions", "attachments"}

type snapshot struct {
	Task     object
	Children map[string][]object
}

func pick(v object, keys ...string) object {
	r := object{}
	for _, k := range keys {
		if x, ok := v[k]; ok {
			r[k] = x
		}
	}
	return r
}
func childBody(kind string, v object) object {
	switch kind {
	case "checklistItems":
		return pick(v, "displayName", "isChecked")
	case "linkedResources":
		return pick(v, "webUrl", "applicationName", "displayName", "externalId")
	case "attachments":
		r := pick(v, "name", "contentType", "contentBytes")
		r["@odata.type"] = "#microsoft.graph.taskFileAttachment"
		return r
	default:
		r := object{}
		for k, x := range v {
			if k != "id" && k != "httpStatus" && !strings.HasPrefix(k, "@odata.") {
				r[k] = x
			}
		}
		r["@odata.type"] = "microsoft.graph.openTypeExtension"
		return r
	}
}
func (g *graph) snapshot(ctx context.Context, path string) (snapshot, error) {
	s := snapshot{Children: map[string][]object{}}
	r, e := g.json(ctx, http.MethodGet, path, nil)
	if e != nil {
		return s, e
	}
	s.Task = r
	for _, kind := range childKinds {
		items, e := g.all(ctx, path+"/"+kind)
		if e != nil {
			return s, e
		}
		for _, v := range items {
			if kind == "attachments" {
				id, ok := v["id"].(string)
				if !ok {
					return s, errors.New("attachment missing ID")
				}
				id, e = segment(id)
				if e != nil {
					return s, e
				}
				content, e := g.json(ctx, http.MethodGet, path+"/attachments/"+id+"/$value", nil)
				if e != nil {
					return s, e
				}
				v["contentBytes"] = content["contentBytes"]
			}
			s.Children[kind] = append(s.Children[kind], v)
		}
	}
	return s, nil
}
func normalizedChildren(kind string, items []object) []string {
	r := []string{}
	for _, v := range items {
		r = append(r, encoded(childBody(kind, v)))
	}
	slices.Sort(r)
	return r
}
func verifySnapshot(source, target snapshot) error {
	if encoded(pick(source.Task, taskFields...)) != encoded(pick(target.Task, taskFields...)) {
		return errors.New("copied task properties do not match")
	}
	for _, kind := range childKinds {
		if !slices.Equal(normalizedChildren(kind, source.Children[kind]), normalizedChildren(kind, target.Children[kind])) {
			return fmt.Errorf("copied %s do not match", kind)
		}
	}
	return nil
}
func (g *graph) attach(ctx context.Context, path string, body object) (object, error) {
	b, e := base64.StdEncoding.DecodeString(fmt.Sprint(body["contentBytes"]))
	if e != nil {
		return nil, e
	}
	if len(b) > maxFile {
		return nil, errors.New("attachment exceeds 25 MiB")
	}
	if len(b) < 3*1024*1024 {
		return g.json(ctx, http.MethodPost, path+"/attachments", body)
	}
	session, e := g.json(ctx, http.MethodPost, path+"/attachments/createUploadSession", object{"attachmentInfo": object{"attachmentType": "file", "name": body["name"], "size": len(b)}})
	if e != nil {
		return nil, e
	}
	u, ok := session["uploadUrl"].(string)
	if !ok {
		return nil, errors.New("upload session omitted uploadUrl")
	}
	var out object
	for start := 0; start < len(b); {
		end := min(start+2*1024*1024, len(b))
		out, e = g.uploadChunk(ctx, input{SessionURL: u, ContentBytes: base64.StdEncoding.EncodeToString(b[start:end]), Offset: int64(start), Total: int64(len(b))})
		if e != nil {
			return nil, e
		}
		start = end
	}
	if out["httpStatus"] != http.StatusCreated {
		return nil, errors.New("upload did not confirm completion")
	}
	return out, nil
}
func (g *graph) copySnapshot(ctx context.Context, s snapshot, destination string) (object, error) {
	created, e := g.json(ctx, http.MethodPost, destination+"/tasks", pick(s.Task, taskFields...))
	if e != nil {
		return nil, e
	}
	id, ok := created["id"].(string)
	if !ok {
		return nil, errors.New("created task missing ID; inspect destination before retrying")
	}
	seg, e := segment(id)
	if e != nil {
		return nil, e
	}
	path := destination + "/tasks/" + seg
	partial := func(err error) (object, error) {
		return object{"complete": false, "destination_task_id": id, "source_preserved": true, "error": err.Error(), "retry": "Inspect the partial destination; do not blindly repeat copy."}, nil
	}
	for _, kind := range childKinds {
		for _, v := range s.Children[kind] {
			body := childBody(kind, v)
			if kind == "attachments" {
				_, e = g.attach(ctx, path, body)
			} else {
				_, e = g.json(ctx, http.MethodPost, path+"/"+kind, body)
			}
			if e != nil {
				return partial(e)
			}
		}
	}
	target, e := g.snapshot(ctx, path)
	if e != nil {
		return partial(e)
	}
	if e = verifySnapshot(s, target); e != nil {
		return partial(e)
	}
	return object{"complete": true, "destination_task_id": id, "source_preserved": true, "note": "New ID and timestamps; sharing and UI-only metadata are not cloned."}, nil
}
func (g *graph) copyTask(ctx context.Context, in input) (object, error) {
	src, e := taskPath(in.ListID, in.TaskID)
	if e != nil {
		return nil, e
	}
	dst, e := listPath(in.TargetListID)
	if e != nil {
		return nil, e
	}
	s, e := g.snapshot(ctx, src)
	if e != nil {
		return nil, e
	}
	return g.copySnapshot(ctx, s, dst)
}
func (g *graph) moveTask(ctx context.Context, in input) (object, error) {
	if !in.Confirm {
		return nil, errors.New("move removes the source; confirm=true requires user authorization")
	}
	if in.ListID == in.TargetListID {
		return nil, errors.New("source and target list are identical")
	}
	src, e := taskPath(in.ListID, in.TaskID)
	if e != nil {
		return nil, e
	}
	dst, e := listPath(in.TargetListID)
	if e != nil {
		return nil, e
	}
	before, e := g.snapshot(ctx, src)
	if e != nil {
		return nil, e
	}
	out, e := g.copySnapshot(ctx, before, dst)
	if e != nil || out["complete"] != true {
		return out, e
	}
	after, e := g.snapshot(ctx, src)
	if e == nil && encoded(before) != encoded(after) {
		e = errors.New("source changed during copy; both tasks retained")
	}
	if e != nil {
		out["complete"] = false
		out["error"] = e.Error()
		return out, nil
	}
	// There is no atomic Graph To Do move contract. A final concurrent edit may still race;
	// the tool description and README explicitly require avoiding concurrent editing.
	_, e = g.json(ctx, http.MethodDelete, src, nil)
	if e != nil {
		out["complete"] = false
		out["source_preserved"] = nil
		out["error"] = e.Error()
		out["source_status"] = "unknown: re-read source before retrying"
		return out, nil
	}
	out["source_preserved"] = false
	return out, nil
}
func (g *graph) duplicateList(ctx context.Context, in input) (object, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, errors.New("new list name is required")
	}
	src, e := listPath(in.ListID)
	if e != nil {
		return nil, e
	}
	tasks, e := g.all(ctx, src+"/tasks")
	if e != nil {
		return nil, e
	}
	extensions, e := g.all(ctx, src+"/extensions")
	if e != nil {
		return nil, e
	}
	target, e := g.json(ctx, http.MethodPost, listsPath, object{"displayName": in.Name})
	if e != nil {
		return nil, e
	}
	id, ok := target["id"].(string)
	if !ok {
		return nil, errors.New("created list missing ID; inspect before retry")
	}
	dst, e := listPath(id)
	if e != nil {
		return nil, e
	}
	out := object{"complete": false, "destination_list_id": id, "tasks": []object{}, "source_preserved": true}
	partial := func(err error) (object, error) { out["error"] = err.Error(); return out, nil }
	for _, v := range extensions {
		if _, e = g.json(ctx, http.MethodPost, dst+"/extensions", childBody("extensions", v)); e != nil {
			return partial(e)
		}
	}
	actual, e := g.all(ctx, dst+"/extensions")
	if e != nil {
		return partial(e)
	}
	if !slices.Equal(normalizedChildren("extensions", extensions), normalizedChildren("extensions", actual)) {
		return partial(errors.New("list extensions verification failed"))
	}
	for _, v := range tasks {
		tid, ok := v["id"].(string)
		if !ok {
			return partial(errors.New("source task missing ID"))
		}
		copied, e := g.copyTask(ctx, input{ListID: in.ListID, TaskID: tid, TargetListID: id})
		if e != nil {
			return partial(e)
		}
		out["tasks"] = append(out["tasks"].([]object), copied)
		if copied["complete"] != true {
			return partial(errors.New("task copy incomplete; inspect returned task IDs"))
		}
	}
	out["complete"] = true
	out["note"] = "List membership is captured at start. Sharing permissions and UI-only metadata are not copied."
	return out, nil
}
