# v2.0.0 소개 영상 촬영 대본

이 문서는 `PROMPT.md`로 Hera v2.0.0 소개 영상을 찍는 순서다. 위에서 아래로
그대로 따라 하면 된다. 결정 배경은 2026-10-16 그릴링(자율 시연 + 30초 수동
인트로, 순수 C#, Game Feel ON 1회, 프롬프트 고정)이다.

- 에이전트: Codex, `gpt-6.1-sol` medium (Codex 기본 설정에 이미 저장됨)
- 엔진: Godot 4.7.2 .NET, 이 폴더(`demo/td-csharp`) 프로젝트
- CLI: **공개 릴리스 2.0.0** (개발 shim 아님)
- 녹화: ScreenToGif, 1920×1080, MP4로 내보내기

## 촬영 1번이 곧 최종 리허설이다

Codex 리허설(3분 40초, 4개 기준 통과)은 레포를 이미 아는 Codex가 격리 프로필에서
돌린 것이다. 실제 촬영은 **아무것도 모르는 새 Codex 세션**이고, 인트로에서
에이전트 설정을 먼저 붙여넣는다. 그래서 첫 촬영을 리허설 겸 본편으로 찍는다.
잘 나오면 그대로 쓰고, 아니면 리셋하고 다시 찍는다.

## 1. 녹화 30분 전 준비

### 1-1. 다른 Godot 창을 모두 닫는다

이 레포 루트 프로젝트를 연 에디터를 포함해 Godot 창을 전부 닫는다. 에디터가
하나만 떠 있어야 Codex가 `--instance` 없이 바로 명령을 쓸 수 있다. PowerShell에서
아래가 `[]`나 빈 목록이면 된다.

```powershell
$hera = Join-Path $env:LOCALAPPDATA 'hera-godot-release\node_modules\.bin\hera-godot.cmd'
& $hera instances
```

### 1-2. 데모 폴더를 빈 베이스로 되돌린다

레포 루트에서 실행한다. 지난 촬영의 게임 코드와 씬이 전부 지워진다. 남기고 싶은
결과가 있으면 먼저 다른 곳에 복사한다.

```powershell
cd C:\Users\PC\Desktop\Cowork\hera-agent-godot
git restore --worktree -- demo/td-csharp/Main.tscn demo/td-csharp/project.godot
git clean -fdx -- demo/td-csharp/
```

### 1-3. 준비 스크립트를 돌린다

애드온 복사, .NET 빌드, 컴파일러 헬퍼 설치를 한 번에 한다. 마지막 줄에
`"installed":true`가 나오면 성공이다.

```powershell
.\demo\td-csharp\prepare.ps1 -HeraCli $hera
```

### 1-4. Godot으로 프로젝트를 연다

1. `Godot_v4.7.2-stable_mono_win64.exe`를 실행한다.
2. 프로젝트 매니저에서 **가져오기(Import)** → `demo\td-csharp\project.godot` → **열기**.
3. 처음 열 때 .NET 빌드와 에셋 가져오기가 끝날 때까지 기다린다. 아래 출력 패널이
   조용해지면 된다.
4. 위쪽 가운데 **HeraAgent** 탭을 누른다. 연결 줄이 초록색
   "Listening 127.0.0.1:8770"이면 준비 끝이다. 버전 줄은 아직
   "아직 CLI 요청 없음"이어야 한다. 이게 인트로에서 바뀌는 장면이다.
5. 씬 탭에 `Main.tscn`이 비어 있는 상태로 열려 있는지 본다.

### 1-5. 녹화용 터미널을 연다

herdr가 아니라 **새 Windows Terminal 창 하나**를 쓴다. 화면에 다른 pane이 안 나오게
하려는 것이다. 설정에서 글꼴 크기를 16으로 올린다.

```powershell
$env:PATH = (Join-Path $env:LOCALAPPDATA 'hera-godot-release\node_modules\.bin') + ';' + $env:PATH
cd C:\Users\PC\Desktop\Cowork\hera-agent-godot\demo\td-csharp
hera-godot --version          # v2.0.0 이어야 한다
Clear-Host
codex -c features.hooks=false -c features.plugin_hooks=false -c features.plugins=false
```

- 마지막 줄은 내 개인 플러그인과 훅(ponytail, herdr 상태 훅 등)을 끈다. 시청자가 쓰는
  기본 Codex와 같은 조건이고, 화면에 "Hook failed" 같은 빨간 줄도 안 나온다.
- Codex 화면 아래 상태줄에 `GPT-6.1-Sol medium`이 보여야 한다.
- 이 폴더에서 시작해도 레포 루트 `AGENTS.md`가 자동으로 읽힌다(2026-10-16 확인).

### 1-6. 창 배치

주 모니터(1920×1200)에서 위쪽 1920×1080만 녹화한다. 작업표시줄은 녹화 영역 밖이다.

| 창 | 위치 | 크기 |
|---|---|---|
| Godot 에디터 | 왼쪽 위 (0, 0) | 가로 1200, 세로 1080 |
| Windows Terminal (Codex) | (1200, 0) | 가로 720, 세로 1080 |

알림(Discord, 메신저 등)은 끈다. Windows 집중 모드를 켜면 편하다.

### 1-7. ScreenToGif 설정

1. `C:\Program Files\ScreenToGif\ScreenToGif.exe` 실행 → **옵션 → 추가 기능(Extras)**에서
   FFmpeg 경로를 지정한다.
   `C:\Users\PC\AppData\Local\Microsoft\WinGet\Packages\Gyan.FFmpeg_Microsoft.Winget.Source_8wekyb3d8bbwe\ffmpeg-8.1.1-full_build\bin\ffmpeg.exe`
2. **녹화기(Recorder)**를 연다. 프레임 속도 **15fps**, 녹화 영역을 (0,0)에서 1920×1080에 맞춘다.
3. 단축키는 기본값 **F7 녹화/일시정지, F8 정지**를 쓴다.
4. 5분 녹화면 프레임 캐시가 수 GB가 된다. C 드라이브 여유 공간이 10GB 이상인지 본다.
5. 예비 수단: ScreenToGif가 버벅이면 Windows 캡처 도구(**Win+Shift+R**)로 같은 영역을
   녹화하면 MP4가 바로 나온다.

## 2. 촬영

### 2-1. 인트로 (사람 손, 약 30초)

| 시간 | 할 일 | 화면에서 보여야 하는 것 |
|---|---|---|
| 0:00 | **F7**로 녹화 시작. 2초간 가만히 | HeraAgent 패널 전체, 오른쪽 Codex 빈 화면 |
| 0:03 | 마우스로 패널을 천천히 훑는다 | 한영 문구, 상태 4줄 |
| 0:08 | **Game Feel Mode**, **Game Feel UI Mode** 토글을 켠다 (이미 켜져 있으면 가리키기만) | 토글 2개 켜짐 |
| 0:13 | **에이전트 설정 복사 Copy agent setup** 클릭 | 버튼 옆 "복사됨 Copied" |
| 0:16 | Codex 창 클릭 → **Ctrl+V** → **Enter** | 설정 텍스트가 Codex에 들어감 |
| 0:18~ | 손을 뗀다. Codex가 `hera-godot status`, `smoke --skip-game`을 실행 | 패널 버전 줄이 **"버전 일치 · CLI v2.0.0 · 애드온 2.0.0"**으로 바뀜 |

Codex가 확인 결과를 말하고 입력창이 다시 뜰 때까지 기다린다.

### 2-2. 프롬프트 전송 (본편 시작)

1. `PROMPT.md`를 메모장으로 열어 두고, ```text 상자 **안의 문단만** 복사한다.
   앞뒤 설명이나 ``` 줄은 넣지 않는다. 글자를 바꾸지 않는다.
2. Codex 창에 **Ctrl+V → Enter**.
3. 이 순간부터 손을 뗀다. 마우스는 녹화 영역 밖(아래 작업표시줄 쪽)으로 치운다.

### 2-3. 자율 구간 (Codex 혼자, 리허설 기준 약 4분)

지켜보기만 한다. 리허설에서 나온 흐름은 이렇다. 순서는 매번 조금 다를 수 있다.

1. AGENTS.md와 게임필 가이드를 읽는다.
2. C# 게임 스크립트를 쓰고 `script validate res://...cs`로 디스크 검증, 빌드.
3. 씬에 붙이고 저장, `run --current`로 게임 실행.
4. 실제 마우스 입력으로 타워 배치와 웨이브 시작, 스크린샷 확인.
5. 문제를 고치고 다시 검증·빌드. **재빌드 직후 C# eval이 한 번 `compile_failed`를 내는 건
   정상이다.** 에디터가 새 DLL을 자동 리로드하면 다음 eval은 성공한다.
6. QA 시나리오 실행, 요약 출력.

Codex가 질문을 하면 그대로 답한다: `Proceed with your best judgment.`

### 2-4. 결과 보여주기 (사람 손, 약 30초)

1. Codex 요약이 끝나면 5초 그대로 둔다.
2. 에디터에서 **F5**(또는 오른쪽 위 재생 버튼)로 게임을 직접 실행한다.
3. 타워 2~3개를 클릭으로 배치하고 웨이브를 시작한다. 적이 맞고 쓰러지는 장면을
   10~15초 보여준다.
4. 게임을 끄고 **F8**로 녹화를 멈춘다.

### 2-5. 이럴 땐 다시 찍는다

| 상황 | 할 일 |
|---|---|
| `csharp_unload_failed`가 나오거나 에디터가 멈춤 | 녹화 정지 → 1-1부터 다시. 로그를 Claude에게 전달 |
| Codex가 C# 명령(`eval`, `script validate`)을 하나도 안 씀 | 이번 테이크는 버리고 Claude에게 알린다. 프롬프트가 아니라 AGENTS.md를 고친다 |
| 10분이 넘어감 | 정지 → 리셋 → 재촬영 |
| 알림 팝업, 다른 창이 끼어듦 | 편집에서 잘라낼 수 있으면 계속, 아니면 재촬영 |

## 3. 녹화 끝난 뒤

### 3-1. MP4로 저장

ScreenToGif 편집기가 열리면 **파일 → 다른 이름으로 저장 → 형식: MP4(FFmpeg)**.
품질은 기본값, 파일 이름은 `hera-v2.0.0-td-csharp-take1.mp4`처럼 테이크 번호를 붙인다.

### 3-2. README용 GIF 2개 (같은 녹화에서 잘라내기)

같은 편집기에서 프레임 범위를 골라 GIF로 따로 저장한다. 가로 960으로 줄여 10MB 아래로 맞춘다.

1. **인트로 컷 (약 10초)**: 토글 켜기 → 설정 복사 → 버전 줄이 "버전 일치"로 바뀌는 순간까지.
2. **전투 컷 (약 8초)**: 타워가 쏘고 적이 쓰러지고 골드가 오르는 장면.

### 3-3. 데모 폴더 리셋

Godot 에디터를 닫고 1-2의 명령을 다시 실행한다. 만들어진 게임을 남기고 싶으면
리셋 전에 `demo\td-csharp\*.cs`와 씬 파일을 다른 폴더로 복사한다.

## 4. 업로드 초안

**제목**

```text
GPT-6.1-Sol builds a C# tower defense in a live Godot editor | Hera Agent Godot v2.0.0
```

**설명 첫 문단**

```text
Codex (GPT-6.1-Sol, medium) uses hera-agent-godot v2.0.0 to build a playable mini tower defense game in C# inside a live Godot 4.7 .NET editor from a single prompt. The agent writes C#, validates it against the real project build, evaluates C# expressions in the running editor, plays the game with real input, checks screenshots, and fixes what it finds. Game Feel Mode is ON. Same task as our GPT-5.5 / v0.7 run, now on Hera v2.0.0.
```

설명 아래쪽 링크와 테스트 셋업 목록은 지난 영상 형식을 따르되 Agent·Model·Reasoning·Engine·
Tooling 줄을 이번 값으로 바꾼다. "single-run workflow demo, not a controlled benchmark" 문장은 유지한다.
