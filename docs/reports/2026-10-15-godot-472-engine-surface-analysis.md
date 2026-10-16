# Godot 4.7.2 엔진 바이너리 분석 — Hera가 추가/보강할 수 있는 기능

- 날짜: 2026-10-15
- 대상: `C:\Users\PC\Desktop\Godot_v4.7.2-stable_mono_win64\Godot_v4.7.2-stable_mono_win64.exe`
  (181,449,736 bytes, PE x86_64, SHA-256 `45336315eb6f1a52a8923bc4f2ce8079a03dc4939dcb7d531047890f1f7cdfab`,
  FileVersion `4.7.2`, ProductVersion `4.7.2.stable.mono.official`, 엔진 커밋 `ed1daf0bf001b61586d9930840f2f1394092c079`)
- 보조 대상: 같은 디렉터리의 `Godot_v4.7.2-stable_mono_win64_console.exe`(콘솔 캡처용, 198,152 bytes),
  `GodotSharp/Api/{Debug,Release}/GodotSharp.dll`(6.07/5.65 MB), `GodotSharpEditor.dll`(0.31/0.29 MB),
  `GodotSharp.xml`(15.63 MB), `GodotSharp/Tools/GodotTools.dll`(174,080 bytes, SHA-256 `d9a463c71f98d67abf38569cbbdd13437bb236a798eaa8e31990ff98d7ca3322`)
- 목적: **에이전틱 게임 개발 관점에서** 이 엔진 바이너리가 이미 제공하지만 Hera가 쓰지 않는 표면을 찾아내고,
  Hera를 "Godot 공식 CLI 도구" 수준으로 승격시키기 위한 기능/보강/강화 목록을 근거와 함께 제시한다.

---

## 0. 요약 (먼저 읽을 것)

**한 줄 결론:** 엔진은 이미 *에이전트가 필요로 하는 것의 대부분*을 갖고 있다 — 디버그 어댑터(GDScript DAP),
언어 서버(LSP), 원격 디버거 채널(`scene:*` 실시간 노드 편집/스크린샷/프레임 스텝/프로파일러), 결정론적
동영상 캡처(`--write-movie`), 노에디터 스크립트 실행(`--headless --script`), 임포트/익스포트, 오프라인
ClassDB·API 문서, 벤치마크 JSON. **Hera의 다음 도약은 새 기능 발명이 아니라 이 표면을 안전·저토큰·증거
중심으로 "포장"하는 것**이다.

가장 큰 6개 기회(근거 등급: 실측=프로브 실행, 문자열=바이너리 내 리터럴 위치 확인):

| # | 후보 | 엔진 근거 | 등급 | Hera 현재 |
|---|---|---|---|---|
| 1 | **GDScript DAP 디버깅** (`--dap-port`, 브레이크포인트/스택/변수/evaluate/setVariable) | 실측(리스너 + initialize 능력 협상) | 높음 | 없음 (ROADMAP 후보로만 존재) |
| 2 | **노에디터 실행 계층** (`--headless --script`, `--import`, `--scene`, `--quit-after`, `-- log args`) | 실측 (exit 0, 사용자 인자 전달 확인) | 높음 | `script validate`만 엔진 스폰 |
| 3 | **결정론적 캡처/재생** (`--write-movie` .avi/.png + `--fixed-fps`) | 실측 (10프레임 AVI 80,680B, PNG 시퀀스+WAV) | 높음 | 라이브 창 스크린샷만 |
| 4 | **오프라인 지식 산출** (`--class-db-json`, `--doctool`, `--dump-extension-api-with-docs`) | 실측 (6.15MB/1,077 XML/11.87MB JSON) | 높음 | 라이브 에디터 `classdb`만 |
| 5 | **LSP 진단(미저장 버퍼)** | 실측(오류 3건 수신) | 높음 | `script validate`는 디스크만 |
| 6 | **런타임 인스펙터를 엔진 프로토콜로 승격** (`EditorDebuggerPlugin` + `scene:*`) | 문자열(채널 70여 종) + API 문서 | 중간 | 자체 autoload 주입(프로젝트 무수정 불가) |

그리고 "공식 도구 승격"을 위해 반드시 지켜야 할 함정 5가지(§8): 로그 텍스트 파싱 금지(로컬라이즈됨),
`--headless`에서 렌더 기대 금지(dummy 렌더러), `--headless + --write-movie` 크래시, `--editor --script`로
EditorScript 실행 불가, `--main-loop`(GDScript 클래스) 불가.

---

## 1. 조사 방법과 증거 등급

| 방법 | 도구 | 산출 | 한계 |
|---|---|---|---|
| 네이티브 바이너리 열기 | REA `open_binary` (providers: ghidra 12.1.4) | PE/x86_64 확인, SHA-256 | Ghidra 심층분석은 **6회 시도 전부 330초 타임아웃**(181MB 자동분석) — 의사코드/xref 증거는 얻지 못함. 대신 REA 매니지드 분석 + 자체 아티팩트 스캔 + 실행 프로브로 대체 |
| 매니지드 어셈블리 | REA `inspect_managed_members` (`rea-dotnet-static`) | `GodotTools.dll`: 141 types / 721 MethodDef | `GodotSharp.dll`은 "CLI metadata could not be admitted" 로 거부 → .NET API는 동봉 `GodotSharp.xml` 문서로 대체 확인 |
| 아티팩트 직접 스캔 | 자체 스크립트(PE 섹션 파싱 + ASCII 리터럴 추출) | 고유 문자열 682,477개, `--flag` 토큰 180개, 주요 리터럴의 **섹션/RVA** | 리터럴 존재는 "그 이름이 배포물에 있다"까지만 증명(동작 증명 아님) |
| 배포물 실행(제한적 재현) | 콘솔 exe, 격리 `HOME`/`USERPROFILE`/`APPDATA`, 런별 타임아웃 | 7개 라운드 프로브(아래 부록 A) | 4.7.2-모노-윈도우 단일 환경, 1대의 PC |

증거 등급 표기: **[실측]** = 이 바이너리를 직접 실행해 관찰, **[문자열]** = 리터럴이 섹션/RVA와 함께 확인됨,
**[문서]** = 배포물에 동봉된 문서(`GodotSharp.xml`)에서 확인, **[미확인]** = 확인하지 못함(재현 방법 제시).

---

## 2. 이 빌드의 CLI 표면 (전부 `--help` 관찰 + 프로브 실측)

### 2.1 빌드 가용성 범례 — Hera가 반드시 구현해야 하는 게이트

이 빌드의 `--help`는 모든 플래그에 적용 범위를 명시한다:

- `R` = editor 빌드 + debug/release **export template** 모두
- `D` = editor 빌드 + **debug** export template만
- `X` = editor 빌드 + `disable_path_overrides=false`로 컴파일된 템플릿만
- `E` = **editor 빌드 전용** (DAP/LSP/export/import/doctool 모두 여기)

→ Hera가 "엔진 CLI 래퍼"를 만들 때 **사용자의 엔진이 editor 빌드인지 템플릿인지에 따라 명령 가용성이
달라진다**. 게이트 없이 명령을 노출하면 "왜 안 되지"를 에이전트가 추측하게 된다. (현재 Hera는
`status.csharp_supported` 같은 단일 능력만 노출)

### 2.2 에이전트 관점 핵심 플래그 (실측/문자열 확인)

