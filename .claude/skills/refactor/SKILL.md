---
name: refactor
description: Refactor the current codebase for simplicity, readability, and consistency while preserving all functionality.
---

## Description

Refactor the codebase. Target: $ARGUMENTS (or the full codebase if not specified).

## Skill-specific hard rules (Non-Negotiable)

- NEVER change functionality — if behavior would change, stop and ask
- Do NOT invent issues — if the code is already clean, move on
- Check for any uncommitted or staged changes in the target files before starting, to avoid conflicting with in-progress work

## Preparation

1. **Read the code**: Read the target code to understand its structure, functionality, and style
2. **Compile**: Compile the code to identify any errors or warnings
3. **Static analysis**: Run static analysis tools and linters to identify potential issues and areas for improvement

## Execution

Apply the following refactoring passes in order:

1. **Dead code**: Eliminate unused functions, variables, constants, and types
2. **Duplicate code**: Eliminate duplication; look for code-reuse opportunities
3. **Simplicity**: Simplify complex or overly nested logic; prioritize readability above all
4. **Consistency**: Ensure consistent error handling, user interface, coding style, and naming across code, comments, configuration, and templates
5. **Modernization**: Use modern language features and idioms where appropriate
6. **Code smells**: Check for and fix code smells (e.g. long functions, large classes, magic numbers, etc.)
7. **Best practices**: Apply any relevant best practices for the language and domain
8. **Comments**: Ensure correctness; remove unnecessary or obvious comments
9. **Whitespace**: Clean up unnecessary blank lines and trailing whitespace

## Verification

1. **Static analysis**: Run static analysis tools and linters to ensure no new issues were introduced
2. **Compile**: Compile the code to ensure it still builds successfully
3. **Tests**: Run all tests to ensure they still pass
