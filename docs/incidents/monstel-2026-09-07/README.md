# Monstel 연동 중 Hera 문제·충돌 인계

작성: 2026-09-07. 대상: **hera-agent-godot 로컬 레포에서의 후속 수정**.

실제로 남은 로그·세션과 9월 7일 소스를 대조한 기록이다. 이 문서 작성 중에는 오류를 새로 유발하거나 패치를 적용하지 않았다. 아래의 **역사적 PASS는 당시 임시 패치에 대한 결과이며 현재 코드가 수정됐다는 뜻이 아니다.**

## 먼저 볼 항목

| ID | 문제 | 근거 수준 | 수정 우선순위 제안 |
|---|---|---|---|
| H01 | 외부 실행 게임이 목록에는 있지만 제어 불가, CLI `game --pid`도 미지원 | 실패 로그 + 현재 소스 + 당시 패치 성공 기록 | 높음 |
| H02 | 같은 씬의 여러 프로세스가 있으면 요청 대상 모호 | 반복 실패 로그 + 현재 소스 | 높음; H01과 함께 |
| H03 | 요청한 1080×1920 대신 embedded 캡처 927×1649 | 실제 PNG 메타데이터; 엔진/창 제약의 정확한 원인은 미확정 | 중간 |
| H04 | 에디터는 살아 있는데 RPC 타임아웃 후 `no live Godot editor` | 세션 원문 확인; headless/캐시 충돌은 추정 | 조사 필요 |
| H05 | Android에서 Hera autoload UID/리소스 로드 오류 | 여러 실기 로그 + 설정/UID/소스 일치 | 높음 |
| H06 | 병렬 런타임이 같은 세이브·편집 상태를 공유 | 세이브 해시 변경·당시 관찰 기록; 작성 프로세스 추적은 없음 | 높은 운영 위험 |
| H07 | headless export 종료 시 EditorSettings/리소스 누수 로그 | 로그 확인; Hera 귀속은 미확정 | 별도 분리 진단 |

우선 H01/H02의 **에디터 PID와 게임 PID를 구분하는 선택 계약**, H05의 **오토로드 수명·익스포트 처리**부터 검토하는 것을 권한다. 모호한 대상에 대한 거부 자체는 안전장치이므로 없애지 않는다.

## 환경과 재현 자료

- 사용 프로젝트: `/Users/admin/Desktop/Cowork/monstel`, Godot **4.7.2 .NET**, C# 모바일 세로 게임.
- 현재 조사 머신: macOS **26.6.2**. 과거 모든 실행이 같은 OS 패치 버전이었다는 의미는 아니다.
- 현재 설치 CLI: `/Users/admin/.local/bin/hera`, `--version` 출력 **`dev`**. addon `plugin.cfg`는 **1.0.0**이므로 이 문자열만으로 배포 빌드를 식별할 수 없다.
- Android 검증: Galaxy Z Flip7 / Android 16, 기본 캡처 1080×2520, 축소 화면 1080×1920.
- 엔진 경로: `/Users/admin/Downloads/Godot_mono.app/Contents/MacOS/Godot`.
- 원래 프로젝트에는 당시 실험 자료가 `.omo/evidence/start-work/monstel-game-feel-implementation/` 아래 남아 있다. 공유 가능한 핵심 자료는 이 폴더의 `evidence/`에 복사했다.
- 조사 시점 소스·CLI와 증거 파일 해시는 [fingerprints.txt](fingerprints.txt)에 기록한다. 로컬 레포는 Monstel에 복사된 addon과 일부 차이가 있으므로 옛 패치를 그대로 적용하지 말 것.

## H01. 외부 게임 발견과 제어의 불일치 / `--pid` 미지원

**실제 결과**

`hera game instances`에는 PID **25806**, 씬 `res://game/Main.tscn`이 보였지만, `hera game screenshot`은 다음 오류로 실패했다.

```text
game: no game is running; start one with `hera run --current --wait`
```

설치 CLI에 게임 PID를 지정하려 하면 다음과 같이 파싱 단계에서 실패했다.

