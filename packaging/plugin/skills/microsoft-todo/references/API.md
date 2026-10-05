# API coverage — Microsoft Graph To Do v1.0

To Do Connect의 Microsoft provider API 참고자료. 모든 도구는 `list_connections`에서 선택한 Microsoft 연결의 `connection_id`를 필수로 받는다. 이 문서의 43개 provider 도구 외에 `list_connections`, `check_connection`, `disconnect_connection` 3개 관리 도구가 있다.

기준: 2026-10-04에 확인한 공식 v1.0 To Do 리소스 문서. 38개 직접 API 도구 + 5개 조합/업로드 도구 = 43개. 정식 To Do API 범위이며 Microsoft Graph 전체 또는 To Do 앱의 모든 UI 기능을 뜻하지 않는다. beta 및 폐기된 Outlook Tasks API는 제외한다.

## 직접 API 도구 38개

경로 앞에는 `https://graph.microsoft.com/v1.0`가 붙는다. `{list}`, `{task}`, `{item}`은 각각 `list_id`, `task_id`, `item_id`다. 사용자 권한 범위만 쓰므로 `/me`로 한정한다. 별도 Graph 사용자 ID를 지정하는 도구는 제공하지 않는다.

| 도구 | HTTP | 경로 |
|---|---|---|
| `microsoft_list_lists` | GET | `/me/todo/lists` |
| `microsoft_create_list` | POST | `/me/todo/lists` |
| `microsoft_get_list` | GET | `/me/todo/lists/{list}` |
| `microsoft_update_list` | PATCH | `/me/todo/lists/{list}` |
| `microsoft_delete_list` | DELETE | `/me/todo/lists/{list}` |
| `microsoft_delta_lists` | GET | `/me/todo/lists/delta` |
| `microsoft_list_tasks` | GET | `/me/todo/lists/{list}/tasks` |
| `microsoft_create_task` | POST | `/me/todo/lists/{list}/tasks` |
| `microsoft_get_task` | GET | `/me/todo/lists/{list}/tasks/{task}` |
| `microsoft_update_task` | PATCH | `/me/todo/lists/{list}/tasks/{task}` |
| `microsoft_delete_task` | DELETE | `/me/todo/lists/{list}/tasks/{task}` |
| `microsoft_delta_tasks` | GET | `/me/todo/lists/{list}/tasks/delta` |
| `microsoft_list_checklists` | GET | `/me/todo/lists/{list}/tasks/{task}/checklistItems` |
| `microsoft_create_checklist` | POST | `/me/todo/lists/{list}/tasks/{task}/checklistItems` |
| `microsoft_get_checklist` | GET | `/me/todo/lists/{list}/tasks/{task}/checklistItems/{item}` |
| `microsoft_delete_checklist` | DELETE | `/me/todo/lists/{list}/tasks/{task}/checklistItems/{item}` |
| `microsoft_update_checklist` | PATCH | `/me/todo/lists/{list}/tasks/{task}/checklistItems/{item}` |
| `microsoft_list_links` | GET | `/me/todo/lists/{list}/tasks/{task}/linkedResources` |
| `microsoft_create_link` | POST | `/me/todo/lists/{list}/tasks/{task}/linkedResources` |
| `microsoft_get_link` | GET | `/me/todo/lists/{list}/tasks/{task}/linkedResources/{item}` |
| `microsoft_delete_link` | DELETE | `/me/todo/lists/{list}/tasks/{task}/linkedResources/{item}` |
| `microsoft_update_link` | PATCH | `/me/todo/lists/{list}/tasks/{task}/linkedResources/{item}` |
| `microsoft_list_attachments` | GET | `/me/todo/lists/{list}/tasks/{task}/attachments` |
| `microsoft_create_attachment` | POST | `/me/todo/lists/{list}/tasks/{task}/attachments` |
| `microsoft_get_attachment` | GET | `/me/todo/lists/{list}/tasks/{task}/attachments/{item}` |
| `microsoft_delete_attachment` | DELETE | `/me/todo/lists/{list}/tasks/{task}/attachments/{item}` |
| `microsoft_list_list_extension` | GET | `/me/todo/lists/{list}/extensions` |
| `microsoft_create_list_extension` | POST | `/me/todo/lists/{list}/extensions` |
| `microsoft_get_list_extension` | GET | `/me/todo/lists/{list}/extensions/{item}` |
| `microsoft_update_list_extension` | PATCH | `/me/todo/lists/{list}/extensions/{item}` |
| `microsoft_delete_list_extension` | DELETE | `/me/todo/lists/{list}/extensions/{item}` |
| `microsoft_list_task_extension` | GET | `/me/todo/lists/{list}/tasks/{task}/extensions` |
| `microsoft_create_task_extension` | POST | `/me/todo/lists/{list}/tasks/{task}/extensions` |
| `microsoft_get_task_extension` | GET | `/me/todo/lists/{list}/tasks/{task}/extensions/{item}` |
| `microsoft_update_task_extension` | PATCH | `/me/todo/lists/{list}/tasks/{task}/extensions/{item}` |
| `microsoft_delete_task_extension` | DELETE | `/me/todo/lists/{list}/tasks/{task}/extensions/{item}` |
| `microsoft_get_attachment_content` | GET | `/me/todo/lists/{list}/tasks/{task}/attachments/{item}/$value` |
| `microsoft_create_upload_session` | POST | `/me/todo/lists/{list}/tasks/{task}/attachments/createUploadSession` |

## 조합/전송 도구 5개

