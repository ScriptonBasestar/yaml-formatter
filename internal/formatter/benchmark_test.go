package formatter

import (
	"strings"
	"testing"
)

// Sample YAML data for benchmarking
var benchmarkYAMLData = []struct {
	name string
	data string
}{
	{
		name: "simple",
		data: `
name: test
version: 1.0
dependencies:
  - package1
  - package2
`,
	},
	{
		name: "complex",
		data: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: default
  labels:
    app: nginx
    version: "1.0"
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.21
        ports:
        - containerPort: 80
        env:
        - name: ENV_VAR
          value: "test"
        resources:
          requests:
            memory: "64Mi"
            cpu: "250m"
          limits:
            memory: "128Mi"
            cpu: "500m"
`,
	},
	{
		name: "large",
		data: generateLargeYAML(),
	},
	{
		name: "nested",
		data: `
level1:
  level2:
    level3:
      level4:
        level5:
          data: "deeply nested"
          array:
            - item1
            - item2
            - item3
          object:
            key1: value1
            key2: value2
            key3:
              nested_key: nested_value
              another_array:
                - element1
                - element2
                - element3
`,
	},
}

// generateLargeYAML creates a large YAML document for benchmarking
func generateLargeYAML() string {
	var builder strings.Builder
	builder.WriteString("large_config:\n")
	builder.WriteString("  services:\n")
	
	// Generate 100 services
	for i := 0; i < 100; i++ {
		builder.WriteString("    service_")
		builder.WriteString(strings.Repeat("0", 3-len(string(rune(i))))[:3-len(string(rune(i)))])
		builder.WriteString(string(rune(i)))
		builder.WriteString(":\n")
		builder.WriteString("      name: service-")
		builder.WriteString(string(rune(i)))
		builder.WriteString("\n")
		builder.WriteString("      port: ")
		builder.WriteString(string(rune(8000 + i)))
		builder.WriteString("\n")
		builder.WriteString("      enabled: true\n")
		builder.WriteString("      config:\n")
		builder.WriteString("        timeout: 30\n")
		builder.WriteString("        retries: 3\n")
		builder.WriteString("        endpoints:\n")
		for j := 0; j < 5; j++ {
			builder.WriteString("          - /api/v")
			builder.WriteString(string(rune(j + 1)))
			builder.WriteString("/health\n")
		}
	}
	
	return builder.String()
}

// BenchmarkFormatter_Format benchmarks the main Format function
func BenchmarkFormatter_Format(b *testing.B) {
	formatter := New()
	
	for _, testCase := range benchmarkYAMLData {
		b.Run(testCase.name, func(b *testing.B) {
			data := []byte(testCase.data)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				_, err := formatter.Format(data)
				if err != nil {
					b.Fatalf("Format failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkFormatter_FormatWithOptions benchmarks formatting with different options
func BenchmarkFormatter_FormatWithOptions(b *testing.B) {
	testData := []byte(benchmarkYAMLData[1].data) // Use complex data
	
	testCases := []struct {
		name    string
		options Options
	}{
		{
			name: "default",
			options: Options{
				Indent:      2,
				LineWidth:   80,
				SortKeys:    false,
				SortArrays:  false,
				TrimSpaces:  true,
				DoubleQuote: false,
			},
		},
		{
			name: "sort_keys",
			options: Options{
				Indent:      2,
				LineWidth:   80,
				SortKeys:    true,
				SortArrays:  false,
				TrimSpaces:  true,
				DoubleQuote: false,
			},
		},
		{
			name: "sort_all",
			options: Options{
				Indent:      2,
				LineWidth:   80,
				SortKeys:    true,
				SortArrays:  true,
				TrimSpaces:  true,
				DoubleQuote: false,
			},
		},
		{
			name: "wide_lines",
			options: Options{
				Indent:      4,
				LineWidth:   120,
				SortKeys:    false,
				SortArrays:  false,
				TrimSpaces:  true,
				DoubleQuote: true,
			},
		},
	}
	
	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			formatter := NewWithOptions(tc.options)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				_, err := formatter.Format(testData)
				if err != nil {
					b.Fatalf("Format failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkFormatter_Parse benchmarks the parsing phase
func BenchmarkFormatter_Parse(b *testing.B) {
	formatter := New()
	
	for _, testCase := range benchmarkYAMLData {
		b.Run(testCase.name, func(b *testing.B) {
			data := []byte(testCase.data)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				var yamlData interface{}
				err := formatter.parser.Parse(data, &yamlData)
				if err != nil {
					b.Fatalf("Parse failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkFormatter_Write benchmarks the writing phase
func BenchmarkFormatter_Write(b *testing.B) {
	formatter := New()
	
	// Pre-parse the data for writing benchmarks
	parsedData := make([]interface{}, len(benchmarkYAMLData))
	for i, testCase := range benchmarkYAMLData {
		data := []byte(testCase.data)
		err := formatter.parser.Parse(data, &parsedData[i])
		if err != nil {
			b.Fatalf("Failed to parse test data: %v", err)
		}
	}
	
	for i, testCase := range benchmarkYAMLData {
		b.Run(testCase.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			
			for j := 0; j < b.N; j++ {
				_, err := formatter.writer.Write(parsedData[i])
				if err != nil {
					b.Fatalf("Write failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkFormatter_Reorder benchmarks the reordering functionality
func BenchmarkFormatter_Reorder(b *testing.B) {
	formatter := New()
	formatter.options.SortKeys = true
	
	// Pre-parse the data
	parsedData := make([]interface{}, len(benchmarkYAMLData))
	for i, testCase := range benchmarkYAMLData {
		data := []byte(testCase.data)
		err := formatter.parser.Parse(data, &parsedData[i])
		if err != nil {
			b.Fatalf("Failed to parse test data: %v", err)
		}
	}
	
	for i, testCase := range benchmarkYAMLData {
		b.Run(testCase.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			
			for j := 0; j < b.N; j++ {
				reordered := formatter.reorderer.Reorder(parsedData[i])
				if reordered == nil {
					b.Fatal("Reorder returned nil")
				}
			}
		})
	}
}

// BenchmarkFormatter_Memory benchmarks memory usage
func BenchmarkFormatter_Memory(b *testing.B) {
	formatter := New()
	data := []byte(benchmarkYAMLData[2].data) // Use large data
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		formatted, err := formatter.Format(data)
		if err != nil {
			b.Fatalf("Format failed: %v", err)
		}
		// Ensure the result is used to prevent optimization
		_ = len(formatted)
	}
}

// BenchmarkFormatter_Parallel benchmarks parallel execution
func BenchmarkFormatter_Parallel(b *testing.B) {
	data := []byte(benchmarkYAMLData[1].data) // Use complex data
	
	b.ResetTimer()
	b.ReportAllocs()
	
	b.RunParallel(func(pb *testing.PB) {
		formatter := New() // Each goroutine gets its own formatter
		for pb.Next() {
			_, err := formatter.Format(data)
			if err != nil {
				b.Fatalf("Format failed: %v", err)
			}
		}
	})
}

// BenchmarkFormatter_FileOperations benchmarks file-like operations
func BenchmarkFormatter_FileOperations(b *testing.B) {
	formatter := New()
	
	testCases := []struct {
		name string
		size string
		data []byte
	}{
		{"small_file", "1KB", []byte(strings.Repeat(benchmarkYAMLData[0].data, 10))},
		{"medium_file", "10KB", []byte(strings.Repeat(benchmarkYAMLData[1].data, 10))},
		{"large_file", "100KB", []byte(strings.Repeat(benchmarkYAMLData[2].data, 5))},
	}
	
	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.SetBytes(int64(len(tc.data)))
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				_, err := formatter.Format(tc.data)
				if err != nil {
					b.Fatalf("Format failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkFormatter_EdgeCases benchmarks edge cases and special scenarios
func BenchmarkFormatter_EdgeCases(b *testing.B) {
	formatter := New()
	
	edgeCases := []struct {
		name string
		data string
	}{
		{
			name: "empty_yaml",
			data: "",
		},
		{
			name: "only_comments",
			data: "# This is a comment\n# Another comment\n",
		},
		{
			name: "mixed_types",
			data: `
string_value: "hello"
int_value: 42
float_value: 3.14
bool_value: true
null_value: null
array_value: [1, 2, 3]
object_value: {key: value}
`,
		},
		{
			name: "unicode",
			data: `
unicode_string: "Hello 世界 🌍"
unicode_key_你好: "value"
emoji_array:
  - "😀"
  - "🚀"
  - "🎉"
`,
		},
		{
			name: "long_lines",
			data: `very_long_key_that_exceeds_normal_line_width_and_should_be_handled_properly: "This is a very long value that also exceeds the normal line width and should be handled according to the formatter options"`,
		},
	}
	
	for _, tc := range edgeCases {
		b.Run(tc.name, func(b *testing.B) {
			data := []byte(tc.data)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				_, err := formatter.Format(data)
				if err != nil {
					b.Fatalf("Format failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkFormatter_Stress tests performance under stress
func BenchmarkFormatter_Stress(b *testing.B) {
	formatter := New()
	
	// Create stress test data - very large and complex
	var builder strings.Builder
	builder.WriteString("stress_test:\n")
	
	// Generate deeply nested structure
	for level := 0; level < 10; level++ {
		builder.WriteString(strings.Repeat("  ", level+1))
		builder.WriteString("level_")
		builder.WriteString(string(rune(level)))
		builder.WriteString(":\n")
		
		// Add array at each level
		builder.WriteString(strings.Repeat("  ", level+2))
		builder.WriteString("items:\n")
		for item := 0; item < 20; item++ {
			builder.WriteString(strings.Repeat("  ", level+3))
			builder.WriteString("- name: item_")
			builder.WriteString(string(rune(item)))
			builder.WriteString("\n")
			builder.WriteString(strings.Repeat("  ", level+4))
			builder.WriteString("value: ")
			builder.WriteString(string(rune(item * 100)))
			builder.WriteString("\n")
		}
	}
	
	stressData := []byte(builder.String())
	
	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(stressData)))
	
	for i := 0; i < b.N; i++ {
		_, err := formatter.Format(stressData)
		if err != nil {
			b.Fatalf("Format failed: %v", err)
		}
	}
}

// BenchmarkFormatter_CompareOptions compares performance of different option combinations
func BenchmarkFormatter_CompareOptions(b *testing.B) {
	data := []byte(benchmarkYAMLData[2].data) // Use large data
	
	optionSets := []struct {
		name    string
		options Options
	}{
		{
			name: "fastest",
			options: Options{
				Indent:      2,
				LineWidth:   120,
				SortKeys:    false,
				SortArrays:  false,
				TrimSpaces:  false,
				DoubleQuote: false,
			},
		},
		{
			name: "balanced",
			options: Options{
				Indent:      2,
				LineWidth:   80,
				SortKeys:    false,
				SortArrays:  false,
				TrimSpaces:  true,
				DoubleQuote: false,
			},
		},
		{
			name: "thorough",
			options: Options{
				Indent:      2,
				LineWidth:   80,
				SortKeys:    true,
				SortArrays:  true,
				TrimSpaces:  true,
				DoubleQuote: true,
			},
		},
	}
	
	for _, optSet := range optionSets {
		b.Run(optSet.name, func(b *testing.B) {
			formatter := NewWithOptions(optSet.options)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				_, err := formatter.Format(data)
				if err != nil {
					b.Fatalf("Format failed: %v", err)
				}
			}
		})
	}
}