```text
game: unknown game subcommand "--pid" (want tree|ui tree|instances|screenshot|click|input|input-log|assert|node get|node set|node call|qa)
```

증거: [instances](evidence/external-instances.json), [외부 게임 거부](evidence/external-screenshot-failure.txt), [CLI 파싱 실패](evidence/cli-pid-unsupported.txt).

**현재 소스에서 확인한 경로**

- [game_tool.gd](../../../addons/hera_agent_godot/tools/game_tool.gd) `execute_async`: `instances`만 먼저 반환하고, 다른 요청은 `EditorInterface.is_playing_scene()`가 false면 중단한다. 하트비트가 살아 있어도 외부 실행은 여기서 거절된다.
- 같은 파일 `_target_game`: 현재 editor-play 씬으로만 대상을 선택한다. 요청의 게임 PID를 선택하는 경로가 없다.
- [cmd/game.go](../../../cmd/game.go) `parseGameAction`: 첫 인자를 하위 명령으로 해석하며 `--pid` 분기가 없다.
- **`hera --instance <PID>`는 에디터 선택**이다. 게임 PID 선택을 대신하지 않는다.

**수정 후 확인할 재현 순서**

1. 격리된 프로젝트 복사본과 사용자 데이터에서 Hera가 켜진 에디터를 연다. 에디터에서는 게임을 재생하지 않는다.
2. 같은 복사본의 게임을 별도 Godot 프로세스로 실행하고 `game instances`에 나타나는 PID를 기록한다.
3. 현재의 무지정 요청이 위 오류를 내는지 확인한다.
4. 새 CLI·addon을 함께 적용한 뒤 명시한 게임 PID로 screenshot/tree/input 요청을 보낸다.
5. **응답 PID·씬·이미지 크기가 요청 대상과 일치하는지** 확인한다. CLI 성공 종료만으로 판정하지 않는다.

**구현 검토안 / 완료 기준**

- 자연스러운 CLI 구문 예: `hera --instance <EDITOR_PID> game --pid <GAME_PID> screenshot ...`.
- 기존 RPC `{"tool":"game","params":{"action":"screenshot","pid":123,...}}`에 대상 PID를 전달하고 모든 game 하위 명령/QA 경로에 일관되게 적용한다.
- PID 미지정 시 기존 단일 editor-play 동작 유지. 명시 PID가 없거나 만료됐을 때 다른 게임으로 자동 대체하지 않는다.
- 양의 정수 PID만 허용. JSON 숫자 `202.0`의 정상 처리, 문자열·소수·0·음수·존재하지 않는 PID의 거부도 확인한다.
- 역사적 임시 패치에서는 외부 PID의 1080×1920 캡처에 성공했다. [관측값](evidence/historical-pid-success.txt), [당시 결과표](evidence/historical-repair-report.md).

## H02. 여러 게임 프로세스와 요청 대상 혼선

**실제 오류**

```text
game: multiple Hera game processes found for scene res://game/Main.tscn (pid 64250, pid 64159); stop stale Godot game processes and retry
```

[원본 로그](evidence/ambiguous-target.txt). 다른 시도에서도 64998/64944, 67209/67183처럼 여러 PID 조합으로 반복됐다. 살아 있는 프로세스를 모두 “stale”로 볼 근거는 없다.

`_target_game()`는 씬이 같은 fresh 하트비트가 둘 이상이면 거부한다. 이 거부는 잘못된 프로세스에 입력하는 것을 막지만, 현재 CLI에서는 의도한 게임을 선택할 수 없어 작업이 막힌다. 또한 [run_tool.gd](../../../addons/hera_agent_godot/tools/run_tool.gd)의 `stop`은 `EditorInterface.stop_playing_scene()`를 호출하므로 별도 실행한 모든 런타임을 종료한다고 간주하면 안 된다.

**수정/검증 기준**