실행/결정론:
- `--scene <path|UID>` **[실측]** — 프로젝트 설정을 건드리지 않고 임의 씬 실행
  (d2: `res://alt.tscn` 실행 성공 / 6라운드: `uid://heraprobe6scene`으로 실행 성공, exit 0)
- `--headless` (= `--display-driver headless --audio-driver Dummy`) **[실측]**
- `--quit`, `--quit-after <int>` **[실측]** — 프레임 단위 종료(결정론적 경계)
- `--fixed-fps <fps>`, `--time-scale <scale>`, `--delta-smoothing enable|disable`, `--disable-vsync`,
  `--single-threaded-scene`, `--render-thread safe|separate` **[문자열]** — 재현성 파라미터
- `--` / `++` **[실측]** — 이후 인자는 엔진이 먹지 않고 `OS.get_cmdline_user_args()`로 전달 (d3 확인)
- `--main-loop <class>` **[실측 실패]** — GDScript `class_name` MainLoop로는 동작하지 않음(90초 무응답)
- `-s, --script <script>` **[실측]** — `SceneTree` 상속 스크립트가 사실상의 "노에디터 하니스" 진입점 (d3/d5)
- `--check-only` **[실측]** — 구문만 검사 (Hera `script validate`가 이미 사용)

도구:
- `--import` **[실측]** — 에디터 임포트 파이프라인을 헤드리스로 완료 후 종료 (exit 0)
- `--export-pack|--export-release|--export-debug|--export-patch <preset> <path>`, `--patches` **[실측/부분]**
  — `--export-pack`은 **템플릿 없이 성공**(pck 2,024B / zip 1,841B), `--export-release`는 템플릿 필요(아래 §8)
- `--doctool [path]`, `--no-docbase`, `--gdextension-docs`, `--gdscript-docs <path>` **[실측]**
  — `--doctool` 1,077개 XML(4.12MB) 생성 성공, `--gdscript-docs`는 **프로젝트 상대(`res://`) 경로**로 호출해야 하며
  스크립트 1개당 XML 1개를 만든다(§3.5, §7-5)
- `--dump-extension-api`, `--dump-extension-api-with-docs`, `--dump-gdextension-interface(-json)`,
  `--validate-extension-api <path>` **[실측]** — `extension_api.json` 11,871,018B(클래스 1,037개·문서 1,015개·싱글턴 41개)
- `--benchmark`, `--benchmark-file <path>` **[실측]** — 단계별 소요시간 JSON (예: `"[Servers] Rendering]": 0.728383`)
- `-e/--editor`, `-p/--project-manager`, `--recovery-mode`, `--verbose`, `--quiet`, `--no-header` **[실측]**
- `--log-file <file>` **[문자열]** — `user://logs/godot.log` 대신 지정 경로에 로그
- `--install-android-build-template`, `--list-sdks` **[문자열]** — 안드로이드 파이프라인
- C#: `--build-solutions`, `--generate-mono-glue` **[문자열]**

디버깅:
- `--dap-port <port>` **[실측]** — GDScript Debug Adapter Protocol 포트 개방 확인 (6026 리스너 open)
- `--lsp-port <port>` **[실측]** — GDScript Language Server 포트 개방 확인 (6025 리스너 open)
- `--debug-server <uri>` **[문자열]** — 에디터가 디버그 서버 시작 (`tcp://127.0.0.1:6007` 예시가 도움말에 있음)
- `--remote-debug <uri>` **[문자열]** — 실행 중 게임이 지정 디버거로 접속
- `-b, --breakpoints src::line,src::line` **[문자열]** — **비대화식 브레이크포인트 주입** (도움말에 정확한 포맷 명시)
- `--skip-breakpoints`, `--ignore-error-breaks`, `--debug-mute-audio` **[문자열]**
- `-d/--debug`, `-b`, `--profiling`, `--gpu-profile`, `--gpu-validation`, `--gpu-abort`,
  `--accurate-breadcrumbs`, `--extra-gpu-memory-tracking`, `--generate-spirv-debug-info` **[문자열/도움말]**
- `--debug-collisions|--debug-paths|--debug-navigation|--debug-avoidance|--debug-canvas-item-redraw|--debug-stringnames` **[도움말]**
- `--max-fps`, `--frame-delay`, `--print-fps`, `--editor-pseudolocalization`, `--accessibility <mode>`,
  `--accessibility-driver <accesskit|dummy>`, `-l/--language <locale>` **[도움말/문자열]**

### 2.3 `--help`에 없는(숨은) 플래그 — Hera가 활용/주의할 것

| 플래그 | 표시되는 흔적 | 해석 |
|---|---|---|
| `--editor-pid <pid>` | `Missing editor PID argument, aborting.` (RVA 0x71574ce 근처), 문자열 RVA 0x681c846 | 부모 에디터 PID 전달용. 부모 종료 시 자동 종료 계열로 추정 — **에이전트가 띄운 에디터를 회수**할 때 유용 |
| `--class-db-json` | `user://class_db_api.json` / `user://class_db_api_editor.json`와 인접(RVA 0x6f4a79a/0x6f4a7c8) — **[실측] 동작 확인** | C# 글루 생성용이지만 기능적으로는 **오프라인 ClassDB JSON 덤프**. node 없이 6,152,199B + 192,716B 생성 |
| `--test-rd-support`, `--test-rd-creation`, `--debug-mute-audio`, `--skip-breakpoints`, `--generate-mono-glue`, `--enable-on-demand-resources`, `--lightweight-asset-runtime-mode`, `--new-display`, `--no-vd-system-decorations`, `--display-ime-policy`, `--embedded`, `--start-app` | 인자 파싱 실패 메시지/플랫폼 코드 문자열 | **플래그로 확인된 것**(`--test-rd-*`, `--debug-mute-audio`, `--skip-breakpoints`)과 **익스포트 플랫폼 옵션 이름일 가능성이 있는 토큰**(나머지)이 섞여 있음 — Hera는 쓰지 말고 진단에서만 언급 |
| 플랫폼 익스포트 옵션 다수 (`--p12-file`, `--p12-password`, `--ks-pass`, `--ks-key-alias`, `--api-key`, `--api-issuer`, `--api-key-path`, `--entitlements-xml-path`, `--code-signature-flags`, `--target-device`, `--minimum-deployment-target`, `--binary-identifier`, `--development-region`, `--list-sdks`, `--include-all-app-icons`, `--output-partial-info-plist`, `--hdr`, `--pbr-metallic-roughness`) | 플랫폼 익스포트 코드 문자열 | **시크릿 취급 대상**. Hera의 export 래핑은 이 값을 절대 출력/기록하면 안 됨 |

---

## 3. 엔진이 이미 제공하는 "에이전트급" 서비스

### 3.1 GDScript 디버그 어댑터 (DAP) **[실측 + 문자열]**

- 실행: 에디터에 `--dap-port <port>` → TCP 리스너 개방 (프로브 3: `open=True`).
- **실측 세션(프로브 4, 에디터 `--dap-port 6046`)**: 표준 DAP 프레이밍(`Content-Length`)으로 접속 →
  `initialize` 응답 `success: true`, 직후 `initialized` 이벤트 수신. 서버가 스스로 밝힌 능력:
  `supportsConfigurationDoneRequest`, **`supportsSetVariable`**, **`supportsEvaluateForHovers`**,
  `supportsRestartRequest`, `supportsTerminateRequest`, `supportsBreakpointLocationsRequest`,
  `supportsValueFormattingOptions`, `supportedChecksumAlgorithms: ["MD5","SHA256"]`,
  `supportSuspendDebuggee`, `supportTerminateDebuggee`.
  → 즉 에이전트는 중단점/스택/변수 읽기뿐 아니라 **실행 중 변수 대입(setVariable)**과 **표현식 평가(evaluate)**까지
  엔진 표준 프로토콜로 할 수 있다.
