# Formatter 고급 기능 테스트 수정 계획

## 목표
Formatter의 고급 기능(에지 케이스, 특수 문자, 성능 최적화)에 대한 완벽한 테스트 커버리지 달성

## 현재 문제점 분석

### 1. 에지 케이스 처리 부족
```
TestFormatterEdgeCases: 빈 파일, 코멘트만 있는 파일 처리 실패
TestSpecialCharacterHandling: 유니코드, 이모지, 특수 문자 처리 불완전
TestMultiDocumentFormatting: 복합 문서 처리 시 구조 손상
```

### 2. 포맷팅 품질 문제
```
빈 줄 처리 불일치
들여쓰기 설정 미반영
라인 길이 제한 무시
```

### 3. 성능 및 안정성
```
대용량 파일 처리 시 메모리 문제
복잡한 중첩 구조에서 스택 오버플로우 위험
순환 참조 감지 미흡
```

## 수정 작업 목록

### Task 4.1: 에지 케이스 처리 강화 (3시간)

#### 4.1.1 빈 파일 및 최소 구조 처리
**파일**: `internal/formatter/formatter.go`

**처리해야 할 케이스**:
1. 완전히 빈 파일
2. 코멘트만 있는 파일
3. 공백만 있는 파일
4. 하나의 스칼라 값만 있는 파일

**구현 예시**:
```go
func (f *Formatter) FormatContent(content []byte) ([]byte, error) {
    // 빈 파일 처리
    trimmed := bytes.TrimSpace(content)
    if len(trimmed) == 0 {
        return content, nil  // 빈 파일은 그대로 반환
    }
    
    // 코멘트만 있는 파일 검사
    if f.isCommentsOnly(content) {
        return content, nil  // 코멘트만 있으면 그대로 반환
    }
    
    // 기존 포맷팅 로직
    return f.formatNormalContent(content)
}

func (f *Formatter) isCommentsOnly(content []byte) bool {
    lines := strings.Split(string(content), "\n")
    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
            return false
        }
    }
    return true
}
```

#### 4.1.2 특수 YAML 구조 처리
**파일**: `internal/formatter/formatter.go`

**처리할 구조**:
```yaml
# 1. 루트가 배열인 경우
- item1
- item2

# 2. 루트가 스칼라인 경우
"just a string"

# 3. 복합 앵커/얼라이어스
defaults: &defaults
  <<: *other_defaults
  new_key: value
```

**구현**:
```go
func (f *Formatter) formatSingleDocument(content []byte) ([]byte, error) {
    node, err := f.parser.ParseYAML(content)
    if err != nil {
        return nil, fmt.Errorf("failed to parse YAML: %w", err)
    }
    
    // 문서 루트 타입별 처리
    switch node.Kind {
    case yaml.DocumentNode:
        return f.formatDocumentNode(node)
    case yaml.MappingNode:
        return f.formatMappingRoot(node)
    case yaml.SequenceNode:
        return f.formatSequenceRoot(node)
    case yaml.ScalarNode:
        return f.formatScalarRoot(node)
    default:
        return nil, fmt.Errorf("unsupported root node type: %v", node.Kind)
    }
}
```

### Task 4.2: 특수 문자 및 인코딩 처리 (2시간)

#### 4.2.1 유니코드 및 이모지 보존
**파일**: `internal/formatter/writer.go`

**테스트 케이스**:
```yaml
unicode_test:
  korean: "안녕하세요 세계"
  chinese: "你好世界" 
  japanese: "こんにちは世界"
  emoji: "🚀 🎉 ✨ 🌍"
  mixed: "Hello 世界 🌍"
```

**구현 주의사항**:
1. UTF-8 인코딩 보장
2. 이모지 바이트 시퀀스 보존
3. 제어 문자 이스케이프 처리

#### 4.2.2 YAML 특수 문자 처리
**파일**: `internal/formatter/writer.go`

**처리할 문자들**:
```yaml
special_chars:
  quotes: 'He said "Hello" and I said \'Hi\''
  escapes: "Line1\nLine2\tTabbed\r\nWindows"
  yaml_special: "key: value | literal > folded"
  control_chars: "\u0000\u0001\u0007\u0008"
```

**구현**:
```go
func (w *Writer) formatStringValue(value string) string {
    // 특수 문자 감지
    needsQuoting := w.needsQuoting(value)
    if needsQuoting {
        return w.quoteString(value)
    }
    return value
}

func (w *Writer) needsQuoting(value string) bool {
    // YAML 특수 문자 검사
    specialChars := []string{":", "|", ">", "[", "]", "{", "}", "#", "&", "*"}
    for _, char := range specialChars {
        if strings.Contains(value, char) {
            return true
        }
    }
    
    // 제어 문자 검사
    for _, r := range value {
        if r < 32 && r != '\t' && r != '\n' && r != '\r' {
            return true
        }
    }
    
    return false
}
```

### Task 4.3: 포맷팅 품질 개선 (3시간)

#### 4.3.1 스마트 빈 줄 처리
**파일**: `internal/formatter/writer.go`

**빈 줄 규칙**:
1. 최상위 섹션 간에 빈 줄 추가
2. 배열 요소 간에는 빈 줄 없음
3. 중첩 레벨에 따른 빈 줄 조정
4. 코멘트 전후 빈 줄 처리

**구현**:
```go
type LineFormatter struct {
    indent      int
    addBlankLines bool
    currentLevel  int
}

func (lf *LineFormatter) formatLines(content string) string {
    lines := strings.Split(content, "\n")
    var result []string
    
    for i, line := range lines {
        // 현재 라인의 들여쓰기 레벨 계산
        level := lf.getIndentLevel(line)
        
        // 빈 줄 추가 조건 검사
        if lf.shouldAddBlankLine(i, lines, level) {
            result = append(result, "")
        }
        
        result = append(result, line)
        lf.currentLevel = level
    }
    
    return strings.Join(result, "\n")
}
```

