# Formatter 핵심 기능 테스트 수정 계획

## 목표
Formatter 패키지의 핵심 기능(파싱, 리오더링, 출력)에 대한 안정적인 테스트 구축

## 현재 문제점 분석

### 1. Parser 테스트 문제
```
TestParseValidYAML: 일부 valid 파일에서 파싱 실패
TestParseInvalidYAML: invalid 파일이 예상과 달리 파싱 성공
TestCommentPreservation: 코멘트 보존 검증 로직 불완전
```

### 2. Reorder 테스트 문제
```
TestReorderNode: 키 순서 변경 로직 실패
TestCheckOrder: 순서 검증 로직 불일치
TestReorderWithWildcards: 배열 요소 처리 실패
```

### 3. Writer 테스트 문제
```
TestFormatToString: 출력 형식 불일치
TestFormatWithComments: 코멘트 포함 출력 문제
TestIndentSettings: 들여쓰기 설정 미반영
```

## 수정 작업 목록

### Task 3.1: Parser 모듈 강화 (3시간)

#### 3.1.1 기본 파싱 로직 검증
**파일**: `internal/formatter/parser.go`, `parser_test.go`

**문제**: NewParser, ParseYAML 등 기본 메소드 미구현
```go
// 테스트 실패
parser := NewParser(true)  // undefined: NewParser
node, err := parser.ParseYAML(content)  // undefined method
```

**해결책**:
1. Parser 구조체 및 기본 메소드 구현
2. YAML 파싱 래퍼 함수 작성
3. 에러 핸들링 강화

**구현 예시**:
```go
type Parser struct {
    preserveComments bool
}

func NewParser(preserveComments bool) *Parser {
    return &Parser{preserveComments: preserveComments}
}

func (p *Parser) ParseYAML(content []byte) (*yaml.Node, error) {
    var node yaml.Node
    err := yaml.Unmarshal(content, &node)
    if err != nil {
        return nil, fmt.Errorf("failed to parse YAML: %w", err)
    }
    return &node, nil
}

func (p *Parser) ValidateYAML(content []byte) error {
    var temp interface{}
    return yaml.Unmarshal(content, &temp)
}
```

#### 3.1.2 멀티 도큐먼트 처리
**파일**: `internal/formatter/parser.go`

**기능 추가**:
```go
func (p *Parser) IsMultiDocument(content []byte) bool
func (p *Parser) ParseMultiDocument(content []byte) ([]*yaml.Node, error)
func (p *Parser) SetPreserveComments(preserve bool)
func (p *Parser) PreserveComments() bool
```

#### 3.1.3 코멘트 보존 검증
**파일**: `internal/formatter/parser_test.go`

**개선할 테스트**:
```go
func TestCommentPreservation(t *testing.T) {
    content := `# Header comment
name: test  # Inline comment
# Footer comment
version: 1.0`
    
    parser := NewParser(true)
    node, err := parser.ParseYAML(content)
    
    // 구체적인 코멘트 검증 로직 구현
    assert.Contains(t, node.HeadComment, "Header comment")
    assert.Contains(t, node.Content[1].LineComment, "Inline comment")
}
```

### Task 3.2: Reorder 모듈 완성 (4시간)

#### 3.2.1 Reorderer 구조체 구현
**파일**: `internal/formatter/reorder.go`

**필요 구현**:
```go
type Reorderer struct {
    schema *schema.Schema
    parser *Parser
}

func NewReorderer(s *schema.Schema, p *Parser) *Reorderer {
    return &Reorderer{schema: s, parser: p}
}

func (r *Reorderer) ReorderNode(node *yaml.Node, path string) error {
    // 실제 리오더링 로직 구현
}

func (r *Reorderer) CheckOrder(node *yaml.Node, path string) (bool, error) {
    // 순서 검증 로직 구현
}
```

#### 3.2.2 키 순서 변경 알고리즘
**파일**: `internal/formatter/reorder.go`

**핵심 로직**:
1. 현재 노드의 키 순서 추출
2. 스키마에서 정의된 순서 가져오기
3. 스키마 순서에 따라 노드 재정렬
4. 스키마에 없는 키는 마지막에 배치

**구현 예시**:
```go
func (r *Reorderer) reorderMapping(node *yaml.Node, path string) error {
    if node.Kind != yaml.MappingNode {
        return nil
    }
    
    // 현재 키-값 쌍 추출
    keyValuePairs := make(map[string]*yaml.Node)
    for i := 0; i < len(node.Content); i += 2 {
        key := node.Content[i].Value
        value := node.Content[i+1]
        keyValuePairs[key] = value
    }
    
    // 스키마 순서 가져오기
    expectedOrder := r.schema.GetKeyOrder(path)
    
    // 새로운 순서로 재배치
    newContent := make([]*yaml.Node, 0, len(node.Content))
    
    // 스키마에 정의된 순서대로 배치
    for _, key := range expectedOrder {
        if keyNode, exists := keyValuePairs[key]; exists {
            newContent = append(newContent, &yaml.Node{Value: key}, keyNode)
            delete(keyValuePairs, key)
        }
    }
    
    // 남은 키들 추가
    for key, value := range keyValuePairs {
        newContent = append(newContent, &yaml.Node{Value: key}, value)
    }
    
    node.Content = newContent
    return nil
}
```

