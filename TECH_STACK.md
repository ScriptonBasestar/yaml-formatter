# 🛠️ YAML Formatter – TECH_STACK.md

> **핵심 목표**  
> 빠르고, 안정적이며, 단독 실행 가능한 YAML 포매터를 구현하기 위한 최적의 기술 스택

---

## 📋 요구사항 분석

**DEFINITION.md 기반 핵심 요구사항:**
- ✅ **단독 실행** 가능한 바이너리
- ✅ **빠른 성능** (대용량 파일 처리)
- ✅ **CI/Git 훅** 통합 용이성
- ✅ **스키마 기반** YAML 키 순서 정렬
- ✅ **다양한 YAML 타입** 지원 (Docker Compose, K8s, GitHub Actions, Ansible, Helm)

**testdata 기반 복잡도 분석:**
- 🔹 **중첩 구조** 처리 (서비스 > 컨테이너 > 환경변수)
- 🔹 **배열 내 객체** 정렬 (ports, volumes, env)
- 🔹 **조건부 정렬** (non_sort 섹션)
- 🔹 **멀티 도큐먼트** YAML 지원
- 🔹 **주석 보존** 필요

---

## 🚀 최적 기술 스택

### 1. **프로그래밍 언어: Go** 🐹

**선택 이유:**
- ⚡ **컴파일된 단일 바이너리** → 의존성 없이 배포
- 🔥 **빠른 성능** → 대용량 YAML 파일 처리에 최적
- 📦 **뛰어난 YAML 라이브러리** 생태계
- 🔧 **CLI 도구 개발**에 특화된 에코시스템
- 🐧 **크로스 플랫폼** 빌드 지원

**대안 비교:**
| 언어 | 장점 | 단점 | 결론 |
|------|------|------|------|
| **Go** | 단일바이너리, 빠름, CLI특화 | 러닝커브 | ✅ **최적** |
| Rust | 매우 빠름, 메모리 안전 | 복잡한 문법, 개발속도 | ❌ 과도한 복잡성 |
| Python | 쉬운 개발, 풍부한 라이브러리 | 느림, 의존성 관리 | ❌ 성능 이슈 |
| Node.js | 빠른 개발, JSON/YAML 처리 | 런타임 필요, 메모리 사용량 | ❌ 배포 복잡성 |

### 2. **YAML 처리: gopkg.in/yaml.v3** 📝

**선택 이유:**
- 🎯 **주석 보존** 지원
- 🔄 **키 순서 유지** 기능
- 📊 **AST 접근** 가능 (세밀한 제어)
- 🏗️ **커스텀 마샬링** 지원

```go
// 핵심 기능 예시
type Node struct {
    Kind    Kind
    Tag     string
    Value   string
    Content []*Node
    // 주석과 위치 정보 포함
}
```

**대안 비교:**
- `gopkg.in/yaml.v2`: 주석 보존 안됨
- `sigs.k8s.io/yaml`: JSON 호환 중심, 기능 제한적

### 3. **CLI 프레임워크: Cobra + Viper** 🐍

**Cobra (CLI 구조):**
```go
sb-yaml/
├── cmd/
│   ├── root.go      // 메인 명령어
│   ├── schema.go    // schema 서브명령어
│   └── format.go    // format 서브명령어
└── main.go
```

**Viper (설정 관리):**
- 📁 **설정 파일** 자동 탐지 (`~/.sb-yaml/config.yaml`)
- 🌍 **환경 변수** 지원 (`SB_YAML_SCHEMA_DIR`)
- ⚙️ **플래그 바인딩** 자동화
2
**대안:**
- `urfave/cli`: 단순하지만 기능 제한적
- `spf13/pflag`: 저수준, 보일러플레이트 많음

### 4. **파일 시스템 처리: afero + filepath** 📂

**afero 선택 이유:**
- 🧪 **테스트 가능성** (메모리 파일시스템)
- 🔒 **추상화된 파일 접근**
- 🌐 **다양한 백엔드** 지원

```go
// 실제 파일시스템
fs := afero.NewOsFs()

// 테스트용 메모리 파일시스템
fs := afero.NewMemMapFs()
```

