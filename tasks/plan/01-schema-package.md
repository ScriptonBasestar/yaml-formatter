# Schema 패키지 테스트 수정 계획

## 목표
Schema 패키지의 모든 테스트를 안정화하여 formatter의 기반을 견고히 함

## 현재 문제점 분석

### 1. Schema 구조 불일치
```
TestLoaderSaveAndLoad: schema validation failed: schema must have at least one key defined
TestGenerateFromYAML: Order 생성 로직 불일치  
TestGetKeyOrder: 경로 기반 키 순서 추출 실패
```

### 2. 테스트 데이터 문제
```
TestLoaderWithRealTestData: 실제 파일 경로 참조 실패
TestSchemaString: 직렬화/역직렬화 불일치
```

## 수정 작업 목록

### Task 1.1: Schema 구조 표준화 (2시간)
**파일**: `internal/schema/schema.go`

**문제**: Order 필드와 Keys 필드 간 불일치
```go
// 현재 문제
type Schema struct {
    Keys  map[string]interface{} // 실제 구조
    Order []string               // 플랫 리스트
}
```

**해결책**:
1. Order 생성 로직 수정 - buildOrderFromKeys 함수 개선
2. Keys와 Order 동기화 보장
3. 검증 로직 강화

**검증 방법**:
```bash
go test ./internal/schema -v -run TestLoadFromBytes
go test ./internal/schema -v -run TestGenerateFromYAML
```

### Task 1.2: Loader 테스트 데이터 수정 (1시간)
**파일**: `internal/schema/loader_test.go`

**문제**: Schema 생성 시 필수 필드 누락
```go
// 문제 코드
s := &Schema{
    Name: "test",
    Order: []string{"key1"},  // Keys 필드 누락
}
```

**해결책**:
1. 모든 테스트에서 Keys 필드 포함
2. Order와 Keys 일치성 확보
3. 테스트용 Schema 생성 헬퍼 함수 추가

**구현**:
```go
func createTestSchema(name string, keys []string) *Schema {
    schema := &Schema{
        Name: name,
        Keys: make(map[string]interface{}),
        Order: keys,
    }
    for _, key := range keys {
        schema.Keys[key] = nil
    }
    return schema
}
```

### Task 1.3: Schema Validation 강화 (1시간)
**파일**: `internal/schema/schema.go`

**문제**: 불완전한 검증 로직
```go
func (s *Schema) Validate() error {
    // 현재: 기본적인 nil 체크만
    // 필요: Keys-Order 일치성, 순환 참조 등
}
```

**해결책**:
1. Keys와 Order 일치성 검증
2. 순환 참조 검사
3. 키 이름 유효성 검증
4. 중복 키 검사

### Task 1.4: 경로 기반 키 순서 수정 (2시간)
**파일**: `internal/schema/schema.go`

**문제**: GetKeyOrder 메소드의 경로 해석 실패
```go
// 테스트 실패: GetKeyOrder("metadata") → ["author", "created"]
// 현재: 빈 배열 반환
```

**해결책**:
1. 경로 파싱 로직 개선
2. 중첩 구조 탐색 수정
3. 배열 인덱스 ([*]) 처리 강화
4. 테스트 케이스 추가

### Task 1.5: 실제 파일 테스트 분리 (1시간)
**파일**: `internal/schema/loader_test.go`

**문제**: 실제 파일 의존성으로 인한 불안정성
```go
// 문제: 실제 파일 경로 참조
yamlPath: "../../testdata/valid/simple.yml"
```

**해결책**:
1. Mock 파일 시스템 사용
2. 테스트 데이터 임베딩
3. 실제 파일 테스트는 별도 분리
4. 유닛 테스트와 통합 테스트 구분

## 테스트 실행 순서

### 1단계: 기본 구조 테스트
```bash
go test ./internal/schema -v -run TestLoadFromBytes
go test ./internal/schema -v -run TestSchemaValidate  
```

### 2단계: 생성 로직 테스트
```bash
go test ./internal/schema -v -run TestGenerateFromYAML
go test ./internal/schema -v -run TestGetKeyOrder
```

### 3단계: 파일 시스템 테스트
```bash
go test ./internal/schema -v -run TestLoaderSaveAndLoad
go test ./internal/schema -v -run TestLoaderListSchemas
```

### 4단계: 전체 테스트
```bash
go test ./internal/schema -v
```

## 예상 소요 시간
- **총 7시간**
- Task 1.1: 2시간 (핵심 로직)
- Task 1.2: 1시간 (테스트 수정)
- Task 1.3: 1시간 (검증 강화)
- Task 1.4: 2시간 (경로 처리)
- Task 1.5: 1시간 (테스트 분리)

## 성공 지표
- [ ] Schema 패키지 테스트 통과율 100%
- [ ] 테스트 실행 시간 10초 이내
- [ ] 외부 파일 의존성 제거
- [ ] Mock을 활용한 격리된 테스트

## 다음 단계
Schema 패키지가 안정화되면 `02-test-data-cleanup.md`로 진행