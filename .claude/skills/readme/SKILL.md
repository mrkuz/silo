---
name: readme
description: Create or update a README for the current project based on its code, CLI help, and configuration.
disable-model-invocation: true
---

## Instructions

Create a README for the project. Target: $ARGUMENTS (or the full project if not specified).

## General rules

- NEVER assume — when in doubt, ask for clarification about the intent
- ALWAYS create a plan before taking action
- Present the plan as a "Summary" section with a bullet list
- NEVER take action without user confirmation of the plan
- Identify other potential improvements that fit the task and present them as an "Optional" section with a bullet list
- Omit the "Optional" section if there are no suggestions
- Optional items require explicit opt-in before applying
- Create a task list and use it to track progress

## Skill-specific rules

- Avoid complex, nested sentences
- If the tool is a CLI tool and provides a help command (like -h, --help), use its output as a starting point for usage documentation
- Clearly mark examples as such
- If a README already exists, read it first and preserve any manually written content that is still accurate

## Preparation

1. **Read the code**: Examine the project files to understand structure, functionality, and style
2. **Build**: Build the project to confirm it compiles successfully
3. **CLI help**: If the project produces a CLI tool, run its help command to capture usage information

## Content

The README should contain the following sections:

1. **Title**
2. **Short description**
3. **Core features**
4. **Quick start guide**
5. **Detailed running instructions** — use CLI help output as a starting point where available
6. **Build and install instructions**
7. **Configuration reference** — include examples where possible
8. **How it works** — detailed explanation of the tool's internals

## Verification

1. **Accuracy**: Ensure all documented commands, flags, and paths match the actual code
2. **Completeness**: Ensure all sections from the Content list are covered
3. **Readability**: Ensure the document reads clearly with no complex, nested sentences
