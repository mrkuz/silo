---
name: bdd-spec
description: Create BDD feature specifications in Gherkin format for the current project.
---

## Description

Create feature specifications using Gherkin syntax to document user-facing behavior.

## Scope

- User-provided scope takes priority (e.g., specific feature or command)
- If none provided: all features discoverable from README.md
- If CLI tool: CLI commands and arguments

## General rules

1. Ask if anything is unclear - do not guess
2. All phases are mandatory - do not skip without user approval
3. Track non-trivial steps with tasks

## Skill-specific rules

- Use Gherkin format: https://cucumber.io/docs/gherkin/reference
- Store each feature under `features/<snake_case_name>.spec`
- Use `Background` and `Rule` blocks where appropriate
- Focus on user impact and experience, not implementation details

## Phase 1: Preparation

1. **Read README**: Discover features and understand terminology
2. **CLI help**: Discover available commands and arguments
3. **Check source code**: Examine relevant files for step definitions and expected outcomes

## Phase 2: Execution

Write specification files:

1. **One feature per file**: `features/<name>.feature`
2. **Use Gherkin elements appropriately**:
   - `Background` for common preconditions
   - `Rule` for business rules across scenarios
   - `Scenario Outline` with `Examples` for parametrized cases
3. **Cover comprehensively**:
   - All commands, arguments, and flag combinations
   - Mutually exclusive flags
   - Error handling
   - Optional arguments
   - Force flags (`--force`, `--yes`, etc.)
4. **Step definitions**:
   - Use `Given` for preconditions
   - Use `When` for actions
   - Use `Then` for expected outcomes
   - Use `And` / `But` for readability

## Phase 3: Verification

1. **Syntax**: Validate all feature files are syntactically correct
2. **Coverage**: Ensure all commands and argument combinations are covered
3. **No duplication**: Avoid duplicate scenarios
4. **Meaningful names**: Feature and scenario names clearly describe user impact
