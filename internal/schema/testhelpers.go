package schema

// TestHelpers provides utilities for creating test schemas in a consistent way
// These functions are meant to be used in test files across the project

// CreateTestSchema creates a schema with the given name and key structure
func CreateTestSchema(name string, keys []string) *Schema {
	schema := &Schema{
		Name:  name,
		Keys:  make(map[string]interface{}),
		Order: make([]string, len(keys)),
	}

	// Set up the keys and order
	copy(schema.Order, keys)
	for _, key := range keys {
		schema.Keys[key] = nil
	}

	return schema
}

// CreateNestedTestSchema creates a schema with nested structure
func CreateNestedTestSchema(name string) *Schema {
	return &Schema{
		Name: name,
		Keys: map[string]interface{}{
			"name":    nil,
			"version": nil,
			"metadata": map[string]interface{}{
				"author":  nil,
				"created": nil,
			},
			"items": map[string]interface{}{
				"name":  nil,
				"value": nil,
			},
		},
		Order: []string{
			"name",
			"version",
			"metadata",
			"metadata.author",
			"metadata.created",
			"items",
			"items[*].name",
			"items[*].value",
		},
	}
}

// CreateDockerComposeTestSchema creates a schema for Docker Compose files
func CreateDockerComposeTestSchema() *Schema {
	return &Schema{
		Name: "compose",
		Keys: map[string]interface{}{
			"version": nil,
			"services": map[string]interface{}{
				"image":       nil,
				"depends_on":  nil,
				"ports":       nil,
				"volumes":     nil,
				"environment": nil,
			},
			"volumes": nil,
		},
		Order: []string{
			"version",
			"services",
			"services[*].image",
			"services[*].depends_on",
			"services[*].ports",
			"services[*].volumes",
			"services[*].environment",
			"volumes",
		},
	}
}

// CreateKubernetesTestSchema creates a schema for Kubernetes resources
func CreateKubernetesTestSchema() *Schema {
	return &Schema{
		Name: "k8s",
		Keys: map[string]interface{}{
			"apiVersion": nil,
			"kind":       nil,
			"metadata": map[string]interface{}{
				"name":      nil,
				"namespace": nil,
				"labels":    nil,
			},
			"spec": map[string]interface{}{
				"containers": map[string]interface{}{
					"name":  nil,
					"image": nil,
					"ports": nil,
				},
			},
		},
		Order: []string{
			"apiVersion",
			"kind",
			"metadata",
			"metadata.name",
			"metadata.namespace",
			"metadata.labels",
			"spec",
			"spec.containers",
			"spec.containers[*].name",
			"spec.containers[*].image",
			"spec.containers[*].ports",
		},
	}
}

// CreateMinimalTestSchema creates a minimal schema for basic testing
func CreateMinimalTestSchema() *Schema {
	return &Schema{
		Name: "minimal",
		Keys: map[string]interface{}{
			"name":        nil,
			"version":     nil,
			"description": nil,
		},
		Order: []string{
			"name",
			"version",
			"description",
		},
	}
}
