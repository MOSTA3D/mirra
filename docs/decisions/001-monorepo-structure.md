# ADR-001: Monorepo with Self-Contained Projects

**Date:** 2026-03-21
**Status:** Accepted

## Context
We need to host backend, frontend, and mobile projects together for easier coordination, while keeping the ability to extract each into its own repo later.

## Decision
Use a monorepo where each project is fully self-contained:
- Own dependency manifests (`go.mod`, `package.json`, `build.gradle`)
- No shared code at the filesystem level — communication via APIs only
- Separate CI workflows per project

## Consequences
- Easy to extract any project to its own repo via `git filter-repo` with zero refactoring
- CI can run only what changed (path filters)
- Simple for a small team — one clone, one context
