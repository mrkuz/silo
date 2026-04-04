---
name: refactor
description: Refactor the current codebase for simplicity, readability, and consistency while preserving all functionality.
disable-model-invocation: true
---

## Instructions

Refactor the codebase. Target: $ARGUMENTS (or the full codebase if not specified).

## General rules

- NEVER assume — when in doubt, ask for clarification about the intent
- ALWAYS create a plan before taking action
- Present the plan as a "Summary" section with a bullet list
- NEVER take action without user confirmation of the plan
- Identify other potential improvements that fit the task and present them as an "Optional" section with a bullet list
- Omit the "Optional" section if there are no suggestions
- Optional items require explicit opt-in before applying

## Skill-specific rules

- NEVER change functionality — if behavior would change, stop and ask
- Do NOT invent issues — if the code is already clean, move on
- Check for any uncommitted or staged changes in the target files before starting, to avoid conflicting with in-progress work

## Preparation

1. **Read the code**: Read the target code to understand its structure, functionality, and style
2. **Compile**: Compile the code to identify any errors or warnings
3. **Static analysis**: Run static analysis tools and linters to identify potential issues and areas for improvement

## Steps

Apply the following refactoring passes in order:

1. **Dead code**: Eliminate unused functions, variables, constants, and types
2. **Duplicate code**: Eliminate duplication; look for code-reuse opportunities
3. **Simplicity**: Simplify complex or overly nested logic; prioritize readability above all
4. **Consistent error handling**: Ensure errors are handled consistently (wrapping style, message format, fatal vs. return)
5. **Consistent user interface**: Ensure consistent command arguments, flags, and output format across all commands
6. **Consistent style**: Ensure a consistent coding style across all files
7. **Consistent naming**: Ensure consistent naming across code, comments, configuration, and templates
8. **Comments**: Ensure correctness; remove unnecessary or obvious comments
9. **Whitespace**: Clean up unnecessary blank lines and trailing whitespace

## Verification

1. **Static analysis**: Run static analysis tools and linters to ensure no new issues were introduced
2. **Compile**: Compile the code to ensure it still builds successfully
3. **Tests**: Run all tests to ensure they still pass
