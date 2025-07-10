# CI/CD 최적화 계획

## 목표
CI/CD 파이프라인에서 최적의 성능과 안정성을 제공하는 테스트 환경 구축

## 현재 상황 및 최적화 포인트

### 1. 테스트 실행 시간 최적화
```
현재 예상 시간: 20-30분
목표 시간: 5-10분
병목 구간: E2E 테스트, 대용량 파일 테스트
```

### 2. 리소스 사용량 최적화
```
메모리 사용량 최소화
CPU 병렬 처리 활용
네트워크 의존성 제거
```

### 3. 테스트 안정성 강화
```
플래키 테스트 제거
환경별 차이 최소화
재시도 메커니즘 구현
```

## 최적화 작업 목록

### Task 6.1: 테스트 분류 및 선택적 실행 (2시간)

#### 6.1.1 테스트 레벨 분류
**파일**: `scripts/test-categories.sh`

**테스트 분류 체계**:
```bash
#!/bin/bash

# 테스트 카테고리 정의
UNIT_TESTS="./internal/..."
INTEGRATION_TESTS="./cmd/..."
E2E_TESTS="./tests/e2e/..."
SMOKE_TESTS="./tests/smoke/..."

# 실행 모드별 테스트 선택
case "${TEST_MODE}" in
    "fast")
        echo "Running fast tests (unit only)..."
        go test -short ${UNIT_TESTS}
        ;;
    "ci")
        echo "Running CI tests (unit + integration)..."
        go test -short ${UNIT_TESTS} ${INTEGRATION_TESTS}
        ;;
    "full")
        echo "Running full test suite..."
        go test ${UNIT_TESTS} ${INTEGRATION_TESTS} ${E2E_TESTS}
        ;;
    "smoke")
        echo "Running smoke tests..."
        go test ${SMOKE_TESTS}
        ;;
    *)
        echo "Unknown test mode: ${TEST_MODE}"
        exit 1
        ;;
esac
```

#### 6.1.2 빌드 태그 활용
**파일**: 각 테스트 파일

**성능 집약적 테스트 분리**:
```go
//go:build integration
// +build integration

package formatter

import "testing"

func TestLargeFileProcessing(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping large file test in short mode")
    }
    
    // 성능 집약적 테스트 로직
}
```

**E2E 테스트 분리**:
```go
//go:build e2e
// +build e2e

package e2e

func TestFullWorkflow(t *testing.T) {
    // E2E 테스트만 실행할 때만 동작
}
```

#### 6.1.3 조건부 테스트 실행
**파일**: `Makefile` 업데이트

```makefile
# 기존 targets 수정
.PHONY: test-fast test-ci test-full test-smoke

# 빠른 테스트 (개발 중)
test-fast:
	go test -short -race ./internal/...

# CI용 테스트 (PR 검증)
test-ci:
	go test -short -race ./internal/... ./cmd/...

# 전체 테스트 (릴리스 전)
test-full:
	go test -race ./...
	go test -tags=integration -race ./...
	go test -tags=e2e -race ./tests/e2e/...

# 스모크 테스트 (배포 후 검증)
test-smoke:
	go test -tags=smoke -race ./tests/smoke/...

# 병렬 실행
test-parallel:
	go test -race -parallel=4 ./internal/...
	go test -race -parallel=2 ./cmd/...

# 코버리지 포함
test-coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
```

### Task 6.2: 테스트 데이터 최적화 (3시간)

#### 6.2.1 테스트 데이터 크기 최소화
**파일**: `testdata/optimized/`

**경량화된 테스트 데이터 생성**:
```go
// testdata/mini_samples.go
package testdata

// 최소한의 의미 있는 데이터만 포함
var MiniSamples = map[string]string{
    "docker-compose-mini": `version: '3.8'
services:
  web:
    image: nginx`,
    
    "kubernetes-mini": `apiVersion: v1
kind: Pod
metadata:
  name: test`,
    
    "complex-mini": `app:
  config:
    debug: true
items:
  - name: test`,
}

func GetMiniSample(key string) []byte {
    return []byte(MiniSamples[key])
}
```