- **미확인/주의**: DAP `launch`(게임 시작) 요청은 **헤드리스 20초·창 모드 45초 모두 응답이 없었다**(프로브 4/7).
  대신 **`attach` 경로가 실측으로 검증됐다**(§8 표): 에디터 `--debug-server` + 게임 `--remote-debug` + DAP `attach`.
  → Hera의 DAP 통합은 `launch` 대신 이 경로를 쓴다(Hera가 게임 실행과 디버거 연결을 각각 제어할 수 있어 더 적합하다).
- **엔진 버그 관찰(중요)**: `setBreakpoints` 요청에 `checksums`를 넣지 않으면 에디터가
  `ERROR: Bug: Dictionary::operator[] used when there was no value for the given key "checksums". Please report.`를
  로그에 남긴다(응답 자체는 성공). 또 DAP 중단점이 편집기 설정에 저장되어
  `Couldn't find the given section "res://main.gd" and key "state"` 노이즈가 함께 나온다 **[프로브 7]**
  → Hera의 DAP 클라이언트는 **`checksums`(MD5/SHA256)를 함께 보내야** 하고, 로그 오염을 진단에 포함해야 한다.
- 구현 흔적: `DebugAdapterServer`, `DebugAdapterParser`, `DebugAdapterProtocol`,
  `editor/debugger/debug_adapter/{debug_adapter_parser.cpp,debug_adapter_protocol.cpp,debug_adapter_types.h}`,
  `[DAP] Connection Taken`, `Failed to take incoming DAP connection.`,
  `prepare_error_response(p_params, DAP::ErrorType::UNKNOWN)`.
- 핸들러 이름 토큰(`req_*`): `initialize, launch, attach, restart, terminate, configuration, disconnect,
  set_breakpoint, breakpoint, pause, continue, next, step, threads, stack, scopes, variables, evaluate,
  permission, godot` — 즉 Godot 전용 확장 요청(`godot`)까지 포함. 여기서 `launch`만 실측에서 무응답이었다(아래).
  ※ 이 토큰들은 `.rdata`의 소문자 바이트 열로 확인했고, 리터럴/심볼 중 무엇인지까지는 확정하지 못했다.
  다만 바로 위 실측 세션 결과가 기능 자체를 뒷받침한다.
- 설정 키: `network/debug_adapter/{remote_port,request_timeout,sync_breakpoints}`.

### 3.2 GDScript 언어 서버 (LSP) **[실측 + 문자열]**

- 실행: `--lsp-port <port>` **[실측: 리스너 open]**.
- **실측 세션(프로브 4, 에디터 `--lsp-port 6045`)**:
  1. 접속 직후 서버가 `gdscript_client/changeWorkspace`(프로젝트 경로)를 보낸다 — Godot 고유 확장 알림.
  2. **디스크에 존재하지 않는 버퍼**를 `textDocument/didOpen`으로 열고 오류를 심은 결과 `publishDiagnostics` 3건 수신:
     `Cannot assign a value of type "String" as "int".`(4행), `Cannot assign a value of type String to variable "x" with specified type int.`(4행),
     `Function "undefined_function_call()" not found in base self.`(5행) — **severity 1**.
     → Hera 문서의 기존 한계("script validate는 미저장 편집 텍스트를 검사하지 않는다")를 **LSP 경로로 해소 가능**.
  3. `textDocument/documentSymbol`이 구조화 심볼(`detail: "var counter: int = 0"`, `kind: 13`, range/selectionRange) 반환.
  4. `textDocument/hover`는 해당 위치에서 빈 `contents`였다(위치에 따라 비어 있을 수 있음).
- 구현: `GDScriptLanguageServer`, `GDScriptLanguageProtocol`, `GDScriptTextDocument`, `GDScriptWorkspace`,
  `TCPServer` + `JSONRPC` 기반, `[LSP] Connection Taken` / `[LSP] Disconnected`.
- 지원 메서드(문자열 전체 확인): `initialize`, `initialized`,
  `textDocument/{didOpen,didClose,didChange,willSaveWaitUntil,didSave,documentSymbol,documentHighlight,
  completion,rename,prepareRename,references,foldingRange,codeLens,documentLink,colorPresentation,hover,
  definition,declaration,signatureHelp,nativeSymbol}`, `textDocument/publishDiagnostics`.
- Godot 고유: **`textDocument/nativeSymbol`** (네이티브/엔진 심볼 문서 조회),
  `network/language_server/{enable_smart_resolve,show_native_symbols_in_editor}`.
- 한계(문자열로 확인): 동시 클라이언트 **최대 8개**(`clients.size() >= 8` → "Max client limits reached"),
  `LSP: The language server only supports the file protocol:`, `LSP: The language server does not support nonlocal files:`,
  **"The GDScript Language Server might not work correctly with other projects than the one opened in Godot."**

### 3.3 원격 디버거 채널 — Hera 런타임 인스펙터의 "엔진 판" **[문자열]**

바이너리에서 확인된 채널 이름(전부 `채널:메시지` 형태 리터럴):

- 씬/런타임 트리: `scene:scene_tree`, `scene:request_scene_tree`, `scene:inspect_object`, `scene:inspect_objects`,
  `scene:set_object_property`, `scene:set_object_property_field`, `scene:save_node`, `scene:clear_selection`
- **실행 중 게임 편집**: `scene:live_create_node`, `scene:live_duplicate_node`, `scene:live_instantiate_node`,
  `scene:live_remove_node`, `scene:live_remove_and_keep_node`, `scene:live_reparent_node`, `scene:live_restore_node`,
  `scene:live_set_root`, `scene:live_node_prop`, `scene:live_node_prop_res`, `scene:live_node_call`, `scene:live_node_path`,
  `scene:live_res_prop`, `scene:live_res_prop_res`, `scene:live_res_call`, `scene:live_res_path`
- 프레임/시간/제어: `scene:next_frame`, `scene:speed_changed`, `scene:suspend_changed`, `scene:debug_mute_audio`
- 카메라/뷰: `scene:transform_camera_2d`, `scene:transform_camera_3d`, `scene:override_cameras`,
  `scene:rq_screenshot`, `scene:runtime_node_select_setup|set_mode|set_type|set_visible|set_prefer_group|set_avoid_locked|reset_camera_2d|reset_camera_3d`
- 창/포커스/HDR: `scene:window_request_size`, `scene:report_window_focused`, `scene:hdr_output_request_state`,
  `scene:hdr_output_toggle_requested`, `scene:reload_cached_files`, `scene:click_ctrl`
- 게임 뷰(에디터 내장 실행): `scene:setup_game_view`, `scene:setup_embedded_shortcuts`,
  `game_view:{setup_complete,window_size,get_screenshot,hdr_state}`
- 프로파일러/서버: `profiler:visual`, `profiler:servers`, `servers:{memory,memory_usage,draw,drawn,foreground,
  function_signature,profile_frame,profile_total}`, `performance:{profile_frame,profile_names}`
- 네트워크(멀티플레이어 프로파일러): `multiplayer:{bandwidth,cache,replication,rpc,syncs}`
- 파일: `filesystem:update_file`

