# 설치 안내

공식 배포처는 [GitHub Releases](https://github.com/Kook-Dohyun/To-Do-Connect/releases)입니다. 현재 공개 배포 준비 중이며, **실제로 게시된 릴리스가 있을 때만** 다운로드 설치를 진행합니다. AI는 존재하지 않는 릴리스나 다운로드 링크를 만들어내면 안 됩니다.

네이티브 패키지 사용자는 Go·Node.js·Docker를 설치하지 않습니다.

## 설치 방식 선택

| 원하는 방식 | 선택할 파일 |
| --- | --- |
| Codex 화면에서 ZIP 업로드 | `todo-connect-plugin-<version>-<os>-<arch>.zip` |
| 설치기·로컬 마켓플레이스·다른 MCP 호스트 | `todo-connect-<version>-<os>-<arch>.zip` 및 설치기 |

두 ZIP은 같은 제품의 다른 설치 진입점입니다. 둘 다 설치할 필요는 없습니다. `plugin-` ZIP은 루트에 `plugin.json`이 있고, 일반 ZIP은 로컬 마켓플레이스와 시작 안내를 함께 담습니다.

OS는 `windows`, `darwin`(macOS), `linux`, CPU는 `amd64`(Intel/AMD x64) 또는 `arm64`입니다. 원격 호스트라면 접속 중인 PC가 아니라 실제 실행 호스트 기준으로 선택합니다.

## Codex 화면에서 설치

1. 해당 플랫폼의 `todo-connect-plugin-…zip`을 다운로드합니다.
2. 릴리스의 해당 파일 SHA-256과 비교합니다.
3. **플러그인 → 추가 → 플러그인 압축 파일 업로드**에서 ZIP을 선택합니다.
4. 표시된 플러그인을 설치하고 대화에서 To Do Connect를 선택합니다.
5. “내 할 일 서비스를 연결해줘”라고 요청합니다.

Microsoft To Do, Google Tasks 또는 둘 다 연결할 수 있습니다. 정상 연결은 재사용하고 추가 계정은 별도 연결로 만듭니다. 연결 검증은 목록 읽기부터 시작하며, 임의의 할 일을 만들지 않습니다.

ZIP 설치와 OpenAI 공개 디렉터리 검색 노출은 별개입니다. 이 안내는 공개 디렉터리 등록 완료를 의미하지 않습니다.

## AI를 통한 GitHub 설치

[README의 설치 요청 프롬프트](../README.md#ai에게-github-설치-요청)를 사용할 수 있습니다. 에이전트는 다음 순서로 진행합니다.

1. 실행 호스트의 OS·CPU와 기존 설치·연결 여부를 확인합니다. 인증정보의 내용은 출력하지 않습니다.
2. 공식 저장소의 실제 릴리스·파일 목록을 조회합니다. 사전 배포 버전이면 사용자에게 알립니다.
3. 버전·설치 경로·호스트 등록 변경을 설명하고 기존 파일을 덮어쓰지 않는 설치 방법을 선택합니다.
4. 설치기와 체크섬을 내려받아 내용을 확인하고 사전 빌드 패키지를 설치합니다.
5. 프로그램 버전과 MCP 도구 조회를 확인한 뒤, 선택한 서비스의 동봉 지침에 따라 연결합니다.

비밀번호·MFA·권한 동의는 사용자가 직접 처리합니다. Google 프로젝트·OAuth 클라이언트를 새로 만들기 전에 기존 설정부터 확인합니다. 인증정보를 채팅이나 Git에 복사하지 않습니다.

### 설치기 명령

`VERSION`을 실제 릴리스 버전으로 바꿉니다. 태그는 `vVERSION`이지만 설치기 인자에는 `v`를 붙이지 않습니다. 내려받아 검토한 설치기가 있는 폴더에서 실행합니다.

Windows PowerShell 5.1 이상:

```powershell
./install.ps1 -Version VERSION -RegisterCodex
```

macOS/Linux:

```sh
sh ./install.sh --version VERSION --register-codex
```

Codex CLI가 이미 있어야 등록할 수 있습니다. 등록 옵션을 빼면 프로그램만 설치하고 Codex 설정은 바꾸지 않습니다. 설치기는 플랫폼에 맞는 일반 ZIP의 체크섬과 프로그램 버전을 검사하며, 자동 로그인하지 않습니다.

Unix에는 `curl`, `unzip`, `sha256sum`(Linux) 또는 `shasum`(macOS)이 필요합니다. 체크섬은 파일 손상 검출 수단이지 배포자 서명이 아닙니다.

이미 다운로드한 일반 ZIP과 `SHA256SUMS`를 사용하려면 Windows에 `-ReleaseDirectory ABSOLUTE_RELEASE_FOLDER`, Unix에 `--release-directory ABSOLUTE_RELEASE_FOLDER`를 추가합니다.

### 설치기의 기본 프로그램 위치

- Windows: `%LOCALAPPDATA%\Programs\ToDoConnect\<version>`
- macOS: `~/Library/Application Support/ToDoConnect/<version>`
- Linux: `${XDG_DATA_HOME:-~/.local/share}/todo-connect/programs/<version>`

이는 설치기의 위치입니다. Codex ZIP 업로드 설치는 Codex가 관리하는 플러그인 위치를 사용합니다. 계정 데이터는 두 경우 모두 프로그램과 별도로 저장됩니다.

## 다른 MCP 호스트에 연결

일반 ZIP의 실행 파일은 `plugins/todo-connect/bin/todo-connect.exe`(Windows) 또는 `plugins/todo-connect/bin/todo-connect`(macOS/Linux)에 있습니다. 디렉터리 구조와 Unix 실행 권한을 유지합니다.

`mcpServers` 형식을 지원하는 호스트의 예시입니다. `command`는 설치된 실행 파일의 실제 절대 경로로 바꿉니다.

```json
{
  "mcpServers": {
    "todo-connect": {
      "command": "C:/YOUR_INSTALL_DIRECTORY/plugins/todo-connect/bin/todo-connect.exe",
      "args": ["serve"]
    }
  }
}
```

macOS/Linux의 경로 예시는 `/absolute/install/plugins/todo-connect/bin/todo-connect`입니다. 호스트별 설정 형식은 다를 수 있습니다. stdio MCP이므로 수신 포트나 공개 URL은 추가하지 않습니다.

MCP 설정만 추가하는 호스트는 동봉 Skill을 자동으로 읽지 않을 수 있습니다. 호스트의 지침 등록 방식을 따르거나 [시작 안내](../packaging/GETTING_STARTED.md)의 CLI 연결 절차를 사용합니다.

## 업데이트와 제거

새 버전은 별도 폴더에 설치하고 기존 계정 데이터는 유지합니다. 기존 MCP 프로세스를 종료한 뒤 호스트가 새 실행 파일을 사용하도록 바꿉니다. 버전·연결 목록·읽기 접근을 확인하기 전에는 이전 프로그램을 지우지 않습니다.

설치기가 등록한 `todo-connect-local` 마켓플레이스의 경로를 바꾸는 경우에만 다음 순서를 사용합니다. GUI 업로드 설치에 그대로 적용하지 않습니다.

```text
codex plugin remove todo-connect@todo-connect-local
codex plugin marketplace remove todo-connect-local
codex plugin marketplace add "NEW_PACKAGE_ROOT" --json
codex plugin add todo-connect@todo-connect-local --json
codex plugin list --marketplace todo-connect-local --json
```

`NEW_PACKAGE_ROOT`는 새 일반 패키지의 `BUILD.json`과 `.agents`가 있는 폴더입니다. 등록 교체와 계정 삭제는 별개입니다. 플러그인 제거만으로 별도 저장된 인증정보나 서비스 측 동의가 없어지지는 않습니다.

연결 해제는 해당 연결을 명시하여 요청합니다. 서비스 측 동의 철회는 해당 계정 설정에서 처리합니다. 원격 할 일 삭제와 혼동하지 마세요.
