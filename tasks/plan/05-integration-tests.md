# 통합 테스트 수정 계획

## 목표
CLI 명령어와 End-to-End 시나리오에 대한 안정적인 통합 테스트 구축

## 현재 문제점 분석

### 1. CLI 테스트 실패
```
cmd/format_test.go: 바이너리 빌드 의존성
cmd/schema_test.go: 임시 디렉터리 설정 문제
cmd/root_test.go: 명령어 실행 환경 불일치
```

### 2. E2E 테스트 불안정성
```
tests/e2e/e2e_test.go: 바이너리 존재 전제
파일 시스템 의존성으로 인한 격리 부족
환경변수 설정 충돌
```

### 3. 테스트 환경 격리 부족
```
테스트 간 상태 공유 문제
임시 파일 정리 미흡
병렬 실행 시 충돌
```

## 수정 작업 목록

### Task 5.1: CLI 테스트 모듈화 (4시간)

#### 5.1.1 CLI 테스트 인프라 구축
**파일**: `cmd/testing_utils.go`

**공통 테스트 유틸리티**:
```go
package cmd

import (
    "bytes"
    "io"
    "os"
    "path/filepath"
    "testing"
    "github.com/spf13/cobra"
)

type CLITestHarness struct {
    tempDir    string
    schemaDir  string
    stdout     *bytes.Buffer
    stderr     *bytes.Buffer
    originalEnv map[string]string
}

func NewCLITestHarness(t *testing.T) *CLITestHarness {
    tempDir := t.TempDir()
    schemaDir := filepath.Join(tempDir, "schemas")
    
    harness := &CLITestHarness{
        tempDir:     tempDir,
        schemaDir:   schemaDir,
        stdout:      &bytes.Buffer{},
        stderr:      &bytes.Buffer{},
        originalEnv: make(map[string]string),
    }
    
    // 환경변수 백업 및 설정
    harness.backupAndSetEnv("SB_YAML_SCHEMA_DIR", schemaDir)
    harness.backupAndSetEnv("HOME", tempDir)
    
    // 디렉터리 생성
    if err := os.MkdirAll(schemaDir, 0755); err != nil {
        t.Fatalf("Failed to create schema dir: %v", err)
    }
    
    t.Cleanup(harness.cleanup)
    
    return harness
}

func (h *CLITestHarness) ExecuteCommand(args ...string) error {
    cmd := rootCmd
    cmd.SetOut(h.stdout)
    cmd.SetErr(h.stderr)
    cmd.SetArgs(args)
    
    return cmd.Execute()
}

func (h *CLITestHarness) CreateTestFile(path string, content string) error {
    fullPath := filepath.Join(h.tempDir, path)
    dir := filepath.Dir(fullPath)
    
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }
    
    return os.WriteFile(fullPath, []byte(content), 0644)
}

func (h *CLITestHarness) GetOutput() string {
    return h.stdout.String()
}

func (h *CLITestHarness) GetError() string {
    return h.stderr.String()
}

func (h *CLITestHarness) cleanup() {
    // 환경변수 복원
    for key, value := range h.originalEnv {
        if value == "" {
            os.Unsetenv(key)
        } else {
            os.Setenv(key, value)
        }
    }
}

func (h *CLITestHarness) backupAndSetEnv(key, value string) {
    h.originalEnv[key] = os.Getenv(key)
    os.Setenv(key, value)
}
```

#### 5.1.2 Format 명령어 테스트 개선
**파일**: `cmd/format_test.go`

**리팩터링된 테스트**:
```go
func TestFormatCommand(t *testing.T) {
    harness := NewCLITestHarness(t)
    
    // 테스트 스키마 생성
    schemaContent := `version:
services:`
    err := harness.CreateTestFile("schemas/compose.yaml", schemaContent)
    assert.NoError(t, err)
    
    // 테스트 YAML 파일 생성
    yamlContent := `services:
  web:
    image: nginx