| 도구 | 입력/동작 |
|---|---|
| `microsoft_copy_task` | list_id, task_id, target_list_id. 전체 지원 내용 복사·재조회 검증. |
| `microsoft_move_task` | 위 입력 + confirm=true. 검증 뒤 원본 재조회·삭제. 비원자적. |
| `microsoft_duplicate_list` | list_id, name. 새 목록 및 작업들 복제. 공유 설정 제외. |
| `microsoft_upload_chunk` | session_url, content_bytes(base64), offset, total. uploadUrl/content에 인증된 PUT. |
| `microsoft_cancel_upload` | session_url, confirm=true. uploadUrl에 DELETE. |

## 호출 계약

- `body`에는 해당 Microsoft Graph 메서드의 JSON을 넣는다. API 응답도 JSON 구조로 반환한다. `httpStatus`와 업로드 완료 시 `location`은 전송 확인용 추가 필드다.
- 필수 ID 누락·경로 문법·다른 origin URL·다른 컬렉션 cursor·미확인 삭제는 로컬에서 거부한다. 개별 Graph body의 지원 속성·값·업무 규칙은 Graph가 검증한다. 요청 본문 전체를 완전한 Graph JSON Schema로 재구현한 것은 아니다.
- 읽기 도구의 `query`는 `$select`, `$expand`, `$top`, `$filter`, `$orderby` 등이다. 각 메서드가 지원하는 조합만 사용한다. 모든 메서드에서 모든 쿼리가 지원된다는 뜻은 아니다.
- 한 페이지씩 반환한다. 다음 페이지/변경 추적은 같은 도구의 `cursor`에 응답의 `@odata.nextLink`/`@odata.deltaLink`를 그대로 넣는다. cursor와 query는 함께 쓰지 않는다.
- 일반 할 일 수정은 title, body, status, importance, dates, reminder, recurrence, categories를 Graph body로 전달한다. 완료/재개는 status 값이다. 일시 정보는 `{ "dateTime": "2026-10-12T09:00:00", "timeZone": "Asia/Seoul" }` 형태이며 실제 timezone 수용 여부는 Graph에서 확인한다. API 문서상 작업 update의 body는 HTML이므로 text를 임의로 그대로 PATCH하지 않는다.
- 작은 첨부는 `microsoft_create_attachment`의 body에 `@odata.type`, name, contentBytes, contentType을 넣는다. 3 MB 미만 단일 POST, 큰 파일은 create_upload_session 및 upload_chunk를 사용한다. 파일당 최대 25 MB, chunk는 4 MiB 미만이다. 반환된 nextExpectedRanges를 따른다.
- content_bytes/첨부 body의 contentBytes는 base64이며 파일 경로를 주는 방식이 아니다. GET attachment/$value는 contentBytes와 contentType으로 반환한다. 대용량 첨부는 모델 대화창보다 로컬 CLI를 이용하는 것이 적합하다.
- open extension은 목록과 작업에 붙이는 사용자 정의 데이터다. 생성 시 `@odata.type: microsoft.graph.openTypeExtension`, `extensionName` 및 추가 필드를 넣는다. task ID를 사용자 정의 extension ID와 혼동하지 않는다.
- list 읽기의 isShared는 조회할 수 있지만 공유 초대·권한 복제 API를 구현한 것은 아니다. 앱 UI의 My Day, 그룹, 테마, 수동 정렬 등은 이 v1.0 문서 목록에서 계약을 확인하지 못했으므로 지원을 주장하지 않는다.
- `$batch`는 Graph 공통 기능으로 이번 To Do 범위에 포함하지 않는다. 여러 작업 처리는 각각의 도구 호출로 수행한다. 전체 실패 시 롤백·트랜잭션은 제공하지 않는다.

## 근거

- [To Do API overview](https://learn.microsoft.com/en-us/graph/api/resources/todo-overview?view=graph-rest-1.0)
- [todoTaskList](https://learn.microsoft.com/en-us/graph/api/resources/todotasklist?view=graph-rest-1.0)
- [todoTask](https://learn.microsoft.com/en-us/graph/api/resources/todotask?view=graph-rest-1.0)
- [checklistItem](https://learn.microsoft.com/en-us/graph/api/resources/checklistitem?view=graph-rest-1.0)
- [linkedResource](https://learn.microsoft.com/en-us/graph/api/resources/linkedresource?view=graph-rest-1.0)
- [taskFileAttachment](https://learn.microsoft.com/en-us/graph/api/resources/taskfileattachment?view=graph-rest-1.0)
- [Create upload session (v1.0)](https://learn.microsoft.com/en-us/graph/api/taskfileattachment-createuploadsession?view=graph-rest-1.0)
- [Upload chunk protocol](https://learn.microsoft.com/en-us/graph/todo-attachments) — walkthrough에 beta 예시가 남아 있으나 session 생성은 v1.0 개별 문서 기준. 실계정 전송은 미검증.
- [Open extensions](https://learn.microsoft.com/en-us/graph/api/resources/opentypeextension?view=graph-rest-1.0)
- [List delta](https://learn.microsoft.com/en-us/graph/api/todotasklist-delta?view=graph-rest-1.0), [Task delta](https://learn.microsoft.com/en-us/graph/api/todotask-delta?view=graph-rest-1.0)

## 검증 구분

로컬 테스트는 문서 기반 요청과 모의 응답을 검증한다. 실제 Microsoft Graph의 정규화·권한·서비스 상태를 대체하지 않는다. 43개 도구의 등록은 검증했지만 실계정 API별 완료를 의미하지 않는다. 앱 등록 및 사용자 로그인 뒤 별도 동의한 테스트 목록으로 실제 검증해야 한다.
