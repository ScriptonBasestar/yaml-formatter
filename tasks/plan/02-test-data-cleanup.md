# 테스트 데이터 정리 계획

## 목표
일관성 있고 CI/CD에 최적화된 테스트 데이터 구조 구축

## 현재 문제점 분석

### 1. 테스트 데이터 불일치
```
Expected vs Actual 포맷 차이
스키마와 실제 데이터 간 순서 불일치
빈 줄 처리 차이
```

### 2. 파일 경로 의존성
```
하드코딩된 상대 경로: ../../testdata/
환경별 경로 차이
테스트 격리 불완전
```

### 3. 데이터 품질 문제
```
일부 테스트 파일의 YAML 형식 오류
스키마 정의와 맞지 않는 구조
중복된 테스트 케이스
```

## 수정 작업 목록

### Task 2.1: 테스트 데이터 표준화 (3시간)

**목표**: 모든 테스트 데이터를 표준 형식으로 통일

#### 2.1.1 Docker Compose 테스트 데이터
**파일**: `testdata/formatting/input/unordered-docker-compose.yml`
**파일**: `testdata/formatting/expected/unordered-docker-compose.yml`

**문제**: 
```yaml
# Current order in output
services:
  web:
    ports: [...]
    environment: [...]
    image: [...]
    depends_on: [...]

# Expected order from schema
services:
  web:
    image: [...]
    depends_on: [...]
    ports: [...]
    environment: [...]
```

**해결책**:
1. 스키마 순서에 맞는 expected 파일 재생성
2. 포맷터 출력과 일치하도록 조정
3. 빈 줄 규칙 명확화

#### 2.1.2 Kubernetes 테스트 데이터
**파일**: `testdata/formatting/input/unordered-kubernetes.yml`
**파일**: `testdata/formatting/expected/unordered-kubernetes.yml`

**문제**: apiVersion, kind 순서 불일치

**해결책**:
1. Kubernetes 스키마 검증
2. 표준 k8s 리소스 순서 적용
3. metadata.name, metadata.namespace 순서 수정

#### 2.1.3 Edge Case 데이터 정제
**파일들**: `testdata/edge-cases/*.yml`

**검증 필요 파일**:
- `empty.yml` - 빈 파일 처리
- `special-characters.yml` - 유니코드/이모지 처리
- `very-deep-nesting.yml` - 깊은 중첩 구조

### Task 2.2: 테스트 데이터 임베딩 (2시간)

**목표**: 파일 시스템 의존성 제거 및 테스트 격리

#### 2.2.1 Go 임베딩 활용
**파일**: `testdata/fixtures.go`

**현재 구조**:
```go
//go:embed valid/*.yml invalid/*.yml
var TestFiles embed.FS
```

**개선점**:
1. 더 세분화된 임베딩
2. 테스트별 데이터 분리
3. 성능 최적화

**새로운 구조**:
```go
//go:embed formatting/input/*.yml
var FormattingInputFiles embed.FS

//go:embed formatting/expected/*.yml  
var FormattingExpectedFiles embed.FS

//go:embed valid/*.yml
var ValidFiles embed.FS

//go:embed invalid/*.yml
var InvalidFiles embed.FS
```

#### 2.2.2 테스트 헬퍼 함수
**파일**: `testdata/helpers.go`

**구현할 함수들**:
```go
func GetFormattingPair(name string) (input, expected []byte, err error)
func GetValidTestFile(name string) ([]byte, error)
func GetInvalidTestFile(name string) ([]byte, error)
func GetSchemaForTest(schemaType string) (*schema.Schema, error)
```

### Task 2.3: 스키마-데이터 일치성 확보 (2시간)

**목표**: 스키마 정의와 테스트 데이터 간 완벽한 일치성

#### 2.3.1 스키마 검증 도구
**파일**: `testdata/validate.go`

**기능**:
1. 테스트 데이터가 해당 스키마를 만족하는지 검증
2. 포맷팅 결과가 예상과 일치하는지 확인
3. 자동 수정 기능 제공

**구현**:
```go
func ValidateTestData(schemaName string, yamlData []byte) error
func GenerateExpectedOutput(schemaName string, input []byte) ([]byte, error)
func UpdateAllExpectedFiles() error
```

#### 2.3.2 자동 생성 스크립트
**파일**: `scripts/generate-expected.go`