이 채널들을 **플러그인에서 받아쓰는 API**도 배포물에 있다(문자열/문서 확인):
`EditorDebuggerPlugin`(`get_sessions`, `get_session`, `add_session_tab`, `has_capture`/`_has_capture`, `_capture`,
`_setup_session`, `_clear`, `_goto_script_line`, `_breakpoint_set_in_tree`), `EditorDebuggerSession`,
그리고 게임측 API `EngineDebugger.{RegisterProfiler,RegisterMessageCapture,SendMessage,IsActive,InsertBreakpoint,
RemoveBreakpoint,ClearBreakpoints,IsBreakpoint,SetLinesLeft,…}` **[문서: GodotSharp.xml]**.

동봉 문서(`GodotSharpEditor.xml`)가 계약을 명확히 밝힌다 — 이는 B3 제안의 직접 근거다:

> `EditorDebuggerPlugin`은 `EditorPlugin.add_debugger_plugin()`으로 등록하고,
> `_setup_session(id)`가 세션마다 호출되며, `_has_capture("my_message")`를 true로 만들면
> **`"my_message:"`로 시작하는 모든 메시지가 `_capture(message, data, sessionId)`로 전달된다**.
> (인식한 메시지에 대해 true를 반환)

즉 `_has_capture("scene")` 한 줄로 `scene:*` 채널 전체를 받을 수 있고, 세션(`EditorDebuggerSession`)을 통해
송신도 가능하다. Hera가 자체 autoload를 주입하지 않고도 실행 중 트리에 접근할 수 있는 **공식 경로**다.

### 3.4 결정론적 캡처/동영상 **[실측]**

- `MovieWriter` 구현 3종(문자열): `MovieWriterMJPEG`(→ 검증된 `.avi`), `MovieWriterPNGWAV`(→ 검증된 `.png` 시퀀스 + `.wav`),
  `MovieWriterOGV`(OGV/Theora 계열로 **추정**, 미검증).
- 실측: `--write-movie out.avi --fixed-fps 30 --quit-after 10` → exit 0, 80,680B AVI, 로그에
  `10 frames at 30 FPS … recorded in 00:00:01 (100% of real-time speed)`, CPU/GPU/인코딩 시간 표기.
- 실측: `--write-movie frame.png --fixed-fps 30 --quit-after 5` → `frame00000000.png`~`frame00000004.png`(각 686B) + `frame.wav`(64,044B).
- 디스크 부족 경고 문자열: `Current available space on disk is low (%s). MovieWriter will fail during movie recording…`

### 3.5 오프라인 지식 산출물 **[실측]**

| 명령 | 결과 | 내용 |
|---|---|---|
| `--headless --class-db-json` | `user://class_db_api.json` 6,152,199B + `class_db_api_editor.json` 192,716B | 클래스별 `inherits/constants/methods[{name,arguments[{name,type,hint,hint_string}],default_arguments,return_type,hint_flags}]` — **에디터 전용 클래스(`EditorDebuggerPlugin`, `EditorExportPlatform`, `EditorDock`, `EditorCommandPalette`…)까지 포함** |
| `--headless --doctool <dir>` | XML 1,077개 / 4,121,736B | 엔진 클래스 레퍼런스 (`@GlobalScope.xml` 66,827B 등) |
| `--headless --dump-extension-api-with-docs` | `extension_api.json` 11,871,018B | 헤더(4.7.2 stable official, precision single), 클래스 1,037(문서 1,015), 싱글턴 41, 빌트인 38, 메서드별 `description` |
| `--headless --doctool <dir> --no-docbase --gdscript-docs res://` | `main.gd.xml` 882B (스크립트 1개당 1개) | **프로젝트 소스의 API 문서**: `brief_description`(=`##` 주석), `<method name="bump">` + `<param index="0" name="amount" type="int" default="1"/>` + 설명 자동 생성. `res://`(프로젝트 상대) 경로 필수 |
| `--headless --dump-gdextension-interface-json` | `gdextension_interface.json` 328,021B | GDExtension 바인딩 인터페이스 |
| `--benchmark-file <path>` | JSON 1,081B | 단계별 초 단위 시간(시작/서버/씬 비용) |

### 3.6 그 밖에 Hera가 알아야 할 엔진 내부 표면 **[문자열]**

- **에디터 실행 바/게임 뷰**: `editor/run/{editor_run.cpp,editor_run_bar.cpp,editor_run_native.cpp,embedded_process.cpp,
  game_view_plugin.cpp}`, `editor/run_project|run_current_scene|run_specific_scene`, `editor/run/main_run_args`
  → 게임을 **에디터 창 안(embedded)**에서 실행하는 경로가 존재. Hera `game screenshot`이 보고하는
  `embedded` 크기/`shared_user_data` 판단과 직결.
- **플러그인 관리**: `editor_plugins/enabled`, `enable_pending_addons`, `Disabling the addon at '%s' to prevent further errors.`,
  `Unable to load addon script from path: '%s'. Script is not in tool mode.`, `restart_editor`
  → 프로젝트 파일을 직접 고치지 않고 **플러그인 활성/비활성 진단**이 가능.
- **설정 오버라이드**: `res://override.cfg`, `Found override.cfg but could not load it.`
  → 프로젝트 설정을 영구 변경하지 않는 임시 오버라이드 경로.
- **캐시/UID**: `.godot/{uid_cache.bin,imported,shader_cache,export_credentials.cfg,editor/*}`,
  `main.gd.uid`/`second.gd.uid`(20B, 임포트가 생성) **[실측]**, `Missing .uid file for path "%s". The file was re-created from cache.`
- **로그**: `debug/file_logging/{enable_file_logging,log_path,max_log_files}`, `user://logs/godot.log`,
  `application/run/{disable_stdout,disable_stderr,print_header}`, `network/limits/debugger/*`
  → Hera `output`/`diagnostics`가 의존하는 파일 로깅은 **프로젝트 설정 의존**이며 `--log-file`로 우회 가능.

---

## 4. Hera 현재 표면 vs 엔진 표면 (갭 표)

Hera 현재 명령(요약): `status/operation/run/stop/scene/editor/node/script/project/signal/resource/theme/classdb/
game/instances/eval/guidance/game_feel/output/diagnostics/screenshot/batch/smoke/version`.

| 영역 | 엔진이 제공 | Hera 현재 | 갭의 성격 |
|---|---|---|---|
| 씬 실행 | `--scene <path\|UID>`, 자체 main loop | `run --scene` = **EditorInterface.play_*** (에디터 필요) | 노에디터 실행 없음 |
| 스크립트 실행 | `--script` (SceneTree 하니스) + `--` 인자 | `script validate`만 (엔진 스폰) | **실행** 계층 부재 |
| 검증 | `--check-only` | 있음(위임) | 미저장 버퍼 검증은 LSP로만 가능 |
| 임포트 | `--import` | `project scan/reimport`(에디터 소켓) | 노에디터 임포트 없음 |
| 익스포트 | `--export-pack/-release/-debug/-patch` | 없음 | **완전 부재** |
| 문서/지식 | `--doctool`, `--dump-extension-api*`, `--class-db-json` | `classdb`(라이브 에디터) | 오프라인 경로 없음 |
| 디버깅 | DAP, `--breakpoints`, `EditorDebuggerPlugin` | 없음("Debugger integration remains separate work") | **최대 공백** |
| 프로파일링 | `--profiling/--gpu-profile/--benchmark-file`, `profiler:*` 채널 | 없음(ROADMAP 후보) | 계측 계층 부재 |
| 캡처 | `--write-movie`, `scene:rq_screenshot` | `screenshot`, `game screenshot`(라이브) | 결정론적 프레임 캡처 없음 |
| 런타임 제어 | `scene:live_*`, `scene:next_frame` | 자체 autoload 인스펙터(프로젝트 autoload 주입, 익스포트 시 제거 가드 필요) | 프로토콜 재사용 여지 |
| UI/QA | 없음 | `game ui audit`, `game qa`, `game assert`, 입력 시퀀스, 증거/영수증 | **Hera 고유 자산** |
| 로그 | `--log-file`, 콘솔/파일 | `output`/`diagnostics`(파일 로깅 의존) | 노에디터 실행 시 로그 확보 개선 여지 |