#### 6.2.2 테스트 데이터 캐싱
**파일**: `testdata/cache.go`

**메모리 캐싱 구현**:
```go
package testdata

import (
    "sync"
    "time"
)

type CachedData struct {
    content   []byte
    timestamp time.Time
}

type DataCache struct {
    cache map[string]*CachedData
    mutex sync.RWMutex
    ttl   time.Duration
}

func NewDataCache(ttl time.Duration) *DataCache {
    return &DataCache{
        cache: make(map[string]*CachedData),
        ttl:   ttl,
    }
}

func (dc *DataCache) Get(key string) ([]byte, bool) {
    dc.mutex.RLock()
    defer dc.mutex.RUnlock()
    
    data, exists := dc.cache[key]
    if !exists {
        return nil, false
    }
    
    // TTL 체크
    if time.Since(data.timestamp) > dc.ttl {
        return nil, false
    }
    
    return data.content, true
}

func (dc *DataCache) Set(key string, content []byte) {
    dc.mutex.Lock()
    defer dc.mutex.Unlock()
    
    dc.cache[key] = &CachedData{
        content:   content,
        timestamp: time.Now(),
    }
}

// 글로벌 캐시 인스턴스
var globalCache = NewDataCache(5 * time.Minute)

func GetCachedTestData(key string) []byte {
    if content, found := globalCache.Get(key); found {
        return content
    }
    
    // 실제 데이터 로드
    content := loadTestData(key)
    globalCache.Set(key, content)
    return content
}
```

#### 6.2.3 지연 로딩 구현
**파일**: `testdata/lazy.go`

```go
package testdata

import "sync"

type LazyLoader struct {
    key    string
    loader func() []byte
    once   sync.Once
    data   []byte
}

func NewLazyLoader(key string, loader func() []byte) *LazyLoader {
    return &LazyLoader{
        key:    key,
        loader: loader,
    }
}

func (ll *LazyLoader) Get() []byte {
    ll.once.Do(func() {
        ll.data = ll.loader()
    })
    return ll.data
}

// 전역 지연 로더들
var (
    DockerComposeLoader = NewLazyLoader("docker-compose", func() []byte {
        return MustLoadEmbedded("formatting/input/docker-compose.yml")
    })
    
    KubernetesLoader = NewLazyLoader("kubernetes", func() []byte {
        return MustLoadEmbedded("formatting/input/kubernetes.yml")
    })
)
```

### Task 6.3: 병렬 처리 최적화 (2시간)

#### 6.3.1 테스트 병렬성 설정
**파일**: `scripts/parallel-test.sh`

```bash
#!/bin/bash

# CPU 코어 수 감지
CPU_CORES=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)
PARALLEL_LEVEL=$((CPU_CORES / 2))

echo "Detected ${CPU_CORES} CPU cores, using ${PARALLEL_LEVEL} parallel test processes"

# 패키지별 병렬 실행
go test -race -parallel=${PARALLEL_LEVEL} -timeout=10m ./internal/config &
go test -race -parallel=${PARALLEL_LEVEL} -timeout=10m ./internal/utils &
go test -race -parallel=${PARALLEL_LEVEL} -timeout=15m ./internal/schema &
go test -race -parallel=${PARALLEL_LEVEL} -timeout=20m ./internal/formatter &

# 모든 백그라운드 작업 완료 대기
wait

echo "Unit tests completed. Starting integration tests..."

# 통합 테스트는 순차 실행 (파일 시스템 의존성 때문)
go test -race -timeout=15m ./cmd/...

echo "All tests completed successfully!"
```

#### 6.3.2 리소스 풀링
**파일**: `internal/testing/pools.go`