**기능**:
1. 현재 포맷터 로직으로 expected 파일 자동 생성
2. 기존 파일과 비교하여 차이점 리포트
3. 대량 업데이트 지원

### Task 2.4: 테스트 데이터 최적화 (1시간)

**목표**: CI/CD 성능 향상을 위한 데이터 최적화

#### 2.4.1 크기 최적화
- 불필요한 테스트 파일 제거
- 중복 케이스 통합
- 최소한의 의미 있는 데이터만 유지

#### 2.4.2 로딩 성능 개선
- 지연 로딩 구현
- 캐싱 메커니즘 추가
- 병렬 처리 지원

## 구현 순서

### 1단계: 기존 데이터 분석
```bash
# 현재 테스트 실행하여 모든 차이점 수집
go test ./internal/formatter -v > test_output.log 2>&1

# 스키마별 문제점 분석
grep -A 10 -B 5 "Formatted output doesn't match" test_output.log
```

### 2단계: Docker Compose 데이터 수정
```bash
# 포맷터로 실제 출력 생성
./sb-yaml format compose testdata/formatting/input/unordered-docker-compose.yml --dry-run

# Expected 파일 업데이트
cp output testdata/formatting/expected/unordered-docker-compose.yml
```

### 3단계: Kubernetes 데이터 수정
```bash
# 동일한 과정을 Kubernetes 데이터에 적용
./sb-yaml format k8s testdata/formatting/input/unordered-kubernetes.yml --dry-run
```

### 4단계: 검증 및 테스트
```bash
go test ./internal/formatter -v -run TestFormatterWithTestData
```

## 자동화 스크립트

### 스크립트 1: 테스트 데이터 검증
**파일**: `scripts/validate-testdata.sh`

```bash
#!/bin/bash
set -e

echo "Validating test data consistency..."

# 모든 포맷팅 테스트 케이스 검증
for input_file in testdata/formatting/input/*.yml; do
    base_name=$(basename "$input_file")
    expected_file="testdata/formatting/expected/$base_name"
    
    if [ ! -f "$expected_file" ]; then
        echo "Missing expected file: $expected_file"
        exit 1
    fi
    
    echo "Validating $base_name..."
    # 실제 검증 로직
done

echo "All test data validated successfully!"
```

### 스크립트 2: Expected 파일 생성
**파일**: `scripts/generate-expected.sh`

```bash
#!/bin/bash
set -e

echo "Generating expected outputs..."

# 빌드 확인
if [ ! -f "./sb-yaml" ]; then
    echo "Building sb-yaml..."
    go build -o sb-yaml .
fi

# 각 스키마별 expected 파일 생성
declare -A schemas=(
    ["compose"]="docker-compose"
    ["k8s"]="kubernetes"
)

for schema in "${!schemas[@]}"; do
    pattern="${schemas[$schema]}"
    echo "Processing $schema files..."
    
    for input_file in testdata/formatting/input/*${pattern}*.yml; do
        if [ -f "$input_file" ]; then
            base_name=$(basename "$input_file")
            expected_file="testdata/formatting/expected/$base_name"
            
            echo "  Generating $base_name..."
            ./sb-yaml format "$schema" "$input_file" --dry-run > "$expected_file"
        fi
    done
done

echo "Expected files generated successfully!"
```

## 테스트 확인 방법

### 1단계: 개별 파일 테스트
```bash
go test ./internal/formatter -v -run "TestFormatterWithTestData/unordered-docker-compose"
```

### 2단계: 전체 포맷팅 테스트
```bash
go test ./internal/formatter -v -run TestFormatterWithTestData
```

### 3단계: 모든 formatter 테스트
```bash
go test ./internal/formatter -v
```

## 예상 소요 시간
- **총 8시간**
- Task 2.1: 3시간 (데이터 표준화)
- Task 2.2: 2시간 (임베딩 구현)
- Task 2.3: 2시간 (일치성 확보)
- Task 2.4: 1시간 (성능 최적화)

## 성공 지표
- [ ] 모든 포맷팅 테스트 expected/actual 일치
- [ ] 파일 시스템 의존성 제거
- [ ] 테스트 데이터 로딩 시간 100ms 이내
- [ ] 자동 검증 스크립트 동작

## 다음 단계
테스트 데이터가 정리되면 `03-formatter-core.md`로 진행