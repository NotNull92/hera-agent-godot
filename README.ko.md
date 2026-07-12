<p align="center">
  <img src="docs/assets/hera_godot_logo.png" alt="hera-agent-godot logo" width="420">
</p>

# hera-agent-godot

[English](README.md) · **한국어**

> Hera gives agents eyes, hands, and proof in the live Godot editor.

AI 코딩 에이전트가 **실행 중인 Godot 4.7+ 에디터**를 실시간으로 검사·제어하게 해주는
**저토큰(low-token) CLI**입니다 — 출력/에러 읽기, 씬 실행, 노드 트리 탐색·편집,
GDScript 평가 등. 에이전트가 낡은 학습 데이터로 추측하는 대신 *실제* 에디터에
직접 작용하고 결과를 확인합니다.

**왜 MCP가 아니라 CLI인가?** Godot에는 이미 MCP 애드온 생태계가 활발합니다 —
헤라는 일부러 그 반대에 베팅합니다. MCP 서버는 폭을 토큰으로 지불합니다: 수십~100개+의
도구 스키마와 장황한 JSON 응답이 **매 턴** 에이전트 컨텍스트에 얹힙니다. 헤라는
**MCP급의 에디터 제어 범위를 compact-JSON 기본의 CLI로** 제공합니다 — 액션당 한 명령,
최소 토큰, 그리고 MCP 클라이언트만이 아니라 **셸 명령을 실행할 수 있는 무엇과도**
동작합니다(파이프, `batch`, CI, 어떤 에이전트든).

헤라의 제품 아이덴티티는 간단합니다: **라이브 에디터 진실, 저토큰 제어,
증거 우선 QA**. 새 기능의 언어와 설계 원칙은
[docs/IDENTITY.md](docs/IDENTITY.md)에 정리되어 있습니다.