```go
package testing

import (
    "sync"
    "yaml-formatter/internal/schema"
    "yaml-formatter/internal/formatter"
)

// 테스트용 객체 풀
var (
    schemaPool = sync.Pool{
        New: func() interface{} {
            return createTestSchema()
        },
    }
    
    formatterPool = sync.Pool{
        New: func() interface{} {
            return formatter.NewFormatter(createTestSchema())
        },
    }
)

func GetTestSchema() *schema.Schema {
    return schemaPool.Get().(*schema.Schema)
}

func PutTestSchema(s *schema.Schema) {
    // 재설정
    s.Reset()
    schemaPool.Put(s)
}

func GetTestFormatter() *formatter.Formatter {
    return formatterPool.Get().(*formatter.Formatter)
}

func PutTestFormatter(f *formatter.Formatter) {
    // 재설정
    f.Reset()
    formatterPool.Put(f)
}

// 테스트에서 사용 예시
func TestWithPooledObjects(t *testing.T) {
    formatter := GetTestFormatter()
    defer PutTestFormatter(formatter)
    
    // 테스트 로직
}
```

### Task 6.4: CI/CD 파이프라인 최적화 (3시간)

#### 6.4.1 GitHub Actions 워크플로우 최적화
**파일**: `.github/workflows/ci.yml`

```yaml
name: CI

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  # 빠른 검증 (필수)
  fast-checks:
    runs-on: ubuntu-latest
    timeout-minutes: 5
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
        cache: true
    
    - name: Run fast tests
      run: make test-fast
    
    - name: Lint
      uses: golangci/golangci-lint-action@v3
      with:
        version: latest
        args: --timeout=5m

  # 상세 테스트 (병렬)
  detailed-tests:
    needs: fast-checks
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest, macos-latest]
        go-version: ['1.19', '1.20', '1.21']
    runs-on: ${{ matrix.os }}
    timeout-minutes: 15
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: ${{ matrix.go-version }}
        cache: true
    
    - name: Run CI tests
      run: make test-ci
    
    - name: Upload coverage
      if: matrix.os == 'ubuntu-latest' && matrix.go-version == '1.21'
      uses: codecov/codecov-action@v3

  # E2E 테스트 (선택적)
  e2e-tests:
    needs: fast-checks
    if: github.event_name == 'push' || contains(github.event.pull_request.labels.*.name, 'e2e')
    runs-on: ubuntu-latest
    timeout-minutes: 10
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
        cache: true
    
    - name: Run E2E tests
      run: make test-e2e
```

#### 6.4.2 의존성 캐싱 최적화
**파일**: `.github/workflows/cache.yml`

```yaml
# 공통 캐싱 설정
- name: Cache Go modules
  uses: actions/cache@v3
  with:
    path: |
      ~/.cache/go-build
      ~/go/pkg/mod
    key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
    restore-keys: |
      ${{ runner.os }}-go-

- name: Cache test data
  uses: actions/cache@v3
  with:
    path: |
      testdata/.cache
      .test-cache
    key: ${{ runner.os }}-testdata-${{ hashFiles('testdata/**') }}
    restore-keys: |
      ${{ runner.os }}-testdata-
```

#### 6.4.3 조건부 실행 최적화
**파일**: `.github/workflows/conditional.yml`

```yaml
# 변경된 파일에 따른 조건부 실행
- name: Check changed files
  id: changes
  uses: dorny/paths-filter@v2
  with:
    filters: |
      core:
        - 'internal/**'
        - 'cmd/**'
        - 'go.mod'
        - 'go.sum'
      tests:
        - 'tests/**'
        - '*_test.go'
      docs:
        - '*.md'
        - 'docs/**'

- name: Run core tests
  if: steps.changes.outputs.core == 'true'
  run: make test-ci

- name: Run full test suite
  if: steps.changes.outputs.tests == 'true'
  run: make test-full

- name: Skip tests for docs-only changes
  if: steps.changes.outputs.docs == 'true' && steps.changes.outputs.core == 'false'
  run: echo "Skipping tests for documentation-only changes"
```

### Task 6.5: 모니터링 및 리포팅 (1시간)

#### 6.5.1 테스트 메트릭 수집
**파일**: `scripts/collect-metrics.sh`

