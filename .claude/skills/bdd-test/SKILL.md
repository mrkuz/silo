---
name: bdd-test
description: Create Go tests from BDD feature specifications.
---

## Description

Generate Go test files from Gherkin feature specifications stored in `features/*.spec`.
Each test class is named after the feature and contains nested test functions
representing rules and scenarios.

## Scope

- User-provided scope takes priority (e.g., specific spec file)
- If none provided: create missing tests and update existing ones based on all specs in `features/*.spec`

## General rules

1. Ask if anything is unclear - do not guess
2. Track non-trivial steps with tasks

## Skill-specific rules

- Name test files as `features/<feature_name>_test.go`
- Cover all scenarios from the spec, even if they already have tests
- Use exact text from spec for descriptions
- Use `t.Run` to nest background, rules, and scenarios
- Write unit tests: Call functions directly, do not run the program or its binary.
- Adhere to the conventions provided by the examples
- If there is a discrepancy between the spec and existing tests, ask for clarification

## Phase 1: Preparation

1. **Read the spec file**: Understand feature, rules, and scenarios
2. **Read relevant source code**: Understand the function being tested

## Phase 2: Execution

1. **Create test class**:
    ```go
    func TestFeature<FeatureName>(t *testing.T) { ... }
    ```
2. **Add feature comment directly before test class** (no blank line):
   ```go
   // Feature: <name> — <description>
   func TestFeature<FeatureName>(t *testing.T) {
   ```
3. **Add background comment directly at the beginning of the test class**:
   ```go
   func TestFeature<FeatureName>(t *testing.T) {
       // Background: <background description>
   ```
4. **Add rules and scenarios as nested t.Run**:
   ```go
   t.Run("Rule: <rule name>", func(t *testing.T) {
       t.Run("Scenario: <scenario name>", func(t *testing.T) {
           // Given/When/Then steps
       })
   })
   ```

   Add the comments before the implementation or concrete assertion.

6. **Implement steps** (use comments matching the step text):
   - `Given/And` → setup fixtures, mock responses
   - `When` → call the function under test
   - `Then/And/But` → assertions

## Phase 3: Verification

1. **Build**: Run `go build .` to confirm no compilation errors
2. **Run**: Run `go test ./...` to confirm tests pass
3. **Coverage**: Ensure all scenarios from spec are covered

## Example

Feature file (`features/calculator.spec`):
```gherkin
Feature: Calculator
  Simple arithmetic operations

  Background:
    Given a calculator instance

  Rule: Addition
    Scenario: Adding two positive numbers
      Given the numbers 2 and 3
      And the operation is addition
      When adding
      Then the result is 5
      But the display shows "5"
```

Generated test (`features/calculator_test.go`):
```go
package main

// Feature: Calculator — Simple arithmetic operations
func TestFeatureCalculator(t *testing.T) {
    // Background: a calculator instance

    t.Run("Rule: Addition", func(t *testing.T) {
        t.Run("Scenario: Adding two positive numbers", func(t *testing.T) {
            // Given the numbers 2 and 3
            calc := NewCalculator()
            a, b := 2, 3

            // And the operation is addition
            op := "addition"

            // When adding
            result := calc.Add(a, b)

            // Then the result is 5
            if result != 5 {
                t.Errorf("expected 5, got %d", result)
            }

            // But the display shows "5"
            display := calc.Display()
            if display != "5" {
                t.Errorf("expected display \"5\", got %q", display)
            }
        })
    })
}
```
