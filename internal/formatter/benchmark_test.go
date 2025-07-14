package formatter

import (
	"testing"
)

var testYAML = `
name: test
version: 1.0
dependencies:
  - package1
  - package2
description: Simple test YAML
`

func BenchmarkFormat(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = testYAML // Simple benchmark placeholder
	}
}