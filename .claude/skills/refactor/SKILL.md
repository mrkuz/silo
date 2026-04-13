---
name: refactor
description: Refactor the current codebase for simplicity, readability, and consistency while preserving all functionality.
---

## Description

Refactor the codebase.

## General rules

1. Understand intent, constraints, success criteria
2. Ask if anything is unclear - do not guess
3. Track non-trivial steps with tasks

## Skill-specific rules

- Never change functionality — stop and ask if behavior would change
- Do not invent issues — if clean, move on
- Check for uncommitted/staged changes in target files first

## Phase 1: Preparation

1. **Read the code**: Read the target code to understand its structure, functionality, and style
2. **Compile**: Compile the code to identify any errors or warnings
3. **Static analysis**: Run linters to identify potential issues

## Phase 2: Execution

Apply refactoring passes in order:

1. **Dead code**: Eliminate unused functions, variables, constants, and types
2. **Duplicate code**: Eliminate duplication; look for code-reuse opportunities
3. **Simplicity**: Simplify complex or overly nested logic
4. **Consistency**: Ensure consistent error handling, UI, coding style, and naming
5. **Modernization**: Use modern language features and idioms where appropriate
6. **Code smells**: Fix long functions, large classes, magic numbers, etc.
7. **Best practices**: Apply relevant best practices for the language and domain
8. **Comments**: Remove unnecessary or obvious comments
9. **Whitespace**: Clean up unnecessary blank lines and trailing whitespace

## Phase 3: Verification

1. **Static analysis**: Run linters to ensure no new issues were introduced
2. **Compile**: Compile the code to ensure it still builds
3. **Tests**: Run all tests to ensure they still pass