- H01의 명시 PID 선택을 추가하되 무지정 다중 대상은 계속 실패시킨다.
- `game instances` 및 실패 응답에서 선택 가능한 PID/씬/하트비트 상태를 알아볼 수 있게 한다.
- 첫 대상이 종료된 후 다른 동일 씬 프로세스의 응답이나 남은 response 파일을 받아들이지 않는다.
- 서로 다른 두 PID로 읽기/입력을 번갈아 보내며 대상 일치를 확인한다. 종료 작업은 자신이 시작한 PID에만 한다.
- 당시 정지된 런타임(SIGSTOP)은 약 **3.112초** 후 timeout으로 실패했다. 이는 실패 처리를 검증하기 위해 만든 상황이며 별도의 자연 발생 고장으로 집계하지 않는다. [로그](evidence/historical-timeout.txt).

## H03. embedded 뷰포트 크기와 기대 해상도 불일치

**관측**: `hera run --scene res://game/Main.tscn --wait` 후 embedded screenshot은 **927×1649**였다. 프로젝트/실행 인자의 목표는 **1080×1920**이었다. 캡처는 nonblank였고 analyzer의 `possible_clipping`은 false였다. [응답 전체](evidence/embedded-screenshot.json).

이 차이 때문에 1080×1920을 전제로 작성한 입력 좌표나 픽셀 검증을 그대로 사용하면 안 됐다. **`possible_clipping=false`는 요구 해상도를 만족했다는 뜻이 아니다.**

[game_viewport_actions.gd](../../../addons/hera_agent_godot/runtime/game_viewport_actions.gd)의 screenshot은 실제 `viewport.get_texture().get_image()` 크기를 반환한다. Hera가 임의 축소했다고 확정할 근거는 없다. 당시 embedded 창 크기 및 macOS titled-window 제약이 의심됐고, borderless 외부 launcher + 명시 PID 경로에서는 1080×1920을 얻었다. 정확한 원인은 환경을 통제해 다시 비교해야 한다.

**검증안**: 동일 씬의 embedded / 별도 titled / 별도 borderless 실행에서 요청 크기, window 크기, visible rect, PNG IHDR를 함께 수집한다. 원하는 크기가 불가능하면 요청/실제 크기를 명시적으로 알려야 하며, 작은 이미지를 사후 확대해 요구 해상도의 실제 렌더로 취급하지 않는다. 게임 입력 좌표도 보고된 실제 뷰포트 기준임을 문서화한다.

## H04. 에디터 생존 상태의 타임아웃·하트비트 만료

**2026-09-04 06:01 UTC 세션 원문**에서 다음 순서가 확인된다.

1. 에디터를 열어 둔 상태에서 같은 프로젝트에 headless 실행 및 폰트 재임포트를 수행했다.
2. `hera project reimport`가 `context deadline exceeded (Client.Timeout exceeded while awaiting headers)`로 실패했다.
3. 이후 scene/editor/screenshot/status가 `no live Godot editor found`를 반환했다.
4. OS에는 에디터 PID **33516**이 남아 있었지만, `hera instances`는 `count:0`이었다.
5. 하트비트 `ts=1788501662`, 확인 시각 `1788501716`: **54초 차이**. 후속 설명에서는 1분 이상 정지로 보고했다.

[원문 발췌](evidence/editor-stall-session.txt). 당시 Claude 세션 ID `a86288c1-4648-4228-ac27-21a7d888f7ab`, raw lines 1820/1826/1843/1852. 원래 JSONL 경로는 Monstel의 로컬 Claude projects 저장소다.

**확정한 것**은 RPC 무응답과 stale 하트비트다. `.godot` 캐시의 동시 접근, 재임포트/메인 스레드 정지, 모달 창 중 무엇이 원인인지는 스택/락 추적이 없어 미확정이다. “Hera가 에디터를 멈췄다” 혹은 “캐시 충돌이 확정됐다”로 인용하지 말 것.

관련 지점: [plugin](../../../addons/hera_agent_godot/hera_agent_plugin.gd)의 `_process` heartbeat 갱신, [server/heartbeat.gd](../../../addons/hera_agent_godot/server/heartbeat.gd), [internal/discovery/discovery.go](../../../internal/discovery/discovery.go)의 5초 freshness 필터, RPC timeout.

