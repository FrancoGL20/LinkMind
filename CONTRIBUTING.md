# Contributing to LinkMind

## ⚠️ Development Guidelines & Contribution

### 1. English-Only Policy
**CRITICAL RULE:** This entire codebase, including variable names, function names, comments, logs, and commit messages, **MUST be written 100% in English**. No Spanish is allowed in any file within this repository. 

### 2. Git Workflow & Branching
We follow the **Git Flow** methodology for branch naming:
- `feature/*` - For new functionality.
- `bugfix/*` or `hotfix/*` - For resolving issues.
- `release/*` - For preparation of a new release.

### 3. Conventional Commits
All commits must follow the **Conventional Commits** standard:
```text
<type>(<optional scope>): <description>
```
**Examples:**
- `feat(api): add jwt authentication middleware`
- `fix(db): resolve postgres connection leak`
- `docs: update installation guide`
- `chore(deps): bump go version to 1.23`