### 5. **패턴 매칭: doublestar** 🌟

**globbing 패턴 지원:**
```bash
sb-yaml format compose *.yml
sb-yaml format k8s **/*.k8s.yaml
```

### 6. **테스트 프레임워크: testify + ginkgo** 🧪

**testify (단위 테스트):**
```go
assert.Equal(t, expected, actual)
require.NoError(t, err)
```

**ginkgo (BDD 통합 테스트):**
```go
Describe("YAML Formatting", func() {
    Context("Docker Compose files", func() {
        It("should reorder keys according to schema", func() {
            // 테스트 구현
        })
    })
})
```

### 7. **빌드 도구: Goreleaser + GitHub Actions** 🚀

**goreleaser.yml:**
```yaml
builds:
  - env: [CGO_ENABLED=0]
    goos: [linux, windows, darwin]
    goarch: [amd64, arm64]
    binary: sb-yaml

release:
  github:
    owner: company
    name: sb-yaml
```

---

## 🏗️ 아키텍처 설계

### 핵심 컴포넌트

```
sb-yaml/
├── cmd/                    # CLI 명령어
├── internal/
│   ├── schema/            # 스키마 관리
│   │   ├── loader.go      # 스키마 로드/저장
│   │   ├── generator.go   # YAML에서 스키마 생성
│   │   └── validator.go   # 스키마 검증
│   ├── formatter/         # YAML 포매팅
│   │   ├── parser.go      # YAML 파싱
│   │   ├── reorder.go     # 키 순서 재배열
│   │   └── writer.go      # 포매팅된 YAML 출력
│   ├── config/            # 설정 관리
│   └── utils/             # 공통 유틸리티
├── pkg/                   # 공개 API
├── testdata/              # 테스트 데이터
└── docs/                  # 문서
```

### 데이터 플로우

```
YAML 파일 → Parser → AST → Reorder → Writer → 포매팅된 YAML
    ↑                         ↑
스키마 파일 ←→ Schema Loader ←→ 키 순서 규칙
```

---

## 🔧 개발 도구

### 필수 도구
```bash
# Go 도구체인
go 1.21+
golangci-lint    # 정적 분석
gofumpt         # 코드 포매팅
govulncheck     # 취약점 검사
```

### CI/CD 파이프라인
```yaml
# .github/workflows/ci.yml
- Test (Go 1.20, 1.21)
- Lint (golangci-lint)
- Security (govulncheck)
- Build (cross-platform)
- Release (goreleaser)
```

---

## 📊 성능 목표

| 지표 | 목표 | 근거 |
|------|------|------|
| **파일 크기** | < 1MB YAML | CI 환경 일반적 크기 |
| **처리 속도** | < 100ms | 사용자 체감 지연 없음 |
| **메모리 사용** | < 50MB | CI 환경 제약 |
| **바이너리 크기** | < 10MB | 다운로드 부담 최소화 |

---

## 🛡️ 보안 고려사항

### 1. **입력 검증**
- YAML 폭탄 공격 방지 (파일 크기/깊이 제한)
- 악성 스키마 파일 차단

### 2. **파일 시스템 보안**
- 경로 탐색 공격 방지 (`filepath.Clean`)
- 권한 검사 (읽기 전용 파일 보호)

### 3. **의존성 관리**
- `go.sum` 체크섬 검증
- Dependabot 자동 업데이트

---

## 🎯 결론

**Go + yaml.v3 + Cobra** 조합이 이 프로젝트의 요구사항에 가장 적합합니다:

✅ **단일 바이너리** 배포로 의존성 없음  
✅ **빠른 성능**으로 CI 환경에 최적  
✅ **풍부한 CLI 생태계**로 개발 생산성 향상  
✅ **강력한 YAML 처리** 기능으로 복잡한 정렬 로직 구현 가능  
✅ **크로스 플랫폼** 지원으로 다양한 환경 대응  

이 기술 스택으로 안정적이고 성능이 뛰어난 YAML 포매터를 구현할 수 있습니다.