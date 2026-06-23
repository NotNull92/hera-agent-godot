# hera-agent-godot

[English](README.md) · **한국어**

> Let's go Hera, now in Godot.

AI 코딩 에이전트가 **실행 중인 Godot 4.7+ 에디터**를 실시간으로 검사·제어하게 해주는
**저토큰(low-token) CLI**입니다 — 출력/에러 읽기, 씬 실행, 노드 트리 탐색·편집,
GDScript 평가 등. 에이전트가 낡은 학습 데이터로 추측하는 대신 *실제* 에디터에
직접 작용하고 결과를 확인합니다.

[`hera-agent-unity`](https://github.com/NotNull92/hera-agent-unity)의 자매
프로젝트로, 동일한 저토큰·쉘 친화 철학을 따르지만 **포팅이 아니라 Godot에 맞춰
새로 설계**했습니다. (Godot 공식 MCP 서버는 존재하지 않으며, 이 프로젝트는 의도적으로
CLI 방식입니다.)

## 현재 상태

🚧 **Phase 0 — 뼈대.** 아키텍처와 디렉토리 구조가 자리잡았고, 구현은
[docs/ROADMAP.md](docs/ROADMAP.md)에 따라 단계적으로 진행됩니다.

## 동작 방식

```
Go CLI  ──HTTP /rpc──▶  Godot 에디터 애드온 (@tool EditorPlugin, GDScript)
 (cmd/, internal/)        (godot/addons/hera_agent_godot/)
        ▲                          │
        └── ~/.hera-agent-godot/instances/ 스캔 ◀── Heartbeat
```

- **CLI** (Go): 에디터를 탐색하고, 명령마다 압축된 JSON 요청 하나를 보냅니다.
- **애드온** (GDScript): localhost HTTP 서버를 띄우고, 각 요청을 `EditorInterface`를 통해
  에디터 메인 스레드에서 실행합니다.

전체 설계는 **[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)**, 명령 목록은
**[docs/COMMANDS.md](docs/COMMANDS.md)**, 구현 계획은
**[docs/ROADMAP.md](docs/ROADMAP.md)**를 참고하세요.

## Unity판과 의도적으로 다른 점 (Godot 최적화)

Godot은 에디터 확장 모델 자체가 달라 그대로 옮기지 않았습니다. (자세한 내용은
[ARCHITECTURE.md §2](docs/ARCHITECTURE.md))

1. **표준 애드온 배포** — GDScript 애드온은 `res://addons/hera_agent_godot/` 폴더만
   복사하면 되고, .NET SDK나 C# 프로젝트 생성이 필요 없습니다.
2. **도메인 리로드 머신 없음** — `EditorPlugin._EnterTree/_ExitTree` 생명주기만 사용.
3. **메인 스레드 펌프** — HTTP 리스너는 워커 스레드, 실제 에디터 조작은 `_Process`에서
   큐를 비우며 실행 (Godot는 오프스레드 에디터 접근 시 크래시).
4. **Godot 어휘** — GameObject/Component/Prefab 대신 **Node/Scene/Resource/Signal**.
5. **eval은 GDScript 기반** — Godot의 `Expression`과 `@tool` 스크립트 흐름을 사용.
6. **명시적 툴 레지스트리** — 리플렉션 스캔 대신 명시 등록.

## 디렉토리 구조

```
cmd/         Go CLI 명령 (status, run, scene, node, eval, output)
internal/    client / discovery / protocol
godot/       개발용 Godot 4.7+ 프로젝트 + 애드온 (godot/addons/hera_agent_godot)
docs/        ARCHITECTURE, COMMANDS, ROADMAP
```

## 요구 사항 (목표)

- Go 1.25+ (CLI)
- Godot **4.7+** 표준 빌드 (애드온)

## 라이선스

MIT — [LICENSE](LICENSE) 참고.
