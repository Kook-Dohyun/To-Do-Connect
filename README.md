# To Do Connect

**Connect your task services to AI. Keep the accounts you already use.**

Microsoft To Do와 Google Tasks를 AI에 연결하는 로컬 MCP 서버·플러그인입니다. 한 서비스만 고를 필요 없이 여러 서비스와 계정을 함께 연결할 수 있습니다.

프로그램은 사용자의 컴퓨터에서 실행되고 각 서비스 API에 직접 접근합니다. 개발자가 운영하는 중계 서버는 없습니다. 네이티브 배포 파일에는 실행 프로그램이 포함되어 **Go·Node.js·Docker를 설치할 필요가 없습니다.**

> 공개 배포 준비 중입니다. 다운로드 파일은 [GitHub Releases](https://github.com/Kook-Dohyun/To-Do-Connect/releases)에 게시됩니다. 아직 게시되지 않은 버전의 다운로드 주소를 사용하지 마세요.

## 할 수 있는 일

- 목록과 할 일을 조회·추가·수정·완료·삭제합니다.
- Microsoft To Do의 체크리스트 단계·첨부파일·연결 참조를 다룹니다.
- Google Tasks의 하위 할 일과 순서를 관리합니다.
- Microsoft와 Google 계정을 함께 연결하고, 연결별로 추가·해제합니다.
- 동봉된 사용 지침으로 AI가 기존 연결을 확인하고 필요한 설정을 안내합니다.

서비스 간 자동 동기화나 Microsoft ↔ Google 간 복사는 제공하지 않습니다. 각 서비스 API의 기능과 제약이 적용됩니다.

## 시작하기

### Codex에 플러그인 ZIP 설치

1. 릴리스에서 실행할 컴퓨터에 맞는 **`todo-connect-plugin-<version>-<os>-<arch>.zip`**을 받습니다.
2. Codex의 **플러그인 → 추가 → 플러그인 압축 파일 업로드**에서 ZIP을 선택하고 설치합니다.
3. 대화에서 To Do Connect를 선택하고 요청합니다.

```text
내 할 일 서비스를 연결해줘. 기존 연결이 있으면 먼저 확인하고 재사용해.
Microsoft To Do와 Google Tasks를 함께 사용할 수 있게 도와줘.
로그인과 권한 동의는 내가 직접 할게.
```

### AI에게 GitHub 설치 요청

로컬 파일과 명령을 다룰 수 있는 AI 에이전트에 다음 내용을 전달할 수 있습니다.

```text
https://github.com/Kook-Dohyun/To-Do-Connect 의 To Do Connect를
이 컴퓨터의 Codex에 설치해줘.

저장소의 docs/installation.md를 읽고, 실제 게시된 릴리스 중
이 컴퓨터의 OS와 CPU에 맞는 사전 빌드 패키지를 사용해.
공식 저장소의 다운로드 주소와 SHA-256을 확인하고 설치 전에
설치 버전과 변경할 Codex 등록 내용을 알려줘.
Go·Node.js·Docker 설치나 소스 빌드는 하지 마.
기존 플러그인·계정 정보를 삭제하지 말고, 설치 후 기존 연결부터 확인해.
연결할 서비스는 나에게 물어보고 로그인·권한 동의는 내가 직접 하게 해줘.
```

[설치 안내와 MCP JSON](docs/installation.md) · [저장 방식과 선택적 Docker 실행](docs/runtime.md)

## 계정 연결

| 서비스 | 처음 연결할 때 |
| --- | --- |
| Microsoft To Do | 개인 Microsoft 계정으로 브라우저 로그인·권한 동의. 기본 앱의 Client ID를 별도로 입력할 필요가 없습니다. |
| Google Tasks | 사용자 소유 Google Cloud 프로젝트의 Tasks API와 Desktop OAuth 클라이언트 설정이 필요합니다. 동봉 지침으로 AI가 준비 상태를 확인하고 설정을 도와줍니다. 로그인·동의는 사용자가 직접 합니다. |

Google 연결은 단순 로그인만으로 준비가 끝나는 방식이 아닙니다. 이미 준비된 프로젝트·클라이언트·연결은 재사용합니다. 두 서비스 모두 저장된 연결이 정상 동작하면 매번 로그인하지 않습니다.

## 플랫폼과 개인정보

Windows·macOS·Linux의 x64(`amd64`)와 ARM64(`arm64`)를 대상으로 패키지를 만듭니다. macOS 파일명은 `darwin`입니다. 현재 바이너리는 코드 서명·macOS 공증을 제공하지 않습니다.

인증정보는 배포 ZIP이나 소스 폴더가 아닌 사용자의 별도 데이터 위치에 암호화하여 저장합니다. Windows는 DPAPI, macOS/Linux는 OS 키 저장소를 사용합니다. Docker는 선택적인 실행 방식이며 별도 키 파일을 사용합니다.

AI에 요청한 할 일 내용과 도구 결과는 사용 중인 AI 호스트가 처리합니다. 로컬 MCP라는 이유만으로 할 일 내용이 AI 서비스에 전달되지 않는 것은 아닙니다. 플러그인 제거와 계정 연결 해제도 별개입니다. [자세한 안내](docs/runtime.md)

## 개발과 지원

[개발 안내](docs/development.md) · [패키지 조립](docs/releases.md) · [자동 검증](docs/ci.md) · [문제 제보](https://github.com/Kook-Dohyun/To-Do-Connect/issues)

프로젝트 코드는 [MIT License](LICENSE)로 배포합니다. 의존성 고지는 배포 파일에 포함됩니다. Microsoft To Do와 Google Tasks는 각 소유자의 제품·상표이며, To Do Connect는 Microsoft 또는 Google의 공식 제품이 아닙니다.
