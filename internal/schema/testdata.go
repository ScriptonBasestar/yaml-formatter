package schema

import "embed"

// Embedded test data for schema testing
//
//go:embed testdata/*.yml
var TestDataFS embed.FS

// TestData contains sample YAML data for testing schemas
var TestData = map[string]string{
	"docker-compose": `version: '3.8'
services:
  web:
    ports:
      - "3000:3000"
    environment:
      NODE_ENV: production
    image: myapp:latest
    depends_on:
      - db
      - redis
    volumes:
      - ./data:/app/data
  db:
    volumes:
      - postgres_data:/var/lib/postgresql/data
    environment:
      POSTGRES_PASSWORD: secret
    image: postgres:14
  redis:
    image: redis:7
volumes:
  postgres_data:`,

	"kubernetes": `apiVersion: v1
kind: Pod
metadata:
  labels:
    app: test
  name: test-pod
  namespace: default
spec:
  containers:
    - name: web
      image: nginx:latest
      ports:
        - containerPort: 80`,

	"nested-complex": `name: test-app
version: 1.0.0
metadata:
  created: 2024-01-01
  author: tester
items:
  - name: item1
    value: 100
  - name: item2
    value: 200`,

	"minimal": `name: test
version: 1.0
description: A test schema`,

	"empty": ``,

	"comments-only": `# This is a comment
# Another comment
# Yet another comment`,

	"special-chars": `name: "Hello 世界 🌍"
unicode: "안녕하세요"
emoji: "🚀 🎉 ✨"
escaped: "Line1\nLine2\tTabbed"`,

	"very-nested": `level1:
  level2:
    level3:
      level4:
        level5:
          deep_value: found`,

	"array-of-objects": `services:
  - name: api
    image: api:latest
    ports:
      - 8080
      - 8081
    config:
      debug: true
  - name: db
    image: postgres:14
    environment:
      POSTGRES_DB: mydb`,
}

// GetTestData returns the test data for a given key
func GetTestData(key string) []byte {
	if data, exists := TestData[key]; exists {
		return []byte(data)
	}
	return nil
}

// GetFormattedTestData returns test data formatted according to the given schema
func GetFormattedTestData(key, schemaType string) (input, expected []byte) {
	input = GetTestData(key)
	if input == nil {
		return nil, nil
	}

	// For now, return the same data as both input and expected
	// In a real implementation, this would apply the schema formatting
	expected = input
	return input, expected
}

// ListTestDataKeys returns all available test data keys
func ListTestDataKeys() []string {
	keys := make([]string, 0, len(TestData))
	for k := range TestData {
		keys = append(keys, k)
	}
	return keys
}