```bash
#!/bin/bash

# 테스트 실행 시간 측정
start_time=$(date +%s)

# 테스트 실행
go test -json ./... > test-results.json

end_time=$(date +%s)
duration=$((end_time - start_time))

# 결과 분석
total_tests=$(jq '[.[] | select(.Action == "pass" or .Action == "fail")] | length' test-results.json)
passed_tests=$(jq '[.[] | select(.Action == "pass")] | length' test-results.json)
failed_tests=$(jq '[.[] | select(.Action == "fail")] | length' test-results.json)

echo "Test Metrics:"
echo "Duration: ${duration} seconds"
echo "Total: ${total_tests}"
echo "Passed: ${passed_tests}"
echo "Failed: ${failed_tests}"
echo "Success Rate: $(bc -l <<< "scale=2; ${passed_tests} * 100 / ${total_tests")%"

# 느린 테스트 찾기
jq -r '.[] | select(.Action == "pass" and .Elapsed != null) | "\(.Elapsed)s \(.Package) \(.Test)"' test-results.json | sort -nr | head -10 > slow-tests.txt

echo "Top 10 slowest tests:"
cat slow-tests.txt
```

#### 6.5.2 플래키 테스트 감지
**파일**: `scripts/flaky-test-detector.sh`

```bash
#!/bin/bash

# 동일한 테스트를 여러 번 실행하여 플래키 테스트 감지
TEST_RUNS=10
FLAKY_THRESHOLD=2

echo "Running flaky test detection (${TEST_RUNS} runs)..."

for i in $(seq 1 ${TEST_RUNS}); do
    echo "Run $i/$TEST_RUNS"
    go test -json ./... > "test-run-$i.json"
done

# 실패한 테스트들 수집
for run_file in test-run-*.json; do
    jq -r '.[] | select(.Action == "fail") | "\(.Package) \(.Test)"' "$run_file"
done | sort | uniq -c | sort -nr > flaky-candidates.txt

# 임계값 이상으로 실패한 테스트들 리포트
echo "Potentially flaky tests (failed $FLAKY_THRESHOLD+ times):"
awk "\$1 >= $FLAKY_THRESHOLD" flaky-candidates.txt

# 정리
rm test-run-*.json
```

### Task 6.6: 성능 벤치마크 통합 (1시간)

#### 6.6.1 벤치마크 테스트 추가
**파일**: `internal/formatter/benchmark_test.go`

```go
package formatter

import (
    "testing"
    "yaml-formatter/testdata"
)

func BenchmarkFormatSmallFile(b *testing.B) {
    formatter := createBenchmarkFormatter()
    data := testdata.GetMiniSample("docker-compose-mini")
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        _, err := formatter.FormatContent(data)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkFormatLargeFile(b *testing.B) {
    formatter := createBenchmarkFormatter()
    data := generateLargeYAML(100 * 1024) // 100KB
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        _, err := formatter.FormatContent(data)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkParseYAML(b *testing.B) {
    parser := NewParser(true)
    data := testdata.GetMiniSample("complex-mini")
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        _, err := parser.ParseYAML(data)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

#### 6.6.2 성능 회귀 감지
**파일**: `scripts/benchmark-compare.sh`

```bash
#!/bin/bash

# 기준 브랜치의 벤치마크 실행
git checkout main
go test -bench=. -benchmem ./... > benchmark-main.txt

# 현재 브랜치의 벤치마크 실행  
git checkout -
go test -bench=. -benchmem ./... > benchmark-current.txt

# 결과 비교 (benchcmp 사용)
if command -v benchcmp >/dev/null 2>&1; then
    echo "Performance comparison:"
    benchcmp benchmark-main.txt benchmark-current.txt
else
    echo "benchcmp not installed. Installing..."
    go install golang.org/x/tools/cmd/benchcmp@latest
    benchcmp benchmark-main.txt benchmark-current.txt
fi

# 회귀 검사 (50% 이상 성능 저하 시 경고)
python3 << 'EOF'
import re
import sys

def parse_benchmark(filename):
    results = {}
    with open(filename, 'r') as f:
        for line in f:
            if 'Benchmark' in line and 'ns/op' in line:
                parts = line.split()
                name = parts[0]
                ns_per_op = int(parts[2])
                results[name] = ns_per_op
    return results