**후속 조사**: 복사본에서만 단독 에디터 / 단독 headless / 동시 실행을 비교하고, 정지 시 에디터 스택·모달 유무·마지막 heartbeat·RPC 상태를 수집한다. CLI가 만료된 인스턴스와 아예 없는 인스턴스를 진단상 구분하면 도움이 된다. 현재 Monstel 운영 규칙은 같은 프로젝트의 에디터와 headless를 동시에 실행하지 않는 것이다.

## H05. Android export에 남은 Hera autoload

**9월 7일 실기 재현 로그**:

```text
ERROR: Unrecognized UID: "uid://c4ug7a211oav8".
ERROR: No loader found for resource: res:// (expected type: unknown)
ERROR: Failed to instantiate an autoload, can't load from path: .
```

그 뒤 `SimCheck OK`와 `loaded save`가 출력되며 게임은 실행됐다. **이 빌드에서 시작을 막지 않았다는 뜻이며, 모든 플랫폼/빌드에서 무해하다는 보장은 아니다.** [실기 로그](evidence/android-startup.log).

**현재 설정과 소스 연결**

- Monstel `project.godot`: `HeraGameInspector="*uid://c4ug7a211oav8"`.
- 해당 UID는 `addons/hera_agent_godot/runtime/game_inspector.gd.uid`와 일치한다.
- Android preset `exclude_filter`에 `addons/*`가 있다. Hera runtime 리소스는 제외되는 반면 project autoload 참조는 남는다.
- plugin `_ensure_game_autoload()`는 설정이 이미 있으면 그대로 반환한다. `_exit_tree()`에서 제거하는 조건은 **이번 plugin 객체가 추가했음을 나타내는 `_game_autoload_injected`**뿐이다. 이미 저장된/이전 실행에서 남은 항목의 수명은 이 조건만으로 처리되지 않는다.
- 잔존 설정과 제외 정책의 불일치는 확인됐지만, 최초 잔존을 만든 종료·재시작 순서는 아직 재현하지 않았다.

**수정 검토**

- editor-only 개발용 inspector의 등록/해제 및 export 계약을 명시한다.
- production export에는 autoload 참조와 리소스가 함께 빠지거나, 지원하는 debug export에서는 둘 다 들어가도록 한다. 단순히 UID를 `res://` 문자열로 바꾸는 것만으로 리소스 제외 문제는 해결되지 않는다.
- 기존 설정을 정리할 때 같은 이름의 사용자 소유 autoload를 무조건 삭제하지 않도록 소유/경로를 확인한다.

**완료 기준**: enable → 저장 → 정상 종료 → 재실행 → disable, headless export, 예전 잔존 설정, UID cache 재생성, addon 제외/포함의 조합에서 설정을 비교한다. Android debug/release 시작 로그에 위 세 오류가 없어야 하고, 개발 환경의 Hera runtime 제어도 유지돼야 한다.

## H06. 병렬 QA와 공유 사용자 데이터

당시 여러 게임이 동일 `user://`의 `save.json`을 사용했다. 역사적 수리 보고서는 다른 embedded 런타임의 쓰기를 관찰했다고 기록했으며, 격리 QA 전후 실제 세이브 해시가 달라졌다. project.godot과 editor settings의 해시는 같았다. [관측 발췌](evidence/shared-save-observation.txt).

해시 변화만으로 어느 PID가 썼는지 증명할 수는 없다. **Hera가 직접 세이브를 손상시켰다는 증거도 없다.** 다만 입력/정지 대상을 혼동하거나 다른 런타임이 실행 중인 상태에서 오래된 백업을 덮어쓰면 최신 진행을 잃을 수 있는 작업 충돌이다.