version: '3.8'`
    err = harness.CreateTestFile("test.yml", yamlContent)
    assert.NoError(t, err)
    
    // 명령어 실행
    err = harness.ExecuteCommand("format", "compose", filepath.Join(harness.tempDir, "test.yml"))
    assert.NoError(t, err)
    
    // 결과 검증
    output := harness.GetOutput()
    assert.Contains(t, output, "Formatting")
    assert.Contains(t, output, "1 file(s)")
    
    // 파일 내용 검증
    formatted, err := os.ReadFile(filepath.Join(harness.tempDir, "test.yml"))
    assert.NoError(t, err)
    assert.True(t, strings.HasPrefix(string(formatted), "version:"))
}

func TestFormatDryRun(t *testing.T) {
    harness := NewCLITestHarness(t)
    
    // 테스트 설정 (위와 동일)
    // ...
    
    // 원본 내용 저장
    originalContent, _ := os.ReadFile(filepath.Join(harness.tempDir, "test.yml"))
    
    // Dry run 실행
    err := harness.ExecuteCommand("format", "compose", filepath.Join(harness.tempDir, "test.yml"), "--dry-run")
    assert.NoError(t, err)
    
    // Dry run 표시 확인
    output := harness.GetOutput()
    assert.Contains(t, output, "DRY RUN")
    
    // 파일이 변경되지 않았는지 확인
    afterContent, _ := os.ReadFile(filepath.Join(harness.tempDir, "test.yml"))
    assert.Equal(t, originalContent, afterContent)
}
```

#### 5.1.3 Schema 명령어 테스트 개선
**파일**: `cmd/schema_test.go`

**개선된 테스트**:
```go
func TestSchemaCommands(t *testing.T) {
    harness := NewCLITestHarness(t)
    
    t.Run("schema gen", func(t *testing.T) {
        // 테스트 YAML 생성
        yamlContent := `name: test
version: 1.0.0
metadata:
  author: tester`
        
        err := harness.CreateTestFile("source.yml", yamlContent)
        assert.NoError(t, err)
        
        // 스키마 생성
        err = harness.ExecuteCommand("schema", "gen", "test-schema", filepath.Join(harness.tempDir, "source.yml"))
        assert.NoError(t, err)
        
        // 출력 검증
        output := harness.GetOutput()
        assert.Contains(t, output, "name:")
        assert.Contains(t, output, "version:")
        assert.Contains(t, output, "metadata:")
    })
    
    t.Run("schema set", func(t *testing.T) {
        // 스키마 파일 생성
        schemaContent := `name:
version:
description:`
        
        err := harness.CreateTestFile("test.schema.yaml", schemaContent)
        assert.NoError(t, err)
        
        // 스키마 설정
        err = harness.ExecuteCommand("schema", "set", "test-schema", filepath.Join(harness.tempDir, "test.schema.yaml"))
        assert.NoError(t, err)
        
        // 성공 메시지 확인
        output := harness.GetOutput()
        assert.Contains(t, output, "saved successfully")
        
        // 파일 존재 확인
        schemaPath := filepath.Join(harness.schemaDir, "test-schema.yaml")
        assert.FileExists(t, schemaPath)
    })
    
    t.Run("schema list", func(t *testing.T) {
        // 리스트 명령어 실행
        err := harness.ExecuteCommand("schema", "list")
        assert.NoError(t, err)
        
        // 출력 검증
        output := harness.GetOutput()
        assert.Contains(t, output, "Available schemas")
    })
}
```

### Task 5.2: E2E 테스트 안정화 (3시간)

#### 5.2.1 바이너리 빌드 자동화
**파일**: `tests/e2e/setup_test.go`

```go
package e2e

import (
    "os"
    "os/exec"
    "path/filepath"
    "testing"
)

var binaryPath string

func TestMain(m *testing.M) {
    // 테스트 전 바이너리 빌드
    if err := buildBinary(); err != nil {
        fmt.Printf("Failed to build binary: %v\n", err)
        os.Exit(1)
    }
    
    // 테스트 실행
    code := m.Run()
    
    // 테스트 후 정리
    cleanup()
    
    os.Exit(code)
}

func buildBinary() error {
    // 프로젝트 루트로 이동
    wd, err := os.Getwd()
    if err != nil {
        return err
    }
    
    projectRoot := filepath.Join(wd, "..", "..")
    binaryPath = filepath.Join(projectRoot, "sb-yaml-test")
    
    // 빌드 명령어 실행
    cmd := exec.Command("go", "build", "-o", binaryPath, ".")
    cmd.Dir = projectRoot
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    
    return cmd.Run()
}

func cleanup() {
    if binaryPath != "" {
        os.Remove(binaryPath)
    }
}

func getBinaryPath() string {
    return binaryPath
}
```

#### 5.2.2 E2E 테스트 환경 격리
**파일**: `tests/e2e/harness.go`

