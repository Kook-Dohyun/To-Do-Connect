# 실행 환경과 보관 방식 — 개발본

이 문서는 현재 실행 가능한 개발본의 절차다. GitHub 릴리스·공개 이미지 주소·완성 플러그인 설치 안내는 아직 확정하지 않았다. 아래 Docker 이미지는 로컬 빌드한 `todo-connect:local`이다. 일반 사용자에게 소스 빌드를 요구하는 최종 배포 방식이 아니다.

## OS별 실행 파일

`scripts/build-targets.ps1`으로 만든 파일은 `dist/dev/<os>-<arch>/`에 있다. `windows`, `darwin`(macOS), `linux` 각각 `amd64`·`arm64`가 있으며 Windows 이름은 `todo-connect.exe`, 나머지는 `todo-connect`다. 실행 자체에 Go·Node·Docker 설치가 필요하지 않다. macOS/Linux에 수동 복사했다면 실행 권한을 부여한다. macOS 코드 서명·공증은 아직 하지 않았다.

| 환경 | 기본 비밀정보 보관 |
| --- | --- |
| Windows | 같은 Windows 사용자의 DPAPI |
| macOS | Keychain의 키 + AES-256-GCM 파일 |
| Linux 데스크톱 | Secret Service/D-Bus의 키 + AES-256-GCM 파일 |
| Linux Docker | 별도 키 파일 + AES-256-GCM 파일 |

