---
name: feature
description: End-to-end workflow to implement a new feature: clarify, spec, plan, implement, test, refactor, docs.
---

## Description

A comprehensive workflow for implementing a new feature from idea to completion.

## General rules

1. Ask if anything is unclear - do not guess
2. All phases are mandatory - do not skip without user approval
3. Track non-trivial steps with tasks

## Skill-specific rules

- Once the feature specification is confirmed, do not modify without user approval

## Phase 1: Preparation

- Each step ends with user review and confirmation before proceeding
- If user rejects at any step, iterate on that step until confirmed
- Resume from the appropriate step after confirmation

### 1.1: Clarify

1. Ask clarifying questions about any ambiguous or missing details in the feature description
2. Present the user a polished, complete description of the feature
3. Ask user to confirm or refine the description
4. If confirmed, proceed to 1.2

### 1.2: Feature Specification

1. Use `/bdd-spec` to update existing or create a new feature specification
2. Present the specs to the user for review
3. Ask user to confirm or refine the specs
4. If confirmed, proceed to 1.3

### 1.3: Implementation Planning

1. Use Plan agent to create an implementation plan
2. Present the plan to the user for review
3. Ask user to confirm or refine the plan
4. If confirmed, proceed to Phase 2

## Phase 2: Execution

### 2.1: Implementation

1. Implement the feature according to the approved plan
2. Verify the implementation compiles and passes existing tests
3. Proceed to 2.2

### 2.2: Testing

1. Use `/bdd-test` to create feature tests
2. Run tests and fix any failures
3. Proceed to 2.3

### 2.3: Refactor

1. Use `/refactor` to review and improve the implementation
2. Apply any recommended refactoring
3. Proceed to 2.4

### 2.4: Documentation

1. Use `/readme` to update documentation if applicable
2. Verify README reflects the new feature
3. Proceed to phase 3

## Phase 3: Verification

1. **Consistency**: Verify implementation matches the specification
