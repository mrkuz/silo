---
name: plex
description: Plan and execute tasks by selecting the best fittings skill and agent.
---

## When to Use

Invoke with the `/plex` slash command.

---

## Overview

Analyze the task, produce a plan, and execute it using the most appropriate skills or agents. Free-form arguments after the skill name are available as `$ARGUMENTS`.

---

## Hard Rules (Non-Negotiable)

1. **Plan before acting.** Produce a plan and get user confirmation before editing files, running commands, or spawning agents.
2. **Track every non-trivial step.** Use tasks with clear subjects and statuses.
3. **Follow process strictly.** Execute each phase and adhere to the defined process.
4. **Ask when unclear.** Resolve ambiguity with the user — do not guess.
5. **Follow the output formats.** Required structures in each phase are mandatory, not suggestions.

---

## Phase 1: Plan (Always First)

### Process

1. **Understand the request.** Identify intent, constraints, and success criteria.
2. **Ask immediately** about anything unclear — do not invent workarounds.
3. **Select skills.** Fetch all available skills and pick the most appropriate ones by name and description.
4. **Inherit skills' abilities.** Use each skill's rules and guidance in planning and execution. Don't invoke them directly.
5. **Select an agent.** Fetch all available agents and choose the best-fit agent (e.g. domain-specific) for execution.
6. **Read relevant context.** Files, code, or other context inform the plan.
7. **List atomic steps.** Each step should be a single, reversible action where possible.
8. **Write the plan** using the format below.

### Required Output Format

```
## Plan

- 1: description
- 2: description
...

## Summary

- Bullet points
- What will be done and why
- Approach rationale

**Execution agent**: agent name, or "None" if executing directly
**Skills**: skill names used, or "None"
```

The Summary section is mandatory. A plan without it is incomplete.

---

## Phase 2: Execute

The selected agent carries out the plan.

### Process

1. **Track progress.** Mark tasks in_progress before starting, completed when done.
2. **Collect learnings** for the post-execution summary:
   - About the task: follow-up work discovered but out of scope
   - About incorporated skills: unclear rules, missing guidance, friction points

---

## Phase 3: Post-Execution

### Exit Criteria

Plex is complete when:
1. All planned steps are resolved
2. Post-execution summary is delivered
3. User acknowledges or gives a follow-up instruction

### Required Output Format

```
## Summary

- 1: outcome description
- 2: outcome description
...

## Follow-up (omit if empty)

- Potential next steps

## Skill Improvements (omit if empty)

- Friction points or gaps found in the incorporated skills
```
