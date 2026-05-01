---
name: readme
description: Create or update a README for the current project based on its code, CLI help, and configuration.
---

## Description

Create or update a README for the project.

## Scope

- User-provided scope takes priority (e.g., section or topic)
- If none provided, update the entire README

## General rules

1. Ask if anything is unclear - do not guess
2. All phases are mandatory - do not skip without user approval
3. Track non-trivial steps with tasks

## Skill-specific rules

- Avoid verbose, complex, nested sentences
- Minimize implementation details in all sections **except** "How It Works"
- Mark examples clearly

## Phase 1: Preparation

1. **Read the code**: Examine project files to understand structure, functionality, and style
2. **Build**: Build the project to confirm it compiles
3. **CLI help**: If the project produces a CLI tool, run its help command

## Phase 2: Execution

The README should contain:

1. **Title**
2. **Short description**
3. **Goals and Non-Goals**
4. **Core features**
5. **Quick start guide**
6. **Detailed running instructions** — use CLI help output where available
7. **Build and install instructions**
8. **Configuration reference** — include examples
9. **How it works** — explanation of the tool's internals

## Phase 3: Verification

1. **Accuracy**: Ensure documented commands, flags, and paths match the actual code
2. **Completeness**: Ensure all sections are covered
3. **Readability**: Avoid complex sentences