후속 테스트는 프로젝트 복사본과 별도 사용자 데이터·discovery 저장소를 함께 격리한다. `--instance`만으로 저장 경로가 격리된다고 가정하지 않는다. macOS와 Windows의 사용자 디렉터리 규칙도 실제 파일 경로로 검증한다. 현재 레포 [DEV_MACHINE.md](../../DEV_MACHINE.md)의 격리 smoke 지침을 따른다. 테스트가 만든 PID만 종료하고, 기존 사용자의 진행을 과거 스냅샷으로 되돌리지 않는다.

## H07. export 종료 로그: 원인 분리 필요

export 완료 뒤 다음 로그가 반복됐다.

```text
[hera] Hera Agent Godot exited
ERROR: EditorSettings not instantiated yet when getting setting "export/android/shutdown_adb_on_exit".
WARNING: 45 ObjectDB instances were leaked at exit
ERROR: 22 resources still in use at exit
```

[export 발췌](evidence/export-tail.log). 당시 export 명령은 종료 코드 0이었고 APK 설치/실행도 성공했다. Hera 종료 메시지 다음에 찍혔다는 순서만으로 Hera 누수라고 판정할 수 없다. engine Android exporter/종료 순서/다른 addon 등과 분리해야 한다.

새 최소 프로젝트에서 Hera 없음 / Hera enabled / clean disable / 잔존 autoload를 비교하고, 동일 엔진·동일 export preset으로 `--verbose` 객체 목록을 수집한다. 차이가 없으면 Hera 수정 대상에서 분리한다.

## 당시 패치의 범위와 재사용 주의

- [historical-partial.patch](evidence/historical-partial.patch): Monstel에서 따로 보관한 README + 게임 PID 선택 부분. **README가 참조하는 `external_launcher.gd`는 이 작은 패치에 포함되지 않는다.**
- [historical-full.patch](evidence/historical-full.patch): 그 누락분까지 포함한 당시 전체 diff. launcher, addon 테스트, 프로젝트 전용 Python CLI wrapper/symlink, 임시 RPC bridge도 들어 있다.
- [PID 선택 테스트](evidence/test_game_target_selection.gd), [launcher 인자 테스트](evidence/test_external_launcher_args.gd)는 당시 자료를 그대로 복사했다. 현행 checkout에 필요한 메서드/launcher가 없으므로 즉시 실행 가능한 현행 테스트라고 주장하지 않는다.
- Monstel의 `tools/hera.py`, `tools/hera`, addon PID 변경은 이후 프로토 범위에서 철회됐다. **설치된 CLI나 현재 addon에 반영 완료된 상태가 아니다.**
- 재사용할 것은 선택 계약·실패 사례·검증 조건이다. Hera 본체 수정은 Go CLI와 addon에서 수행하고, 프로젝트용 wrapper를 그대로 제품 기능으로 편입하지 않는다.
- 현재 로컬 addon에는 후속 변경이 있어 전체 diff의 무검토 적용은 피한다. PID 값들은 당시 기록이지 지금 제어할 대상이 아니다.

## 수정 세션 시작용 프롬프트

```text
이 레포의 docs/incidents/monstel-2026-09-07/README.md와 evidence를 먼저 읽어라.
이는 Monstel 연동에서 나온 관측 기록이며 아직 Hera 수정이 완료됐다는 뜻이 아니다.
H01/H02(게임 PID 선택)와 H05(autoload/export 수명)를 우선 검토하라.
현재 코드와 옛 패치의 차이를 확인하고, 격리된 최소 프로젝트에서 실패를 재현한 뒤
Go CLI·GDScript addon·계약/문서를 함께 수정하라. 에디터 PID와 게임 PID를 구분하고
모호하거나 만료된 대상에 대한 거부를 유지하라. H03/H04/H07은 원인 미확정이므로
로그 순서나 당시 추정만으로 Hera 결함이라고 단정하지 마라.
실사용 Monstel 프로젝트·세이브·열려 있는 사용자 에디터는 재현 실험에 쓰지 마라.
```

이번 산출물은 문서와 증거 사본이다. 게임 코드·Hera 코드·설치 CLI를 수정하거나 커밋/푸시하지 않았다.