main_results = parse_benchmark('benchmark-main.txt')
current_results = parse_benchmark('benchmark-current.txt')

regressions = []
for name in main_results:
    if name in current_results:
        main_time = main_results[name]
        current_time = current_results[name]
        if current_time > main_time * 1.5:  # 50% slower
            regression = (current_time - main_time) / main_time * 100
            regressions.append((name, regression))

if regressions:
    print("PERFORMANCE REGRESSIONS DETECTED:")
    for name, regression in regressions:
        print(f"  {name}: {regression:.1f}% slower")
    sys.exit(1)
else:
    print("No significant performance regressions detected.")
EOF
```

## CI/CD 워크플로우 전체 구성

### 최종 Makefile
**파일**: `Makefile` (완성본)

```makefile
.PHONY: test test-fast test-ci test-full test-e2e test-smoke test-bench
.PHONY: lint fmt vet clean build coverage
.PHONY: ci-setup ci-test ci-benchmark

# 개발자용 빠른 테스트
test-fast:
	@echo "Running fast tests..."
	go test -short -race -timeout=5m ./internal/...

# CI용 표준 테스트
test-ci: ci-setup
	@echo "Running CI test suite..."
	go test -race -timeout=10m ./internal/... ./cmd/...

# 전체 테스트 (릴리스 전)
test-full: ci-setup
	@echo "Running full test suite..."
	go test -race -timeout=15m ./...
	go test -tags=integration -race -timeout=10m ./...

# E2E 테스트
test-e2e: build
	@echo "Running E2E tests..."
	go test -tags=e2e -race -timeout=10m ./tests/e2e/...

# 스모크 테스트
test-smoke: build
	@echo "Running smoke tests..."
	go test -tags=smoke -race -timeout=5m ./tests/smoke/...

# 벤치마크 테스트
test-bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem -timeout=10m ./...

# 코버리지 테스트
coverage:
	@echo "Generating coverage report..."
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# CI 환경 설정
ci-setup:
	@echo "Setting up CI environment..."
	go version
	go mod download
	go mod verify

# CI 전체 실행
ci-test: ci-setup test-ci lint vet

# CI 벤치마크 (성능 회귀 검사)
ci-benchmark:
	@echo "Running benchmark comparison..."
	./scripts/benchmark-compare.sh

# 린팅
lint:
	@echo "Running linter..."
	golangci-lint run --timeout=5m

# 포맷팅
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Vet 검사
vet:
	@echo "Running go vet..."
	go vet ./...

# 빌드
build:
	@echo "Building binary..."
	go build -o sb-yaml .

# 정리
clean:
	@echo "Cleaning up..."
	rm -f sb-yaml sb-yaml-test
	rm -f coverage.out coverage.html
	rm -f test-results.json benchmark-*.txt
	rm -rf .test-cache

# 기본 타겟
test: test-fast
```

## 예상 소요 시간
- **총 12시간**
- Task 6.1: 2시간 (테스트 분류)
- Task 6.2: 3시간 (데이터 최적화)
- Task 6.3: 2시간 (병렬 처리)
- Task 6.4: 3시간 (파이프라인 최적화)
- Task 6.5: 1시간 (모니터링)
- Task 6.6: 1시간 (벤치마크)

## 성공 지표
- [ ] CI 파이프라인 실행 시간 10분 이내
- [ ] 테스트 성공률 99% 이상
- [ ] 플래키 테스트 0개
- [ ] 코드 커버리지 85% 이상
- [ ] 성능 회귀 자동 감지
- [ ] 리소스 사용량 최적화 (메모리 100MB 이하)

## 최종 CI/CD 구조
```
PR 생성 → fast-checks (5분) → detailed-tests (15분) 병렬
                            → e2e-tests (10분) 선택적

main 브랜치 → full-test-suite (20분) → benchmark (5분) → deploy
```

이로써 CI/CD에 최적화된 안정적이고 빠른 테스트 환경이 완성됩니다.