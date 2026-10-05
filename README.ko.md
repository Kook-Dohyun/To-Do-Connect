# To Do Connect

<img src="packaging/plugin/assets/icon.png" width="96" height="96" alt="To Do Connect">

**쓰던 할 일 서비스를 AI에 연결하세요.**

[English](README.md) · 한국어

Microsoft To Do와 Google Tasks를 AI에서 사용합니다. 한 서비스의 계정 하나부터 두 서비스의 여러 계정까지 함께 연결하고, 기존 할 일 앱을 계속 사용할 수 있습니다.

프로그램은 사용자 컴퓨터에서 실행되며 각 서비스 API에 직접 접근합니다. 개발자가 운영하는 중계 서버는 없습니다. 네이티브 패키지에 실행 파일이 포함되어 Go·Node.js·Docker 설치가 필요하지 않습니다.

> 안정판이 아닌 개발 프리뷰입니다. 소스와 로컬 개발 패키지는 [GitHub Releases](https://github.com/Kook-Dohyun/To-Do-Connect/releases)의 공개 파일보다 새 버전일 수 있습니다. GitHub 배포와 OpenAI 공개 플러그인 디렉터리 등록은 별개입니다.

## 할 수 있는 일

| 서비스 | 기능 |
| --- | --- |
| Microsoft To Do | 목록·할 일·체크리스트 단계의 생성, 조회, 수정, 완료, 삭제 |
| Google Tasks | 목록·할 일의 생성, 조회, 수정, 완료, 삭제와 하위 할 일 생성·이동·순서 변경 |
| 연결 관리 | 저장된 연결 재사용, 여러 계정 추가, 계정별 연결 해제 |

현재 서버는 도구 27개를 제공합니다. 서비스 사이의 할 일 동기화·복사는 제공하지 않습니다.

## 설치

다음 중 하나를 선택하세요. 모두 설치할 필요는 없습니다.

- **Codex 플러그인 ZIP:** OS·CPU에 맞는 플러그인 ZIP을 받아 지원되는 데스크톱 앱의 **플러그인 → 추가 → 플러그인 압축 파일 업로드**에서 설치합니다.
- **AI에게 GitHub 설치 요청:** 아래 내용을 로컬 AI 어시스턴트에 전달합니다.
- **다른 MCP 클라이언트:** 네이티브 패키지를 설치하고 실행 파일에 `serve` 인자를 지정합니다. [설치 방법과 MCP JSON](docs/installation.md)을 참고하세요.

```text
https://github.com/Kook-Dohyun/To-Do-Connect 에서 To Do Connect를 설치해줘.
실행할 컴퓨터의 OS·CPU와 기존 설치부터 확인해.
공개된 사전 빌드 패키지와 체크섬을 사용하고, 소스 빌드나
Go·Node.js·Docker 설치는 하지 마. 프리뷰만 있으면 버전을 알려줘.
내가 사용할 로컬 MCP/플러그인 클라이언트에 등록해.
저장된 연결은 보존하고 도구가 로드되는지 검증한 다음,
어떤 할 일 서비스와 계정을 연결할지 물어봐. 미완료 설정도 함께 진행해줘.
```

맞는 공개 패키지가 없다면 제공받은 개발 압축파일을 사용하거나 릴리스를 기다립니다. 소스 버전만 보고 다운로드 파일이 있다고 가정하지 않습니다.

## 계정 연결

- **Microsoft To Do:** Microsoft 개인 계정으로 브라우저 로그인을 완료합니다. 앱 ID는 내장되어 있어 환경변수나 별도 앱 등록이 필요하지 않습니다.
- **Google Tasks:** 본인 소유 Google Cloud 프로젝트에서 Tasks API를 활성화하고 Desktop OAuth 클라이언트를 만듭니다. 동봉 스킬이 준비 과정을 안내하고 다운로드한 JSON을 로컬로 가져온 뒤 브라우저 로그인을 엽니다. 기존 프로젝트·클라이언트·정상 연결은 재사용합니다.

비밀번호·MFA·권한 동의는 서비스 화면에서 직접 처리합니다. 저장된 연결이 정상 동작하면 매번 로그인하지 않습니다.

## 사용 예시

```text
내 할 일 목록을 보여줘.
“서버실 점검” 목록을 만들어줘.
“1. 서버 식별정보 확인” 할 일에 “호스트명 기록”, “OS 버전 기록” 단계를 추가해줘.
내가 완료한 할 일과 단계를 확인해줘.
오늘 오후 3시 이후 완료한 할 일과 하위 항목을 내 시간대 기준으로 보여줘.
```

계정이나 목록이 여러 개라 대상이 불명확하면 어시스턴트가 사용할 대상을 확인합니다.

완료 내역은 대화 기억이 아니라 서비스에 기록된 완료 시각으로 조회합니다. 날짜·시간대를 확인하고 상위 할 일과 체크리스트·하위 할 일을 구분합니다. 현재 완료된 항목을 보여주는 기능이며, 삭제·완료 취소까지 포함한 전체 이력이나 백그라운드 감시·완료 hook은 아닙니다.

## 다른 앱과도 함께 사용하기

다른 앱이 같은 Microsoft To Do 계정·목록으로 동기화하는 할 일도 To Do Connect에서 활용할 수 있습니다. 아래는 각 서비스가 제공하는 기존 연동이며, 이 플러그인이 별도 provider로 직접 구현한 기능은 아닙니다.

| 앱·서비스 | 연결 방식 |
| --- | --- |
| 삼성 갤럭시 리마인더 | 리마인더 설정에서 **Microsoft To Do와 동기화**를 켜고 플러그인에도 같은 Microsoft 계정을 연결합니다. 모바일에서는 리마인더를 사용하고 AI에서는 동기화된 To Do 목록을 다룰 수 있습니다. [공식 설정 안내](https://support.microsoft.com/en-us/todo/sync-microsoft-to-do-with-the-samsung-reminder-app) |
| Outlook | 같은 Microsoft 계정의 Outlook 작업과 To Do가 동기화됩니다. 이 플러그인은 To Do 할 일을 다루며 메일함·캘린더에 직접 접근하지 않습니다. [공식 안내](https://support.microsoft.com/en-us/outlook/how-can-i-manage-my-outlook-tasks-on-mobile) |
| Zapier | Microsoft To Do 커넥터로 할 일을 만들거나 생성·완료를 조건으로 워크플로를 실행할 수 있습니다. 별도로 연결하면 그 결과로 생성된 To Do 할 일을 이 플러그인에서도 활용할 수 있습니다. [커넥터 안내](https://help.zapier.com/hc/en-us/articles/8496034283533-How-to-get-started-with-Microsoft-To-Do-on-Zapier) |
| IFTTT | Microsoft To Do 연동에 할 일 생성과 완료 트리거가 있으며 다른 앱과 연결하는 예시를 제공합니다. Applet은 별도 자동화이며 기본 양방향 동기화를 뜻하지 않습니다. [연동 안내](https://ifttt.com/microsoft_todo) |

예를 들어 **갤럭시 리마인더 ↔ Microsoft To Do ↔ To Do Connect ↔ AI**로 활용합니다. AI에 추가·조회할 때 리마인더와 동기화한 목록을 지정하세요. 현재 Microsoft 안내상 삼성 리마인더는 To Do 목록 하나를 동기화하며, Samsung Cloud에만 저장한 리마인더와 일부 기능(To Do 하위 단계 등)은 같은 방식으로 공유되지 않습니다. 기기·앱 조건과 기능 차이는 공식 안내를 확인하세요. 외부 동기화에는 시간이 걸릴 수 있습니다.

현재 내장 Microsoft 로그인은 개인 계정을 대상으로 합니다. 외부 자동화 서비스는 별도 계정·권한 동의가 필요하고 유료 요금제나 사용 제한이 있을 수 있으며, 해당 서비스로 데이터가 전달됩니다. 선택 사항이고 필수 설치가 아닙니다. 위 연동은 제공업체 문서로 확인했으며 이 프로젝트에서 각각 실기기 종단 검증을 한 것은 아닙니다. 플러그인 설치만으로 외부 연동이 활성화되거나 API 권한이 늘어나지는 않습니다.

## 플랫폼과 개인정보

Windows·macOS·Linux의 x64와 ARM64 패키지를 대상으로 합니다. macOS/Linux는 사용 가능한 OS 키 저장소 또는 명시적으로 설정한 암호화 키가 필요합니다. 현재 실행 파일은 코드 서명·공증을 제공하지 않습니다. Docker는 선택 사항입니다.

인증정보는 프로그램·저장소와 분리된 사용자 데이터 폴더에 암호화해 저장합니다. 도구 결과는 사용하는 AI 앱으로 전달되므로, 로컬 실행이 할 일 내용의 AI 서비스 전달까지 차단한다는 뜻은 아닙니다.

[설치](docs/installation.md) · [실행 환경·Docker](docs/runtime.md) · [개발](docs/development.md) · [패키징](docs/releases.md) · [검증](docs/ci.md)

[지원](docs/support.md) · [개인정보](docs/privacy.md) · [이용 조건](docs/terms.md)

[MIT License](LICENSE)로 배포합니다. Microsoft To Do와 Google Tasks는 각 소유자의 제품이며, 이 프로젝트는 Microsoft 또는 Google의 공식 제품이 아닌 독립적인 커뮤니티 프로젝트입니다.