**해석:** Hera의 자산은 "QA·증거·저토큰"이고, 엔진의 자산은 "실행·디버깅·빌드·지식". 겹치는 부분은
런타임 트리/제어뿐이며, 그마저 Hera가 *자체 프로토콜*을 쓴다. 승격 경로는 **엔진 표면을 얇게 포장하고
겹치는 부분은 엔진 프로토콜로 수렴**시키는 것이다.

---

## 5. 추가/보강/강화 후보 (우선순위별)

각 항목: 무엇을 / 왜 / 근거 / 리스크 / 대략 규모(CLI 명령 + 애드온 로직 기준).
명령 이름은 `docs/IDENTITY.md`의 규칙(엔진 명사 우선, 하위 명령 우선)에 맞춘 제안이며, 확정안이 아니다.

### Tier A — 노에디터 실행 계층 (즉시 가치, 낮은 리스크)

**A1. `hera-godot engine info|doctor`** — 엔진 바이너리 경로·버전·빌드종류(editor/template, mono/standard)·
SHA-256·`--help` 가용성(§2.1 범례)을 JSON으로 보고.
- 왜: 지금은 실패하면 "no live Godot editor found"로 끝난다. 에이전트는 *왜*인지 알 수 없다.
- 근거: `--version` **[실측]**, `--help` 범례 **[실측]**, DEV_MACHINE.md의 해시 기록 관행.
- 리스크: 낮음(읽기 전용). **규모**: 명령 1 + 파서 1.

**A2. `hera-godot script run <res://tool.gd> [-- args…] [--quit-after N] [--timeout ms] [--json]`** (초안 이름 `exec`)
- 왜: Blender의 `--python`에 해당하는 **"임의 GDScript 하니스 실행"**이 에이전트 게임 개발의 기본 연산이다
  (벌크 자산 변환, 콘텐츠 검증, 스크린샷 배치, 데이터 추출, 테스트 러너).
- 근거: d3/d5 **[실측]** (`SceneTree` 상속 + `--` 인자 전달 + exit code), `--quit-after`, `--log-file` **[문자열]**.
- 리스크: 중간(임의 코드 실행) → Hera 규약대로 **명시적 옵트인 + 출력 바운딩 + 증거 옵션** 필요.
  `--main-loop`는 쓰지 않는다(p2 **[실측 실패]**).
- **규모**: `cmd/exec.go` + 기존 `script_validate.go`의 엔진 스폰/타임아웃 재사용.

**A3. `hera-godot project import` (노에디터 임포트)** — `--import` 실행, exit code/소요시간/산출 요약
(`.godot/` 캐시, `.uid` 사이드카 생성 수), 로그는 **파싱하지 않고** 파일 크기/존재만 요약.
- 근거: g1/p9/프로브3 **[실측]**. 진행 로그가 **한국어로 로컬라이즈**되어 나온다는 사실도 **[실측]** → 텍스트 파싱 금지 근거.
- 리스크: 낮음(단, 프로젝트 디렉터리 쓰기 발생 → Hera의 "명시적 쓰기" 규약 필요).

**A4. `hera-godot project export pack|release|debug|patch` + 프리플라이트**
- 왜: "공식 CLI"라면 빌드 산출물 계층이 필수다. 특히 **pack은 템플릿 없이 성공**하므로 CI에서 즉시 쓸 수 있다.
- 근거: `--export-pack` exit 0 (pck 2,024B / zip 1,841B) **[실측]**; `--export-release` 실패 메시지가
  누락 템플릿 경로를 정확히 알려줌(`…/export_templates/4.7.2.stable.mono/windows_release_x86_64.exe`) **[실측]**;
  `export_presets.cfg` 부재 시 명확한 에러 **[실측]**; 템플릿 진단 문자열 다수 **[문자열]**.
- 보안 필수: `.godot/export_credentials.cfg`, `export_presets.cfg`의 키/비밀번호, `--p12-*`/`--ks-*` 플래그는
  **출력·로그·증거에 절대 포함 금지**(Hera `SECURITY.md` 규약 확장).
- **규모**: 명령 1 + 프리플라이트(템플릿/프리셋/경로 검증) + 응답 스키마.

**A5. `hera-godot capture movie` (결정론적 캡처)**
- 왜: `--fixed-fps` + `--write-movie`는 **프레임 인덱스가 고정된 산출물**을 만든다. 기존 `screenshot diff`와
  결합하면 "N프레임에서 시각 회귀"를 자동 검증할 수 있다. 엔진 CLI엔 어서션/증거/저토큰 요약이 없다 = Hera 고유 가치.
- 근거: f1/f2 **[실측]**, MovieWriter 3종 **[문자열]**, `--quit-after`로 프레임 수 고정 **[실측]**.
- 함정: `--headless`와 함께 쓰면 크래시(p8 **[실측]**) → 실제 디스플레이 드라이버 필요(§8).
- **규모**: 명령 1 + 프레임 파일 목록/해시 요약 + 기존 diff 재사용.

**A6. `hera-godot perf sample` (또는 Godot 용어로 `profiler …`)** — 성능 계측/회귀
- 왜: `--benchmark-file` JSON과 `profiler:*` 채널은 "성능 회귀"를 QA 항목으로 만든다(ROADMAP의
  "bounded performance samples" 후보와 일치).
- 근거: benchmark JSON **[실측]**, `--print-fps`/`--max-fps`/`--fixed-fps` **[문자열]**.
- **규모**: 명령 1 + 비교/임계값 로직(기존 `screenshot diff` 패턴 재사용).

### Tier B — 디버거/계측 통합 (최대 신규 능력)

**B1. `hera-godot debug dap` (GDScript DAP 세션)**
- 무엇: 세션 시작(에디터 `--dap-port` 지정) → `setBreakpoints` → 실행/계속 → `stopped` 이벤트 →
  `stackTrace`/`scopes`/`variables`/`evaluate` → `continue`/`next`/`stepIn`/`stepOut` → 종료. 전 과정 JSON 저토큰 출력.
- 왜: 에이전트가 "왜 이 값이 3인가"를 **실행 중 코드에서** 확인할 수 있다. Hera에 유일하게 빠진 능력.
- 실측 근거: `initialize`/`initialized`/`setBreakpoints`(응답에 **소스 MD5+SHA256 체크섬과 `verified: true`**)/
  `configurationDone`까지 창 모드에서 확인 **[프로브 7]**. `supportsSetVariable`/`supportsEvaluateForHovers` 등 능력은
  서버가 직접 광고 **[프로브 4]**. 게임 `launch`는 헤드리스·창 모드 모두 무응답 → **`launch` 미사용 설계** 권장:
  Hera의 기존 `run`(에디터 play)으로 게임을 띄우거나 `--debug-server` + 게임 `--remote-debug`로 세션을 만든 뒤 `attach`.