Linux 데스크톱 기본 저장은 사용 가능한 Secret Service 세션이 필요하다. 키 저장소가 잠겼거나 없을 때 평문 저장으로 전환하지 않는다. 키링을 사용하지 않는 환경은 아래 키 파일 방식을 명시적으로 선택한다. 자동 검증의 범위와 한계는 [CI 안내](https://github.com/Kook-Dohyun/To-Do-Connect/blob/main/docs/ci.md)를 따른다.

## 보관 위치와 키

키 저장소의 자동 테스트와 사용자 데스크톱의 잠금 해제 화면·복구 절차·OAuth 검증은 별개다.

Windows는 Local AppData의 `todo-connect`, 다른 OS는 `os.UserConfigDir()`의 `todo-connect`를 기본 데이터 경로로 쓴다. `TODO_CONNECT_DATA_DIR`에 절대 경로를 지정할 수도 있다. CLI와 MCP에서 **같은 데이터 경로·키 설정**을 사용해야 한다. 소스·플러그인·Vault·동기화 폴더에 인증정보를 넣지 않는다.

Codex 0.155.1의 플러그인 MCP는 부모 셸의 임의 환경 변수를 모두 전달하지 않는다. Linux 호스트 검사에서는 별도로 지정한 `XDG_CONFIG_HOME`이 전달되지 않아 CLI와 MCP가 서로 다른 빈 목록을 읽는 현상을 확인했다. 기본 경로는 `HOME/.config/todo-connect`로 일치시켜 검증했다. 사용자 지정 XDG·데이터·키 경로를 쓰면 MCP 호스트 환경에도 동일한 설정을 명시한다. 빈 목록만 보고 기존 연결을 다시 만들거나 재로그인하지 말고, 먼저 양쪽 저장 경로를 확인한다.

키 파일 방식을 선택한다면 비공개 디렉터리의 절대 경로로 한 번 생성한다.

```text
todo-connect keygen --out ABSOLUTE_PRIVATE_KEY_PATH
```

`TODO_CONNECT_KEY_FILE`에는 그 **파일 경로만** 지정한다. 키 내용은 채팅·환경 변수 값·명령 인자로 전달하지 않는다. Unix에서는 파일 권한 0600을 유지하고 Windows에서는 해당 사용자만 접근하는 디렉터리에 보관한다. 키 파일은 자동 덮어쓰지 않는다.

키가 없으면 기존 암호화 파일을 복호화할 수 없다. 백업에는 암호화 데이터와 복원 가능한 원래 키 보관 수단이 모두 필요하다. 키와 암호화 파일을 함께 가진 사람은 복호화할 수 있으므로 둘 다 접근을 제한한다. OS 키링의 키는 데이터 디렉터리에 연결되므로 경로 변경은 단순 파일 이동으로 처리하지 않는다. Windows DPAPI 파일을 Linux 컨테이너에 복사해 재사용할 수는 없다. 기존 연결의 저장 방식을 자동 변환하는 기능은 아직 없다.

## Docker 실행

### 사전 빌드 이미지 파일

CI는 검증한 Linux/amd64·arm64 이미지를 별도의 `docker-candidate-linux-<arch>` 아티팩트로 보관한다. 공개 Docker 레지스트리나 GitHub 릴리스에는 아직 게시하지 않았다. 접근 가능한 검증 실행에서 후보 아티팩트를 내려받아 사용할 수 있으며, 보관 기간은 하루다. 일반 사용자용 설치 완료를 의미하지 않는다.

Docker 엔진의 Linux CPU 아키텍처에 맞는 파일을 선택한다. Windows/macOS의 Docker Desktop도 여기서는 Linux 컨테이너 엔진을 대상으로 한다. 동봉된 Docker용 `SHA256SUMS`와 파일의 SHA-256을 비교한 뒤 불러온다. 이 목록은 네이티브 ZIP용 체크섬 목록과 별개다.

```text
docker image load --input todo-connect-VERSION-linux-ARCH.tar.gz
docker image inspect todo-connect:VERSION --format "{{.Os}}/{{.Architecture}}"
docker run --rm todo-connect:VERSION version
```

`VERSION`과 `ARCH`는 받은 파일에 맞춰 바꾼다. 이 방식은 Go나 소스 빌드가 필요 없지만 Docker는 설치돼 있어야 한다. 아래 예시의 `todo-connect:local` 대신 불러온 `todo-connect:VERSION` 태그를 사용한다. 이미지를 불러오는 것만으로 연결 설정이나 로그인·계정 데이터 볼륨은 만들어지지 않는다. 네이티브 프로그램 사용자는 Docker를 설치할 필요가 없다.

### 개발자용 이미지 빌드

현재 개발자용 이미지 빌드:

```text
docker build -f packaging/docker/Dockerfile -t todo-connect:local .
```

릴리스 버전을 지정하려면 `--build-arg VERSION=VERSION`을 추가하고 `VERSION`을 원하는 빌드 버전으로 바꾼다. `docker run --rm todo-connect:local version`으로 확인한다. 프로젝트 MIT 라이선스와 의존성 원문 고지는 `/usr/share/licenses/todo-connect/`에 동봉한다.

이미지는 UID/GID `10001:10001`로 실행한다. 아래 명령을 자신의 로컬 터미널에서 수행한다. `/data`는 연결 설정·암호화 캐시, `/keys`는 암호화 키다. 두 볼륨은 컨테이너가 없어져도 남는다.

```text
docker volume create todo-connect-data
docker volume create todo-connect-keys
docker run --rm -v todo-connect-data:/data -v todo-connect-keys:/keys todo-connect:local keygen --out /keys/todo-connect.key
```

이미 준비한 키가 있으면 `keygen`은 반복하지 않는다. 시작할 때마다 키를 생성하거나 볼륨을 삭제하는 초기화 절차를 만들지 않는다.

### Microsoft

```text
docker run --rm -v todo-connect-data:/data -v todo-connect-keys:/keys:ro todo-connect:local connections add --provider microsoft --id personal --label "Personal Microsoft"
docker run --rm -it -v todo-connect-data:/data -v todo-connect-keys:/keys:ro todo-connect:local login personal --device
docker run --rm -v todo-connect-data:/data -v todo-connect-keys:/keys:ro todo-connect:local check personal
```

사용자가 터미널의 안내에 따라 Microsoft 브라우저 인증을 진행한다. 실제 계정 로그인 출력은 AI의 캡처 로그나 채팅에 붙이지 않는다. Microsoft device flow에는 포트 게시가 필요 없다.

### Google

사용자 소유 Google 프로젝트의 **Desktop app OAuth JSON**과 활성화된 Google Tasks API가 필요하다. 동의 화면을 준비하고 본인의 계정으로 로그인한다. 원본 JSON 파일은 비공개 로컬 경로에 보관한다.

PowerShell에서 JSON 내용을 화면에 출력하지 않고 컨테이너 표준 입력으로 전달한다:

```powershell
Get-Content -Raw -LiteralPath 'ABSOLUTE_DESKTOP_CLIENT_JSON_PATH' | docker run --rm -i -v todo-connect-data:/data -v todo-connect-keys:/keys:ro todo-connect:local connections add --provider google --id google-personal --label 'Personal Google' --credentials -
```

macOS/Linux 셸에서는 입력 리디렉션을 쓴다:

```sh
docker run --rm -i -v todo-connect-data:/data -v todo-connect-keys:/keys:ro todo-connect:local connections add --provider google --id google-personal --label 'Personal Google' --credentials - < '/absolute/private/desktop-client.json'
```

로그인할 때만 호스트 loopback 포트를 게시한다. 브라우저는 Docker가 실행되는 같은 PC에서 연다:

```text
docker run --rm -it -p 127.0.0.1:8765:8765 -v todo-connect-data:/data -v todo-connect-keys:/keys:ro todo-connect:local login google-personal --headless --container-port 8765
docker run --rm -v todo-connect-data:/data -v todo-connect-keys:/keys:ro todo-connect:local check google-personal
```

8765가 사용 중이면 게시하는 호스트 포트·컨테이너 포트·`--container-port`를 모두 같은 미사용 포트로 바꾼다. `--container-port`는 컨테이너 내부에서 수신하기 위한 명시적 옵션이다. 호스트의 `127.0.0.1`을 빼고 외부에 공개하지 않는다. 로그인 URL은 터미널에서 직접 열며 채팅이나 로그로 공유하지 않는다.

### MCP 실행

```text
docker run --rm -i -v todo-connect-data:/data -v todo-connect-keys:/keys:ro todo-connect:local serve
```

MCP 호스트에는 `command: docker`와 위 인자들을 배열로 설정한다. MCP에는 `-t`를 붙이지 않고 포트를 게시하지 않는다. CLI로 로그인한 것과 동일한 두 볼륨을 사용한다. 실제 플러그인 호스트 설치 검증은 별도 단계다.

연결 추가·해제는 다른 provider나 연결을 자동 해제하지 않는다. `connections remove --id ID --confirm`은 해당 로컬 설정·인증정보만 제거하고 서비스의 할 일은 삭제하지 않는다.

## 개발자 검증

Windows에서 일반 테스트·검사:

```text
go -C server test ./... -count=1
go -C server vet ./...
```

Linux의 race detector·검사:

```text
docker build --target test -f packaging/docker/Dockerfile -t todo-connect-test:local .
```

로컬 runtime 이미지의 컨테이너 영속성·MCP 테스트(PowerShell):

```powershell
$savedTestImage = $env:TODO_CONNECT_TEST_IMAGE
try {
    $env:TODO_CONNECT_TEST_IMAGE = 'todo-connect:local'
    go -C server test ./cmd/todo-connect -run TestDockerRuntime -count=1 -v
} finally {
    $env:TODO_CONNECT_TEST_IMAGE = $savedTestImage
}
```

가짜 Microsoft 연결과 Google 클라이언트로 테스트하고 사용자 계정에 로그인하지 않는다. 테스트가 만든 실행별 라벨의 컨테이너·볼륨만 확인 후 제거한다. 테스트 이미지가 더 필요 없으면 정확한 `todo-connect-test:local` 태그를 확인해 제거하며 다른 이미지·볼륨을 일괄 정리하지 않는다. 회귀 테스트 소스는 보존한다.
