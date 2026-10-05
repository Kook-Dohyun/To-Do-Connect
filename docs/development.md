# 개발 안내

## 저장소와 Go 프로젝트의 경계

저장소 루트 `todo-mcp/`는 제품 전체를 담는다. Go CLI·MCP 서버는 `server/` 하위 프로젝트이며 플러그인·Skill·Docker 구성과 문서는 밖에 둔다.

```text
todo-mcp/
├── server/
│   ├── go.mod / go.sum             런타임 모듈·의존성
│   ├── cmd/todo-connect/           CLI·MCP 진입점과 실행 테스트
│   └── internal/app/               인증·연결·API·온보딩 구현과 테스트
├── tools/
│   ├── go.mod                     표준 라이브러리만 쓰는 배포 도구 모듈
│   ├── package-archive/            Unix 실행 권한을 보존하는 ZIP 제작
│   └── license-notices/            실제 바이너리의 의존성 고지 수집
├── packaging/
│   ├── plugin/                    매니페스트·MCP 설정·provider별 Skills
│   └── docker/Dockerfile
├── scripts/                       빌드·패키징·설치 실행 절차
├── docs/                          공개 사용·개발 안내
├── config/                        선택적 로컬 자료, Git 제외
├── bin/                           개발 실행 파일, Git 제외
├── dist/                          배포 조립 결과, Git 제외
├── README.md
└── LICENSE
```

`server/cmd/todo-connect/main.go`는 `internal/app.Run`을 호출한다. CLI와 MCP는 같은 프로그램·연결 저장소를 사용한다. Microsoft/Google을 별도 프로그램으로 나누지 않는다.

`tools/`는 기존 두 개발 도구를 제품 실행 코드에서 분리한 것이다. 외부 의존성이나 서버 구현을 가져오지 않으므로 별도 소형 모듈로 둔다. 모듈 사이 소스 참조가 없어 go.work는 필요 없다. 라이선스 수집은 런타임 의존성 캐시를 사용하도록 패키징 스크립트가 server에서 실행한다.

런타임 모듈명은 `github.com/Kook-Dohyun/To-Do-Connect`, 제품·실행 이름은 `To Do Connect` / `todo-connect`로 유지한다. 소스 이동은 사용자 데이터 경로나 배포 ZIP 내부 구조를 바꾸지 않는다. 라이선스는 MIT다.

`config/`는 사용자 설치 요구사항이 아니다. 동봉 Skill과 MCP 온보딩 도구가 현재 상태를 확인하고 필요한 설정을 안내한다. 내려받은 JSON은 경로만 전달하며 내용을 채팅·Git·배포물에 넣지 않는다.

## 개발자 빌드

일반 사용자는 빌드된 프로그램을 설치한다. 아래는 Go 1.25.13 이상을 사용하는 기여자용 명령이며 **저장소 루트** 기준이다.

```powershell
./scripts/build.ps1
./bin/todo-connect.exe catalog
go -C server build -o ../bin/todo-connect.exe ./cmd/todo-connect
go -C server test ./... -count=1
go -C server vet ./...
go -C tools test ./... -count=1
go -C tools vet ./...
```

macOS/Linux 출력명은 `../bin/todo-connect`다. `go -C server`는 server로 이동한 뒤 실행하므로 출력 경로도 그 기준이다. `./scripts/build-targets.ps1`은 6개 OS·CPU 실행 파일을, `./scripts/package-release.ps1 -Version VERSION`은 실행 파일을 포함한 플러그인 ZIP을 만든다. 스크립트는 호출 위치에 의존하지 않는다. Docker 컨텍스트는 저장소 루트이며 허용 목록으로 server·tools·LICENSE·Dockerfile만 전달한다.

## 현재 CLI

| 명령 | 동작 |
| --- | --- |
| `version` / `--version` | 실행 파일의 빌드 버전. 릴리스·Docker는 빌드 시 주입하며 일반 소스 빌드는 `dev` |
| `catalog` | MCP와 동일한 63개 도구 및 입력 스키마 출력: Microsoft 43개 + Google Tasks 14개 + 연결 관리 3개 + Google 온보딩 3개 |
| `connections list` | 로컬 연결 ID·provider·라벨·캐시 존재 상태; 원격 접근 없음 |
| `connections add --provider microsoft --id ID --label NAME` | 계정별 연결 생성; 로그인은 별도 |
| `connections add --provider google --id ID --label NAME --credentials PATH` | 사용자 소유 Google Desktop OAuth JSON을 암호화해 가져오기; 로그인은 별도 |
| `login ID` | Microsoft 브라우저 PKCE 로그인; 앱 등록의 desktop `http://localhost` 리디렉션 사용 |
| `login GOOGLE_ID` | Google 시스템 브라우저 PKCE 로그인; 임시 loopback 콜백, Tasks+openid 권한 |
| `login ID --device` | 사용자가 명시적으로 선택하는 터미널 device-code 로그인 |
| `login ID --reauthenticate` | 같은 계정의 명시적 재인증; 다른 계정은 별도 연결로 추가 |
| `login GOOGLE_ID --headless --container-port 8765` | Docker 호스트의 브라우저에서 로그인; 호스트 127.0.0.1의 동일 포트를 명시적으로 게시 |
| `keygen --out ABSOLUTE_PATH` | 별도 키 파일 방식에 사용할 32바이트 암호화 키 생성; 기존 파일 덮어쓰기 없음 |
| `check ID` | 선택한 provider의 목록 API를 읽어 연결 확인; 할 일 변경 없음 |
| `serve` | stdio MCP 실행; 수신 포트·공개 서버 없음 |
| `call TOOL` | 표준 입력 JSON으로 도구 호출 |
| `connections remove --id ID --confirm` | 선택한 연결 설정·암호화 캐시 삭제; 원격 할 일·동의는 유지 |
| `connections import-microsoft --id ID --from PATH` | 기존 단일 계정 암호화 캐시를 연결로 복사; 원본 유지, 자동 실행하지 않음 |