- **Hera 고유 강화 포인트**: DAP가 돌려주는 소스 체크섬을 기존 증거 모델에 그대로 붙일 수 있다 →
  "이 브레이크포인트는 SHA-256 `…`인 스크립트에서 멈췄다"를 증거로 남길 수 있다(엔진 도구에는 없는 조합).
- 안전: `supportsSetVariable`은 **실행 중 상태를 바꾸는 능력**이다 → Hera 규약대로 명시적 옵트인 + 조작 로그 필요
  (`eval`과 동일한 등급의 위험으로 취급).
- 근거(문자열/실측): §3.1, ROADMAP이 이미 후보로 명시("GDScript debugger commands over Godot's DAP").
- 리스크: 중~높음(에디터가 게임을 띄우는 경로, 버전별 DAP 차이). 다만 DAP는 표준 프로토콜이라 클라이언트 코드는 재사용 가능.
- **규모**: 전송 계층(Content-Length 프레이밍) + 요청/이벤트 매핑 + 세션 수명 관리.

**B2. 브레이크포인트 주입(에디터 없이)** — `--remote-debug tcp://host:port` + `-b res://x.gd::12`
- 왜: 디버그 익스포트 템플릿에서 실행되는 게임에 **비대화식**으로 중단점을 걸 수 있다(CI 재현).
- 근거: 도움말의 정확한 포맷 설명 + 플래그 문자열. **[미확인: 실제 세션 동작은 검증 필요]**

**B3. 런타임 인스펙터의 엔진 프로토콜 수렴(중기)**
- 무엇: 애드온에 `EditorDebuggerPlugin`을 등록해 `scene:*`/`profiler:*`/`multiplayer:*` 채널을 수신·송신.
  자체 autoload 인스펙터는 "디버거 미접속 환경"용 폴백으로 유지.
- 왜: 현재 구조는 **프로젝트에 autoload를 주입**하고, 익스포트 시 제거하는 가드(`game_inspector_export_guard.gd`,
  `EditorExportPlugin`)까지 필요하다. 엔진 프로토콜을 쓰면 **프로젝트 무수정**으로 실행 중 트리/프로퍼티/노드 생성/
  카메라 이동/스크린샷/프레임 스텝을 얻는다. "남의 프로젝트를 건드리지 않는다"는 공식 도구의 기본 예의이기도 하다.
- 근거: 채널 70여 종 **[문자열]**, `EditorDebuggerPlugin` API 이름 **[문자열]**, `EngineDebugger.*` **[문서]**,
  `scene:next_frame`(기존 `game clock --step` 대체 가능), `scene:rq_screenshot`, `scene:transform_camera_*`,
  `scene:window_request_size`.
- 리스크: 채널은 버전 간 변동 가능 → **기존 capability 협상 패턴**으로 게이트. 세션은 에디터 디버그 서버
  (`--debug-server`)에 게임이 접속하는 구조이므로, Hera가 자체 디버그 서버를 띄우는 변형도 설계 여지.
- **규모**: 큼(애드온 신규 계층 + CLI 브리지 + 회귀 스위트). 단계적으로: ①읽기(트리/프로퍼티) → ②제어(스텝/카메라) → ③스크린샷.

**B4. 프로파일러 지표 노출** — `profiler:visual|servers`, `servers:memory|draw|profile_frame`, `multiplayer:*`.
- 왜: 프레임 예산/메모리/복제 트래픽을 QA 시나리오의 요구사항으로 검증("프레임 60 이상 유지", "RPC 폭주 없음").

### Tier C — 코드 인텔리전스 (LSP)

**C1. `hera-godot script lsp diagnostics <res://file.gd>`**
- 왜: Hera 문서가 명시한 한계 — "script validate는 **미저장 편집 텍스트를 검사하지 않는다**".
  LSP `didOpen`/`didChange` 경로는 **디스크에 없는 버퍼**도 검사한다. 에이전트가 편집 중 즉시 검증 가능.
- 근거: LSP 메서드 문자열 전체 **[문자열]**, 리스너 **[실측]**, `willSaveWaitUntil`(저장 전 검사) 존재,
  최대 8클라이언트, "파일 프로토콜만 지원"/"Godot에 열린 프로젝트가 아닌 다른 프로젝트와는 정상 동작하지 않을 수 있음" **[문자열]**.
- 함정: LSP는 **포트당 1개 서버 + 8클라이언트** → 다중 에디터 환경에서는 Hera가 포트 할당/충돌 보고 책임.

**C2. `hera-godot script symbols|refs|hover|rename`** — `documentSymbol`/`references`/`definition`/`hover`/`rename`
및 **`nativeSymbol`**(엔진 심볼 문서) 노출. 저토큰 요약이 곧 Hera의 차별점.

**C3. `hera-godot docs project`** — `--doctool <dir> --gdscript-docs res://`로 **프로젝트 자체 API 문서**를 XML로 생성.
- 왜: 에이전트가 프로젝트를 이해하는 비용을 크게 낮춘다(파일 전체 읽기 → 요약 문서).
- 근거: `--doctool` 1,077 XML **[실측]**, `--gdscript-docs`는 프로젝트 상대경로 필요 **[실측: 절대경로는 파일명 망가짐]**.
  ※ 6라운드 프로브에서 `res://` 경로로 재검증(§7 미확인 항목 참조).

### Tier D — 결정론·재현 QA (Hera 고유 가치 극대화)

**D1. `hera-godot playtest run --file plan.json`**
- 조합: 노에디터/에디터 실행 + `--fixed-fps` + `game input sequence` + `game qa --file` + `game assert` +
  `capture movie` + `screenshot diff`.
- 왜: **"몇 프레임에서 무슨 입력이 들어갔고, 무엇이 참이어야 하며, 화면이 어떤가"**를 하나의 산출물로 묶는 것은
  엔진 CLI에 없는 능력이다. 재현성(고정 FPS)과 증거(해시/영수증)를 동시에 만족.
- 근거: 모든 구성 요소가 이미 Hera에 있거나 A5/A6으로 추가됨.

**D2. CI 레시피 문서/명령** — `.omo/plans/headless-ci-recipe.md`의 계획과 정확히 맞물린다.
  헤드리스 에디터 + 애드온 로드 + `--import` + 하니스 실행 + 아티팩트 업로드 절차를 `hera-godot ci init`류로 제공.

### Tier E — 거버넌스/안전

**E1. `hera-godot doctor`** — 엔진 경로/버전/빌드종류, 템플릿 설치 여부(경로 확인), 프로젝트 캐시 상태
(`.godot/uid_cache.bin`, `imported`, `shader_cache`), 애드온 활성 여부(`editor_plugins/enabled`), 로그 설정,
LSP/DAP 포트 점유, 디스크 여유(MovieWriter 경고와 동일 기준).
**E2. `hera-godot plugin enable|disable|list`** — `editor_plugins/enabled` + `--recovery-mode` 진단. 실패 원인
문자열(`Unable to load addon script…`, `Disabling the addon at '%s'…`)을 그대로 근거로 제시.
**E3. 쓰기 안전 규약 확장** — 노에디터 실행/익스포트에서 생성되는 파일(`.godot/**`, `*.uid`, pack, movie)을
"생성물"로 분류해 기본적으로 건드리지 않고, `--force`/명시 경로에서만 쓰는 규약.