[`hera-agent-unity`](https://github.com/NotNull92/hera-agent-unity)의 자매
프로젝트로, 동일한 저토큰·쉘 친화 철학을 따르며 **포팅이 아니라 Godot에 맞춰
새로 설계**했습니다.

## 현재 릴리스 기준: v0.7.0

`v0.7.0`은 현재 저장소 태그와 애드온 매니페스트 기준 버전입니다. Godot
Asset Store 업로드 준비가 완료되었고, 이전 스토어 업로드는 **2026-07-06**의
`v0.6.0`입니다.

주요 변경 사항:

- **Game Feel Mode**가 Hera 에디터 패널, 영구 설정, 번들 `game_feel` 토픽,
  그리고 게임플레이 피드백 작업용 `hera guidance game-feel`을 제공합니다.
- `game input`으로 마우스, 키보드, InputMap 액션, 텍스트 입력을 런타임에 주입하고
  `game input-log`로 실제 전달된 이벤트를 증명할 수 있습니다.
- `game qa --file` 시나리오가 `game.input`, `game.input_log` 단계를 지원해,
  요구사항 커버 QA가 실제 플레이어 입력 경로를 검증할 수 있습니다.
- Game Feel Mode가 켜져 있으면 관련 `node add` 응답이 짧은 `agent_hint`
  토픽을 반환할 수 있습니다.
- v0.7 프롬프트 게임 사이클에서 얻은 교훈을 guidance와 문서에 반영했습니다:
  순서가 중요한 상태 변경 QA, 주요 입력 검증, 뷰포트/레이아웃 규칙,
  결정적 헬퍼, 플레이어에게 보이는 피드백 증거.

릴리스 노트와 Asset Store 패키징 세부 사항:
[docs/releases/v0.7.0-asset-store-upload.md](docs/releases/v0.7.0-asset-store-upload.md).

## 저토큰, 실측

"MCP급 범위, 더 적은 토큰" 주장을 수치로:

| | 헤라 (CLI) | Godot MCP 서버 (도구 약 41~155개) |
|---|---|---|
| **매 턴** 상주하는 도구 스키마 | **0** | ~4k~31k tok (도구 수에 비례 증가) |
| 에이전트가 로드하는 표면 | 문서 1개, ~1.0k tok — 캐시 가능·평탄 | 전체 도구 목록, 매 턴 재전송 |
| 액션당 응답 | compact JSON — `status` ≈48 tok, `node get` ≈186 tok | JSON, 보통 pretty |

헤라 수치는 라이브 Godot 4.7 에디터에서 **실측**, MCP 열은 공개 Godot MCP
서버들의 표본 도구 수(약 41~155개) × 도구 스키마당 ~100~200 tok 으로 낸
**추정**입니다. 방법론·한계·재현법:
**[docs/LOW_TOKEN.md](docs/LOW_TOKEN.md)**.

## 명령 표면

`v0.7.0` CLI/애드온 표면에는 다음 명령이 포함됩니다:
`status`, `instances`, `run`/`stop`, `scene`, `editor`, `script`, `project`, `classdb`,
`node`(읽기+쓰기+리소스/스크립트 연결), `signal`,
`resource`(get/uid/list/set/create/resave/update-uids/export-mesh-library),
`game`(런타임 검사+input+input-log+set/call/click+assert+QA+screenshot),
`guidance`, `game_feel`, `output`, `diagnostics`, `eval`, `screenshot`, `batch`,
`smoke` + `--json`/`--ids` 출력 모드. 명령 레퍼런스는
[docs/COMMANDS.md](docs/COMMANDS.md), 릴리스와 Asset Store 패키징 상태는
[docs/ROADMAP.md](docs/ROADMAP.md)에서 확인하세요.

## 설치

**CLI** — 최신 릴리스 바이너리를 받아 설치하는 원라인:

```sh
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/NotNull92/hera-agent-godot/main/install.sh | sh
```

```powershell
# Windows (PowerShell)
irm https://raw.githubusercontent.com/NotNull92/hera-agent-godot/main/install.ps1 | iex
```

특정 태그는 `HERA_VERSION`, 설치 경로는 `HERA_BIN_DIR`로 지정할 수 있습니다.
소스 빌드는 `go build -o hera .` (Go 1.25+). `hera version`으로 확인하세요.

**애드온** — [최신 릴리스](https://github.com/NotNull92/hera-agent-godot/releases/latest)에서
`hera-agent-godot-addon.zip`을 받아 Godot 프로젝트 루트에 풀면(`addons/hera_agent_godot/` 생성)
**프로젝트 → 프로젝트 설정 → 플러그인**에서 활성화할 수 있습니다.

## 동작 방식

```
Go CLI  ──HTTP /rpc──▶  Godot 에디터 애드온 (@tool EditorPlugin, GDScript)
 (cmd/, internal/)        (addons/hera_agent_godot/)
        ▲                          │
        └── ~/.hera-agent-godot/instances/ 스캔 ◀── Heartbeat
```

- **CLI** (Go): 에디터를 탐색하고, 명령마다 압축된 JSON 요청 하나를 보냅니다.
- **애드온** (GDScript): localhost HTTP 서버를 띄우고, 각 요청을 `EditorInterface`를 통해
  에디터 메인 스레드에서 실행합니다.

전체 설계는 **[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)**, 명령 목록은
**[docs/COMMANDS.md](docs/COMMANDS.md)**, 릴리스 이력은
**[docs/ROADMAP.md](docs/ROADMAP.md)**를 참고하세요.

## 디렉토리 구조

```
addons/hera_agent_godot/  배포용 Godot 4.7+ 애드온 (GDScript)
project.godot, scenes/    개발용 호스트 프로젝트 — CLI의 run/save/screenshot 대상
cmd/                      Go CLI 명령 (status, instances, run/stop, scene, editor, script, project, classdb, node, signal, resource, game, guidance, game_feel, output, diagnostics, eval, screenshot, batch, smoke)
internal/                 client / discovery / protocol
docs/                     ARCHITECTURE, COMMANDS, ROADMAP, 릴리스 노트, 프롬프트 게임 가이드
```

## 요구 사항

- Go 1.25+ (CLI)
- Godot **4.7+** 표준 빌드 권장 (애드온). 검증된 최소 버전은 **4.2**입니다:
  4.2–4.6에서 애드온 로드와 CLI 응답을 스팟체크로 확인했습니다 —
  [docs/SUPPORT_MATRIX.md](docs/SUPPORT_MATRIX.md) 참고.

## 보안

브리지는 `127.0.0.1`에만 바인딩되고 브라우저 오리진 요청을 거부합니다.
옵트인 공유 토큰 인증(`~/.hera-agent-godot/token` 또는
`HERA_AGENT_GODOT_TOKEN`)으로 비밀을 아는 클라이언트만 허용할 수 있습니다.
위협 모델과 설정: [docs/SECURITY.md](docs/SECURITY.md).

## 자매 프로젝트: hera-agent-unity

Unity도 쓰신다면 — [**hera-agent-unity**](https://github.com/NotNull92/hera-agent-unity)는
동일한 저토큰·셸 친화 철학을 **Unity 에디터**에 적용합니다: 콘솔 에러 읽기, C# 실행,
Play Mode 진입, GameObject 관리, UI 빌드, 테스트 실행 — 전부 compact·에이전트 친화
출력으로. 두 엔진을 오가도 에이전트는 일관된 방식으로 각각을 제어합니다.

## 후원

Hera는 무료이며 MIT 라이선스입니다. 도움이 되었다면 개발을 후원하실 수 있습니다:

[디스코드 커뮤니티 참여하기](https://discord.gg/QBzEVuYwK)

[![Ko-fi에서 후원](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/notnull92)

## 라이선스

MIT — [LICENSE](LICENSE) 참고.