```go
package e2e

import (
    "os"
    "os/exec"
    "path/filepath"
    "testing"
)

type E2EHarness struct {
    tempDir    string
    schemaDir  string
    binaryPath string
    env        []string
}

func NewE2EHarness(t *testing.T) *E2EHarness {
    tempDir := t.TempDir()
    schemaDir := filepath.Join(tempDir, "schemas")
    
    // 디렉터리 생성
    if err := os.MkdirAll(schemaDir, 0755); err != nil {
        t.Fatalf("Failed to create schema dir: %v", err)
    }
    
    harness := &E2EHarness{
        tempDir:    tempDir,
        schemaDir:  schemaDir,
        binaryPath: getBinaryPath(),
        env: []string{
            "SB_YAML_SCHEMA_DIR=" + schemaDir,
            "HOME=" + tempDir,
        },
    }
    
    return harness
}

func (h *E2EHarness) RunCommand(args ...string) *exec.Cmd {
    cmd := exec.Command(h.binaryPath, args...)
    cmd.Env = append(os.Environ(), h.env...)
    cmd.Dir = h.tempDir
    return cmd
}

func (h *E2EHarness) CreateFile(path, content string) error {
    fullPath := filepath.Join(h.tempDir, path)
    dir := filepath.Dir(fullPath)
    
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }
    
    return os.WriteFile(fullPath, []byte(content), 0644)
}

func (h *E2EHarness) ReadFile(path string) ([]byte, error) {
    return os.ReadFile(filepath.Join(h.tempDir, path))
}

func (h *E2EHarness) FileExists(path string) bool {
    _, err := os.Stat(filepath.Join(h.tempDir, path))
    return err == nil
}
```

#### 5.2.3 E2E 테스트 시나리오 개선
**파일**: `tests/e2e/workflow_test.go`

```go
func TestCompleteWorkflow(t *testing.T) {
    harness := NewE2EHarness(t)
    
    // 1. 테스트 YAML 파일 생성
    yamlContent := `database:
  host: localhost
  port: 5432
name: MyApp
version: 1.0.0`
    
    err := harness.CreateFile("app.yml", yamlContent)
    assert.NoError(t, err)
    
    // 2. 스키마 생성
    cmd := harness.RunCommand("schema", "gen", "app", "app.yml")
    output, err := cmd.Output()
    assert.NoError(t, err)
    
    // 3. 스키마 저장
    err = harness.CreateFile("app.schema.yaml", string(output))
    assert.NoError(t, err)
    
    cmd = harness.RunCommand("schema", "set", "app", "app.schema.yaml")
    err = cmd.Run()
    assert.NoError(t, err)
    
    // 4. 스키마 목록 확인
    cmd = harness.RunCommand("schema", "list")
    output, err = cmd.Output()
    assert.NoError(t, err)
    assert.Contains(t, string(output), "app")
    
    // 5. 포맷팅 검사 (실패 예상)
    cmd = harness.RunCommand("check", "app", "app.yml")
    err = cmd.Run()
    assert.Error(t, err) // 포맷되지 않은 상태이므로 실패 예상
    
    // 6. 포맷팅 실행
    cmd = harness.RunCommand("format", "app", "app.yml")
    err = cmd.Run()
    assert.NoError(t, err)
    
    // 7. 포맷팅 검사 (성공 예상)
    cmd = harness.RunCommand("check", "app", "app.yml")
    err = cmd.Run()
    assert.NoError(t, err)
    
    // 8. 포맷된 내용 확인
    formatted, err := harness.ReadFile("app.yml")
    assert.NoError(t, err)
    assert.True(t, strings.HasPrefix(string(formatted), "name:"))
}
```

### Task 5.3: 병렬 테스트 지원 (2시간)

#### 5.3.1 테스트 격리 보장
**파일**: `cmd/parallel_test.go`

```go
func TestParallelFormatCommands(t *testing.T) {
    t.Parallel()
    
    testCases := []struct {
        name       string
        schemaType string
        yamlContent string
    }{
        {
            name:       "docker-compose",
            schemaType: "compose",
            yamlContent: `version: '3.8'
services:
  web:
    image: nginx`,
        },
        {
            name:       "kubernetes",
            schemaType: "k8s",
            yamlContent: `apiVersion: v1
