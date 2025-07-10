# Test Data Directory Structure

This directory contains comprehensive test data for the YAML formatter. The test data is organized into categories to facilitate different types of testing.

## Directory Structure

### `/valid/`
Contains valid YAML files that should parse successfully:
- `simple.yml` - Basic valid YAML structure
- `complex-nested.yml` - Deeply nested structures
- `with-comments.yml` - YAML with various comment styles
- `anchors-and-aliases.yml` - YAML anchors and alias references

### `/invalid/`
Contains invalid YAML files that should fail parsing:
- `bad-indentation.yml` - Incorrect indentation
- `duplicate-keys.yml` - Duplicate keys in mappings
- `unclosed-quote.yml` - Unclosed string quotes
- `invalid-anchor.yml` - References to non-existent anchors
- `mixed-types.yml` - Invalid mixing of sequences and mappings

### `/edge-cases/`
Contains edge cases and boundary conditions:
- `empty.yml` - Empty file
- `only-comments.yml` - File with only comments
- `special-characters.yml` - Unicode, emoji, and special characters
- `long-lines.yml` - Very long lines and text blocks
- `very-deep-nesting.yml` - Extremely nested structures

### `/formatting/`
Contains before/after pairs for formatting tests:
- `/input/` - Unformatted YAML files
- `/expected/` - Expected output after formatting

### `/schema-validation/`
Contains files for schema validation testing:
- `test.schema.yaml` - Sample schema definition
- `matches-schema.yml` - File that matches the schema
- `extra-fields.yml` - File with fields not in schema

### `/multi-document/`
Contains multi-document YAML files:
- `simple-multi.yml` - Simple multi-document file
- `kubernetes-multi.yml` - Kubernetes resources as multi-doc
- `mixed-content.yml` - Different types in each document

## Usage

These test files can be used for:
1. Unit testing the parser
2. Integration testing the formatter
3. Schema validation testing
4. Edge case handling verification
5. Performance testing with various file sizes
6. Comment preservation testing