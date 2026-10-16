# npm 2.0.2 배포 가이드 (Victor 실행)

GitHub 릴리스와 npm은 별도입니다. 이 작업에서는 **pack만** 수행하며 npm 배포는 Victor가 인증 후 실행합니다. 2.0.1을 먼저 배포할 필요 없이 2.0.2를 배포하세요.

## 1. 준비 및 로그인

PowerShell에서 다음을 실행하세요.

```powershell
Set-Location C:\Users\PC\Desktop\Cowork\hera-agent-godot
npm whoami --registry=https://registry.npmjs.org/
```

본인의 npm 계정이 나오면 다음으로 진행합니다. 인증 오류라면 `npm login --registry=https://registry.npmjs.org/`을 실행하고 브라우저에서 로그인·2FA를 마친 후 `npm whoami`를 다시 확인하세요. OTP나 인증 토큰을 채팅에 보내지 마세요.

## 2. 배포 파일 확인

```powershell
npm view hera-godot@2.0.2 version --registry=https://registry.npmjs.org/
npm publish ./packaging/npm/hera-godot-2.0.2.tgz --dry-run --access public --registry=https://registry.npmjs.org/
```

첫 명령에서 버전이 없다는 E404가 나오면 아직 미배포 상태입니다. 이미 2.0.2가 나온다면 아래 배포를 반복하지 말고 게시된 `dist.integrity`를 릴리스 기록과 비교하세요. 두 번째 명령은 배포하지 않고 `hera-godot@2.0.2`와 파일 목록을 보여줍니다. 반드시 준비된 `.tgz`를 지정하세요.

## 3. 실제 배포 — 이 한 번만 직접 실행

```powershell
npm publish ./packaging/npm/hera-godot-2.0.2.tgz --access public --tag latest --registry=https://registry.npmjs.org/
```

npm이 요청하면 인증 앱의 현재 OTP 또는 브라우저 보안 키 인증으로 완료하세요. OTP를 명시해야 하는 환경에서는 위 명령 끝에 `--otp=현재코드`를 붙여 실행합니다. 코드 대신 예시 문자열을 그대로 입력하지 마세요.

`+ hera-godot@2.0.2`가 성공 표시입니다. 네트워크 오류로 결과가 불명확하면 아래 조회로 게시 여부를 먼저 확인하세요. 같은 버전은 재게시할 수 없습니다.

## 4. 게시 검증

```powershell
npm view hera-godot@2.0.2 version dist.integrity --registry=https://registry.npmjs.org/
npm view hera-godot dist-tags --json --registry=https://registry.npmjs.org/
```

버전 2.0.2, 릴리스 기록과 같은 integrity, `latest: 2.0.2`를 확인하세요. 성공 출력만 Codex에게 알려주면 녹화용 별도 npm prefix 설치와 실제 CLI 검증을 이어갑니다. 기존 전역 `hera`를 건드릴 필요는 없습니다.

참고: [npm publish 공식 문서](https://docs.npmjs.com/cli/commands/npm-publish/), [npm 2FA 공식 안내](https://docs.npmjs.com/accessing-npm-using-2fa).