#### 4.3.2 들여쓰기 일관성 보장
**파일**: `internal/formatter/writer.go`

**기능**:
1. 설정된 들여쓰기 크기 적용
2. 탭/스페이스 혼용 방지
3. 중첩 레벨별 정확한 들여쓰기
4. 배열 요소 들여쓰기 정렬

#### 4.3.3 라인 길이 관리
**파일**: `internal/formatter/writer.go`

**기능**:
1. 설정된 라인 길이 한계 준수
2. 긴 문자열의 자동 폴딩
3. 배열/객체의 적절한 줄바꿈
4. 코멘트 위치 조정

### Task 4.4: 성능 및 안정성 강화 (2시간)

#### 4.4.1 메모리 효율성 개선
**파일**: `internal/formatter/formatter.go`

**최적화 포인트**:
1. 스트림 기반 처리로 메모리 사용량 감소
2. 불필요한 복사 작업 제거
3. 가비지 컬렉션 압박 감소
4. 버퍼 풀링 활용

**구현**:
```go
type StreamFormatter struct {
    bufferPool sync.Pool
    maxMemory  int64
}

func (sf *StreamFormatter) FormatStream(reader io.Reader, writer io.Writer) error {
    buffer := sf.bufferPool.Get().(*bytes.Buffer)
    defer sf.bufferPool.Put(buffer)
    buffer.Reset()
    
    // 청크 단위로 처리
    chunk := make([]byte, 64*1024) // 64KB 청크
    for {
        n, err := reader.Read(chunk)
        if n > 0 {
            if err := sf.processChunk(chunk[:n], buffer); err != nil {
                return err
            }
        }
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }
    }
    
    _, err := writer.Write(buffer.Bytes())
    return err
}
```

#### 4.4.2 순환 참조 및 무한 루프 방지
**파일**: `internal/formatter/reorder.go`

**안전장치**:
```go
type SafeReorderer struct {
    *Reorderer
    visitedNodes map[*yaml.Node]bool
    maxDepth     int
    currentDepth int
}

func (sr *SafeReorderer) ReorderNode(node *yaml.Node, path string) error {
    // 순환 참조 검사
    if sr.visitedNodes[node] {
        return fmt.Errorf("circular reference detected at path: %s", path)
    }
    
    // 깊이 제한 검사
    if sr.currentDepth > sr.maxDepth {
        return fmt.Errorf("maximum nesting depth exceeded at path: %s", path)
    }
    
    sr.visitedNodes[node] = true
    sr.currentDepth++
    
    defer func() {
        delete(sr.visitedNodes, node)
        sr.currentDepth--
    }()
    
    return sr.Reorderer.ReorderNode(node, path)
}
```

## 테스트 케이스 강화

### 1. 포괄적인 에지 케이스 테스트
**파일**: `internal/formatter/edge_cases_test.go`

```go
func TestEmptyFileHandling(t *testing.T) {
    testCases := []struct {
        name     string
        input    string
        expected string
    }{
        {"completely empty", "", ""},
        {"only whitespace", "   \n  \t\n  ", "   \n  \t\n  "},
        {"only comments", "# just a comment\n# another comment", "# just a comment\n# another comment"},
        {"yaml with only null", "null", "null"},
        {"yaml with only boolean", "true", "true"},
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            formatter := createTestFormatter()
            result, err := formatter.FormatContent([]byte(tc.input))
            assert.NoError(t, err)
            assert.Equal(t, tc.expected, string(result))
        })
    }
}
```

### 2. 성능 테스트
**파일**: `internal/formatter/performance_test.go`

```go
func TestLargeFilePerformance(t *testing.T) {
    // 큰 YAML 파일 생성 (1MB 이상)
    largeYaml := generateLargeYAML(1024 * 1024) // 1MB
    
    formatter := createTestFormatter()
    
    start := time.Now()
    _, err := formatter.FormatContent(largeYaml)
    duration := time.Since(start)
    
    assert.NoError(t, err)
    assert.Less(t, duration, 5*time.Second, "Large file formatting should complete within 5 seconds")
}

func TestMemoryUsage(t *testing.T) {
    var memBefore, memAfter runtime.MemStats
    runtime.GC()
    runtime.ReadMemStats(&memBefore)
    
    // 메모리 집약적인 포맷팅 작업
    formatter := createTestFormatter()
    for i := 0; i < 100; i++ {
        largeYaml := generateLargeYAML(100 * 1024) // 100KB * 100
        _, _ = formatter.FormatContent(largeYaml)
    }
    
    runtime.GC()
    runtime.ReadMemStats(&memAfter)
    
    memIncrease := memAfter.Alloc - memBefore.Alloc
    assert.Less(t, memIncrease, uint64(50*1024*1024), "Memory increase should be less than 50MB")
}
```

## 예상 소요 시간
- **총 10시간**
- Task 4.1: 3시간 (에지 케이스 처리)
- Task 4.2: 2시간 (특수 문자 처리)
- Task 4.3: 3시간 (포맷팅 품질)
- Task 4.4: 2시간 (성능 및 안정성)

## 성공 지표
- [ ] 모든 에지 케이스 테스트 통과
- [ ] 특수 문자 처리 100% 정확성
- [ ] 1MB 파일 처리 시간 5초 이내
- [ ] 메모리 사용량 50MB 이하 유지
- [ ] 순환 참조 감지 및 처리
- [ ] 중첩 깊이 1000레벨까지 안전 처리

## 다음 단계
고급 기능이 완성되면 `05-integration-tests.md`로 진행