### Tier F — 조건부/니치 (필요 시)

- **접근성 QA**: `--accessibility always|disabled`, `--accessibility-driver accesskit|dummy`, `AccessKit` 로더 문자열
  → 접근성 트리 기반 검증(별도 도구 없이 엔진 내장). **[문자열]**
- **i18n/로컬라이즈 QA**: `-l/--language`, `--editor-pseudolocalization`, `--text-driver`, `--tablet-driver`. **[문자열]**
- **렌더 진단**: `--gpu-validation`, `--gpu-abort`, `--gpu-profile`, `--accurate-breadcrumbs`,
  `--extra-gpu-memory-tracking`, `--generate-spirv-debug-info`(+RenderDoc 소스 디버깅). **[문자열]**
- **창 임베딩/리모트**: `--wid <window_id>`(0 불가 검증 메시지까지 확인), `--remote-fs`(콘솔/XR 배포). **[문자열]**
- **C# 확장**: `--build-solutions`, `--generate-mono-glue`, `--class-db-json` → C# 프로젝트 빌드/리로드 자동화.
  (Hera는 이미 C# 검사/생성 지원; 빌드는 "Hera가 하지 않는다"고 문서화되어 있음 → 자동화하면 이득)
- **Android/iOS**: `--install-android-build-template`, `--list-sdks`, 서명 관련 플래그 → 모바일 빌드 파이프라인.

---

## 6. "공식 Godot CLI 도구" 승격 조건 체크리스트

기술이 아니라 **신뢰 요건**이다. 각 항목 옆에 Hera의 현재 상태.

1. **엔진 표면 패리티(노에디터 경로)** — 미충족(A2~A4). 편집기 없이도 실행/검증/익스포트가 가능해야 한다.
2. **기계 판독 가능한 출력만 사용** — 엔진 텍스트는 **로컬라이즈**된다(본 조사에서 한국어 진행 로그 확인).
   → 파싱은 exit code/파일 산출물/`--benchmark-file`/`--log-file`로 제한. Hera는 이미 "파일 로그 vs 에디터 로그"
   구분을 문서화 → 이 원칙을 엔진 래핑에도 확장.
3. **빌드 종류 인지와 능력 협상** — R/D/X/E 범례를 명령 가용성 맵으로 노출(기존 `capabilities` 패턴 재사용).
4. **비침습 기본값** — 프로젝트 파일/autoload/설정을 조용히 바꾸지 않는다(B3와 E3). `--dry-run` 제공.
5. **버전 매트릭스** — 4.2~4.7 확인(기존 SUPPORT_MATRIX) + 4.7에서 새로 확인된 표면을 **기능 단위로 게이트**.
6. **시크릿 취급** — `export_credentials.cfg`, `*_presets.cfg`의 키, `--p12-*`/`--ks-*` 미출력 원칙.
7. **문서/배포 완결성** — ROADMAP Phase 9의 남은 항목(CONTRIBUTING, docs site, 공지)과 배포 채널 정렬.
8. **명명 규칙** — 엔진 명사와 충돌하지 않게(`scene/node/resource/…`), 새 기능도 엔진 용어를 우선(IDENTITY.md).

---

## 7. 하지 말아야 할 것 / 함정 (실측 근거)

1. **`--headless`에서 렌더/스크린샷 기대 금지** — dummy 렌더러: `get_viewport().get_texture().get_image()` → `null`,
   에러 `Parameter "t" is null. at: texture_2d_get (./servers/rendering/dummy/storage/texture_storage.h:110)`,
   headless viewport는 `64x64`. **실제 창(320x180)에서는 320x180 이미지 저장 성공** **[실측 p6/p7]**.
2. **`--headless` + `--write-movie` 크래시** — signal 11(접근 위반), 0바이트 .avi, 엔진 자체 크래시 핸들러 백트레이스
   (기호 없음: "no debug info in PE/COFF executable") **[실측 p8]**. 캡처는 실제 드라이버에서만.
3. **`--editor --script <EditorScript>` 불가** — `Can't load the script "res://…" as it doesn't inherit from SceneTree or MainLoop.`
   (main/main.cpp:4385) **[실측 p4]**. `--script` 없이 쓰면 `Class 'EditorScript' can only be instantiated by editor.` **[실측 p3]**.
   → 에디터 자동화는 **애드온 경로**를 유지하고, 헤드리스 에디터(+Hera 애드온)로 무인 실행하는 방향이 맞다.
   (문자열 `Cannot run the script because it doesn't extend EditorScript.` @ RVA 0x70ae2e8 은 **에디터 내부 실행** 경로의 것으로 보임)
4. **`--main-loop <class>`(GDScript `class_name`)** — 프로브에서 무응답/exit 1, 출력 없음 **[실측 p2/d4]**. 쓰지 말 것.
5. **`--gdscript-docs`는 프로젝트 상대 경로만** — 절대경로를 주면 `Can't write doc file: <dir>/C:\Users--…--main.gd.xml`로 파일명이 망가진다.
   **올바른 호출은 `--doctool <존재하는 디렉터리> [--no-docbase] --gdscript-docs res://`** 이며, 스크립트 1개당 XML 1개
   (`main.gd.xml`)가 생성되고 파일명은 스크립트 이름을 따른다 **[실측 6라운드]**.
6. **`--doctool`은 존재하는 디렉터리만 허용** — `Argument supplied to --doctool must be a valid directory path.` (main/main.cpp:4198) **[실측]**.
7. **`--export-*` 프리셋 이름은 공백을 포함**하므로 인용 필요 — 인용 실패 시 `Invalid export preset name: Windows.`
   + 사용 가능한 프리셋 목록을 출력 **[실측]**. Hera는 프리셋 이름을 **먼저 읽어 검증**해야 한다.
8. **진행/오류 텍스트의 현지화 + ANSI 색상** — 파싱 금지, 구조화 표면만 사용.
9. **DAP/LSP 포트는 서버당 1개, LSP 클라이언트 8개 제한** — 다중 인스턴스 환경에서 Hera가 포트를 관리·보고해야 한다.
10. **`.godot/`는 생성물** — 임포트/에디터 실행이 캐시(`uid_cache.bin`, `editor/filesystem_cache*`, `project_metadata.cfg`)를
    만들고 `.gd.uid` 사이드카가 생긴다 **[실측]**. Hera의 읽기/정리 규칙은 이 목록을 기준으로 삼아야 한다.
11. **DAP `setBreakpoints`에 `checksums`를 빼면 엔진이 자체 오류를 로그에 남긴다** (엔진 버그:
    `Dictionary::operator[] … key "checksums"`) **[실측]**. Hera의 "깨끗한 로그" QA 규약과 충돌하므로 클라이언트가 체크섬을
    함께 보내야 한다. DAP 중단점은 편집기 설정에 저장되며(`section "res://main.gd" key "state"`) 세션 간 잔존한다.
12. **DAP `launch`를 쓰지 말 것(현 관찰 기준)** — 헤드리스·창 모드 모두 무응답 **[실측]**. 게임 시작은 Hera의 기존
    경로로, 관찰은 DAP `attach`/중단점으로 분리하는 편이 안전하다.
13. **`..._console.exe`는 실제 엔진을 자식 프로세스로 띄운다** — 이번 조사에서 콘솔 래퍼 PID만 종료했더니 실제 엔진
    프로세스가 남아 계속 실행되었다(프로브 도중 발견, 수동 정리). → Hera의 노에디터 실행(`script run`, `movie`, `perf`)은
    **프로세스 트리 단위 종료(Windows Job Object / `taskkill /T` 상당)** 를 반드시 구현해야 한다. 반대로 엔진에는
    `--editor-pid <pid>`(§2.3)처럼 부모를 추적해 스스로 종료하는 경로도 있다.

---

## 8. 미확인 항목과 재현 방법

| 항목 | 상태 | 재현 방법 |
|---|---|---|
| 엔진 PE 의사코드/xref (Ghidra) | **미확인** — 330초 타임아웃 6회(181MB 자동분석) | REA `open_binary` 후 `binary_overview`/`search_strings`; 더 긴 provider 타임아웃, 또는 소스 기반 교차확인 |
| DAP 세션 실동작 | **실측 성공 (attach 경로)** — 2026-10-15: 에디터 `--debug-server tcp://127.0.0.1:6007 --dap-port 6066` + 게임 `--remote-debug tcp://127.0.0.1:6007` → DAP `attach` `success:true` → `setBreakpoints`(`verified:true`, MD5+SHA256) → `configurationDone` → `stopped`(reason `breakpoint`, `hitBreakpointIds`) → `stackTrace`(`_process`, 중단점 라인) → `scopes`(Locals/Members/Globals) → `variables`(`_d=0.0166…`) → `evaluate "counter"`=`703` → `continue` → 재중단 → `disconnect`. 증거: `.superpowers/sdd/hera-v13-engine-surface/evidence-dap-attach.log` | `launch`는 헤드리스·창 모드 모두 무응답이었으므로 attach + `--remote-debug`를 사용 |
| LSP 미저장 버퍼 진단 | **실측 성공**(오류 3건, severity 1) | `textDocument/didOpen`에 오류 있는 텍스트 → `publishDiagnostics` 확인 |
| `-b/--breakpoints` + `--remote-debug` 조합 | **미확인** | 디버그 템플릿 게임 + `--remote-debug tcp://127.0.0.1:6007` + `-b res://main.gd::8` |
| `--scene uid://…` | **실측 성공**(직접 선언한 uid, `--import` 후) | 에디터에서 저장한 씬(uid 속성 보유) + `--import` 후 `--scene uid://…` |
| `--editor-pid` 의미 | **미확인**(파싱/오류 메시지만 확인) | 에디터를 부모 프로세스와 함께 띄우고 부모 종료 시 동작 관찰 |
| 두 에디터 인스턴스의 LSP/DAP 포트 충돌 동작 | **미확인**(오류 메시지 없음) | 동일 포트로 2개 실행 후 `--- Failed to start … on port` 문자열 확인 |
| `--gdscript-docs res://` | **실측 성공**(XML 1개, 882B) | `--headless --doctool <dir> --no-docbase --gdscript-docs res://` |
| Movie Maker 포맷/오디오 세부(코덱 품질, 오디오 믹스) | **미확인** | `--write-movie out.ogv`, `.avi` 비교 + `MovieWriter's audio mix rate` 경고 확인 |

---

## 9. 제안 로드맵 (Hera 관점)

- **v1.3 — "엔진 CLI 패리티(노에디터)"**: A1 `engine info|doctor`, A2 `script run`, A3 `project import`,
  A5 `capture movie`, A6 `perf sample`, A4의 `project export pack` + 프리플라이트. 전부 **읽기/명시적 쓰기** 규약 유지, §7 함정 준수.
- **v1.4 — "디버깅·지능"**: B1 DAP 세션, B2 브레이크포인트 주입, C1/C2 LSP 진단/심볼, C3 프로젝트 문서.
- **v1.5 — "엔진 프로토콜 수렴"**: B3 런타임 인스펙터를 `EditorDebuggerPlugin`+`scene:*`로 이중화(프로젝트 무수정 모드),
  B4 프로파일러/네트워크 채널, D1 `playtest run` 조합 명령.
- **상시**: E1 doctor, E2 plugin 관리, E3 쓰기 안전 규약, §6 승격 체크리스트 항목(문서/배포/시크릿).

---

## 부록 A — 프로브 원자료(요약, 격리 환경)

스크립트: `%TEMP%\opencode\{godot_probe.ps1,godot_probe2.ps1,godot_probe3.ps1,godot_probe5.ps1,godot_probe6.ps1,godot_probe4.py,godot_probe7.py}`,
로그: 같은 경로의 `godot-re-472*/probe*.log`, 문자열 인벤토리 `godot472-strings.txt`, PE 리터럴 로케이터 `pe_locate.py`.

핵심 성공(exit 0): `--class-db-json`(2 파일), `--doctool`(1,077 XML), `--dump-extension-api-with-docs`(11.87MB),
`--dump-gdextension-interface-json`(328KB), `--import`, `--scene <path>`, `--script`(+`--` 인자), `--check-only`,
`--write-movie`(.avi/.png), `--benchmark-file`, `--export-pack`(pck/zip), `--help`, `--version`,
`--headless --editor --quit-after N`, `--lsp-port`/`--dap-port` 리스너.
핵심 실패/한계: `--main-loop`(GDScript), `--editor --script`(EditorScript), `--headless` 렌더/캡처,
`--headless + --write-movie` 크래시, `--export-release`(템플릿 없음), `--gdscript-docs` 절대경로.

6라운드 추가 실측: `--doctool <dir> --no-docbase --gdscript-docs res://` → `main.gd.xml` 882B(성공),
`--scene uid://heraprobe6scene` → 씬 실행 성공(exit 0), `--export-pack` 재확인(pck 2,024B / zip 1,841B).

4/7라운드 DAP·LSP 실측: LSP 접속 → `gdscript_client/changeWorkspace` → 미저장 버퍼 진단 3건(severity 1),
`documentSymbol` 구조 반환. DAP 접속 → `initialize`(능력 광고: `supportsSetVariable`, `supportsEvaluateForHovers`,
`supportsConfigurationDoneRequest`, `supportsBreakpointLocationsRequest`, `supportsRestartRequest`,
`supportsTerminateRequest`, 체크섬 MD5/SHA256) → `initialized` → `setBreakpoints`(`verified: true` + 체크섬) →
`configurationDone` 성공, `launch`는 헤드리스 20s·창 45s 모두 무응답. 에디터 로그에
`Bug: Dictionary::operator[] … key "checksums"`(엔진 버그)와 `section "res://main.gd" key "state"` 노이즈 관찰.

## 부록 B — 이번 분석에서 확인된 Hera 측 사실

- `addons/hera_agent_godot/tools/run_tool.gd`는 `EditorInterface.play_main_scene|play_current_scene|play_custom_scene|stop_playing_scene` 사용 → **실행이 에디터 의존**.
- `addons/hera_agent_godot/tools/output_tool.gd`는 `user://logs/godot.log` + `debug/file_logging/enable_file_logging` 의존.
- 런타임 인스펙터는 프로젝트 autoload 주입 + `runtime/game_inspector_export_guard.gd`(`EditorExportPlugin`)로
  익스포트 시 제거 → **프로젝트/익스포트 무수정이 아님**(B3의 근거).
- 엔진 스폰은 `cmd/script_validate.go` 한 곳뿐(`--headless --path … --check-only --script …`).
- ROADMAP은 이미 후보로 기록: "InputMap inspection/binding, bounded performance samples, GDScript debugger commands over Godot's DAP".