```powershell
'{"connection_id":"personal"}' | ./bin/todo-connect.exe call microsoft_list_lists
```

Client ID는 현재 Microsoft 앱의 공개 식별자로 코드에 포함돼 있다. `TODO_CLIENT_ID` 환경 변수는 필요하지 않다. 비밀번호·토큰·device code를 채팅, 로그, Git에 기록하지 않는다.

새 연결은 Windows에서 `os.UserCacheDir()/todo-connect`, 다른 OS에서 `os.UserConfigDir()/todo-connect`에 저장한다. `<연결ID>.json`은 비밀정보 없는 설정, `.secret`은 암호화 세션, Google의 `.client`는 암호화한 Desktop OAuth 설정, `.lock`은 프로세스 조정용 파일이다. Windows 기본값은 DPAPI, macOS/Linux는 OS 키 저장소의 키로 AES-GCM 암호화, Docker는 명시적 키 파일 방식이다. 캐시가 존재한다는 상태와 실제 API 접근 성공은 구분한다. `TODO_CONNECT_DATA_DIR`에 절대 경로를 지정하면 CLI/MCP가 같은 위치를 사용하게 할 수 있다. 소스, 배포 패키지, Vault 동기화 경로는 사용하지 않는다. 자세한 실행·키 보관 절차는 [런타임 안내](runtime.md)를 따른다.

기존 `os.UserCacheDir()/microsoft-todo-mcp`의 단일 계정 캐시는 새 연결에서 자동 사용하지 않는다. 가져오기는 사용자가 선택한 원본을 로컬 검증한 뒤 암호화 상태로 복사한다. Windows 패키지 앱은 AppData를 리디렉션할 수 있으므로 논리 경로만으로 탐색기에서 보이는 물리 위치를 단정하지 않는다.

CLI와 도구 계약이 바뀌었다. 이전의 인자 없는 `login/status/logout`과 provider 접두사 없는 도구 호출은 더 이상 사용하지 않는다. 모든 Microsoft 데이터 도구는 `microsoft_` 접두사와 명시적 `connection_id`를 사용한다. 기본 계정을 임의로 선택하지 않는다. [동봉 연결 지침](../packaging/plugin/skills/microsoft-todo/references/connections.md)에 재사용·추가·해제 절차와 현재 미검증 사항을 정리했다.

## 플러그인 개발본

Google 호출 예시: `{"connection_id":"google-personal"}`을 `call google_list_lists`의 표준 입력으로 전달한다. `google_` 도구는 Google 연결, `microsoft_` 도구는 Microsoft 연결만 받는다. Microsoft·Google 연결은 함께 유지되며 provider 간 동기화·복사 기능은 제공하지 않는다. [Google 동봉 지침](../packaging/plugin/skills/google-tasks/SKILL.md)에 Tasks의 날짜·이동·완료 항목 숨김 의미와 연결 절차를 구분했다.

`packaging/plugin`은 패키지 소스다. 소스의 `mcp.json`은 개발용 PATH 명령이고, 릴리스 조립 시 `./bin/실행파일`로 바뀐다. `scripts/package-release.ps1`이 해당 OS 실행 파일과 스킬을 포함한 ZIP을 만든다. ZIP 업로드는 플랫폼에 맞는 파일을 선택하고, 설치기는 OS·CPU를 판별한다. [릴리스 검증 안내](releases.md)를 따른다.

프로그램 배포 단계에서 빌드된 파일과 검증된 OS 선택·실행 구성을 묶어 `dist/`에 조립한다. 인증정보는 포함하지 않는다. `dist/`는 생성 결과이며 Git에서 제외한다.

## 테스트와 정리

- 테스트용 Graph 서버는 메모리에서 실행하고 종료한다.
- Google API와 OAuth도 로컬 가짜 서버로 검증한다. 실제 Google 계정·프로젝트를 생성하거나 권한 동의를 수행하지 않는다.
- 실행 파일과 DPAPI 테스트의 임시 결과는 `t.TempDir()`로 정리한다.
- 복수 가짜 계정의 캐시 분리·재시작·선택적 가져오기·개별 해제와 별도 프로세스 잠금을 검증한다.
- 실제 실행 파일로 연결을 만들고 stdio MCP로 해제한 뒤 새 CLI 프로세스에서 남은 연결을 확인한다.
- 회귀 테스트 소스는 보존한다. 실제 계정 데이터를 mock 정리 대상으로 취급하지 않는다.
- 자동 테스트는 사용자 할 일을 생성·변경·삭제하지 않는다.
- 교차 빌드는 해당 OS의 OAuth·보안 저장소·설치 UX 검증을 대신하지 않는다.

Docker 검증은 [런타임 안내](runtime.md)의 명령을 따른다. 테스트는 명시적으로 지정한 로컬 이미지와 실행별 라벨이 있는 임시 볼륨만 사용하고 정리한다. Linux race 검증은 Dockerfile의 `test` 단계에서 수행한다. 해당 테스트 이미지에는 C 컴파일러가 있지만 최종 `runtime` 이미지에는 없다.