kind: Pod
metadata:
  name: test`,
        },
    }
    
    for _, tc := range testCases {
        tc := tc // 클로저 캡처
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            
            harness := NewCLITestHarness(t)
            
            // 각 테스트는 독립적인 환경에서 실행
            err := harness.CreateTestFile("test.yml", tc.yamlContent)
            assert.NoError(t, err)
            
            // 테스트 로직 실행
            // ...
        })
    }
}
```

#### 5.3.2 리소스 경합 방지
**파일**: `tests/e2e/parallel_test.go`

```go
func TestParallelE2EScenarios(t *testing.T) {
    t.Parallel()
    
    scenarios := []func(*testing.T){
        testDockerComposeWorkflow,
        testKubernetesWorkflow,
        testMultiDocumentWorkflow,
        testLargeFileWorkflow,
    }
    
    for i, scenario := range scenarios {
        scenario := scenario // 클로저 캡처
        t.Run(fmt.Sprintf("scenario_%d", i), func(t *testing.T) {
            t.Parallel()
            scenario(t)
        })
    }
}

func testDockerComposeWorkflow(t *testing.T) {
    harness := NewE2EHarness(t)
    // 독립적인 Docker Compose 테스트
}

func testKubernetesWorkflow(t *testing.T) {
    harness := NewE2EHarness(t)
    // 독립적인 Kubernetes 테스트
}
```

### Task 5.4: 에러 시나리오 테스트 (1시간)

#### 5.4.1 잘못된 입력 처리
**파일**: `cmd/error_handling_test.go`

```go
func TestErrorHandling(t *testing.T) {
    harness := NewCLITestHarness(t)
    
    tests := []struct {
        name        string
        args        []string
        expectError bool
        errorMsg    string
    }{
        {
            name:        "missing arguments",
            args:        []string{"format"},
            expectError: true,
            errorMsg:    "requires at least",
        },
        {
            name:        "non-existent schema",
            args:        []string{"format", "non-existent", "test.yml"},
            expectError: true,
            errorMsg:    "failed to load schema",
        },
        {
            name:        "non-existent file",
            args:        []string{"format", "test", "/non/existent/file.yml"},
            expectError: true,
            errorMsg:    "failed to read",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := harness.ExecuteCommand(tt.args...)
            
            if tt.expectError {
                assert.Error(t, err)
                if tt.errorMsg != "" {
                    errorOutput := harness.GetError()
                    assert.Contains(t, errorOutput, tt.errorMsg)
                }
            } else {
                assert.NoError(t, err)
            }
            
            // 버퍼 초기화
            harness.stdout.Reset()
            harness.stderr.Reset()
        })
    }
}
```

## CI/CD 최적화

### 테스트 실행 스크립트
**파일**: `scripts/run-tests.sh`

```bash
#!/bin/bash
set -e

echo "Running unit tests..."
go test -race -timeout=5m ./internal/...

echo "Running integration tests..."
go test -race -timeout=10m ./cmd/...

echo "Building binary for E2E tests..."
go build -o sb-yaml-test .

echo "Running E2E tests..."
go test -race -timeout=15m ./tests/e2e/...

echo "Cleaning up..."
rm -f sb-yaml-test

echo "All tests passed!"
```

### GitHub Actions 워크플로우
**파일**: `.github/workflows/test.yml`

```yaml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: [1.19, 1.20, 1.21]
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: ${{ matrix.go-version }}
    
    - name: Cache dependencies
      uses: actions/cache@v3
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
    
    - name: Install dependencies
      run: go mod download
    
    - name: Run tests
      run: |
        chmod +x scripts/run-tests.sh
        ./scripts/run-tests.sh
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
```

## 예상 소요 시간
- **총 10시간**
- Task 5.1: 4시간 (CLI 테스트 모듈화)
- Task 5.2: 3시간 (E2E 테스트 안정화)
- Task 5.3: 2시간 (병렬 테스트 지원)
- Task 5.4: 1시간 (에러 시나리오)

## 성공 지표
- [ ] 모든 CLI 테스트 격리된 환경에서 실행
- [ ] E2E 테스트 바이너리 의존성 자동 해결
- [ ] 병렬 테스트 실행 시 충돌 없음
- [ ] CI/CD 파이프라인에서 모든 테스트 안정적 실행
- [ ] 테스트 실행 시간 15분 이내

## 다음 단계
통합 테스트가 안정화되면 `06-ci-optimization.md`로 진행