#### 3.2.3 배열 요소 처리
**파일**: `internal/formatter/reorder.go`

**와일드카드 패턴 처리**:
```go
func (r *Reorderer) reorderSequence(node *yaml.Node, path string) error {
    if node.Kind != yaml.SequenceNode {
        return nil
    }
    
    for i, item := range node.Content {
        itemPath := fmt.Sprintf("%s[%d]", path, i)
        if err := r.ReorderNode(item, itemPath); err != nil {
            return err
        }
    }
    return nil
}
```

### Task 3.3: Writer 모듈 구현 (2시간)

#### 3.3.1 Writer 기본 구조
**파일**: `internal/formatter/writer.go`

**필요 구현**:
```go
type Writer struct {
    indent           int
    lineWidth        int
    preserveComments bool
}

func NewWriter() *Writer {
    return &Writer{
        indent:           2,
        lineWidth:        80,
        preserveComments: true,
    }
}

func (w *Writer) FormatToString(node *yaml.Node) (string, error) {
    encoder := yaml.NewEncoder(&strings.Builder{})
    encoder.SetIndent(w.indent)
    
    if err := encoder.Encode(node); err != nil {
        return "", err
    }
    
    return encoder.String(), nil
}
```

#### 3.3.2 설정 메소드들
**파일**: `internal/formatter/writer.go`

**구현 필요**:
```go
func (w *Writer) SetIndent(indent int)
func (w *Writer) SetLineWidth(width int)
func (w *Writer) SetPreserveComments(preserve bool)
func (w *Writer) GetIndent() int
func (w *Writer) GetLineWidth() int
func (w *Writer) ValidateFormattedOutput(content []byte) error
func (w *Writer) FormatNodesToString(nodes []*yaml.Node) (string, error)
```

#### 3.3.3 통계 및 유틸리티
**파일**: `internal/formatter/writer.go`

**FormatStats 구조체**:
```go
type FormatStats struct {
    OriginalLines   int
    FormattedLines  int
    OriginalBytes   int
    FormattedBytes  int
    LinesChanged    int
    KeysReordered   int
}

func (w *Writer) CalculateStats(original, formatted []byte) *FormatStats {
    // 통계 계산 로직 구현
}
```

### Task 3.4: 통합 테스트 보완 (1시간)

#### 3.4.1 전체 플로우 테스트
**파일**: `internal/formatter/formatter_test.go`

**테스트 케이스**:
1. 기본 포맷팅 플로우
2. 코멘트 보존 플로우
3. 멀티 도큐먼트 플로우
4. 에러 핸들링 플로우

#### 3.4.2 성능 테스트
**파일**: `internal/formatter/formatter_bench_test.go`

**벤치마크 테스트**:
```go
func BenchmarkFormatSmallFile(b *testing.B) { /* ... */ }
func BenchmarkFormatLargeFile(b *testing.B) { /* ... */ }
func BenchmarkFormatMultiDocument(b *testing.B) { /* ... */ }
```

## 구현 순서

### 1단계: Parser 기본 구현
```bash
# Parser 구조체 및 기본 메소드 구현
go test ./internal/formatter -v -run TestParseValidYAML
```

### 2단계: Reorderer 구현
```bash
# Reorderer 구조체 및 리오더링 로직 구현
go test ./internal/formatter -v -run TestReorderNode
```

### 3단계: Writer 구현
```bash
# Writer 구조체 및 출력 로직 구현
go test ./internal/formatter -v -run TestFormatToString
```

### 4단계: 통합 테스트
```bash
# 전체 formatter 테스트
go test ./internal/formatter -v
```

## Mock 데이터 활용

### 경량 테스트 데이터
```go
var testData = map[string]string{
    "simple": `name: test
version: 1.0
description: A test`,
    
    "nested": `app:
  name: test
  config:
    debug: true
    port: 8080`,
    
    "array": `items:
  - name: item1
    value: 100
  - name: item2
    value: 200`,
}

func getTestData(key string) []byte {
    return []byte(testData[key])
}
```

### 스키마 Mock
```go
func createTestSchema() *schema.Schema {
    return &schema.Schema{
        Name: "test",
        Keys: map[string]interface{}{
            "name":        nil,
            "version":     nil,
            "description": nil,
            "app": map[string]interface{}{
                "name":   nil,
                "config": map[string]interface{}{
                    "debug": nil,
                    "port":  nil,
                },
            },
        },
        Order: []string{
            "name", "version", "description",
            "app", "app.name", "app.config",
            "app.config.debug", "app.config.port",
        },
    }
}
```

## 예상 소요 시간
- **총 10시간**
- Task 3.1: 3시간 (Parser 구현)
- Task 3.2: 4시간 (Reorderer 구현)
- Task 3.3: 2시간 (Writer 구현)
- Task 3.4: 1시간 (통합 테스트)

## 성공 지표
- [ ] Parser 테스트 통과율 100%
- [ ] Reorderer 테스트 통과율 100%
- [ ] Writer 테스트 통과율 100%
- [ ] 코멘트 보존 기능 정상 동작
- [ ] 멀티 도큐먼트 처리 정상 동작

## 다음 단계
핵심 기능이 안정화되면 `04-formatter-advanced.md`로 진행