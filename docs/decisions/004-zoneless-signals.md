# ADR-004: Zoneless Change Detection with Pure Signals

**Date:** 2026-03-21
**Status:** Accepted

## Context
Angular historically required Zone.js to detect changes (monkey-patching async APIs).
Angular 18+ introduced signals-based reactivity and a zoneless scheduler.
Angular 21 graduated `provideZonelessChangeDetection()` to stable.

## Decision
- No Zone.js — removed from dependencies entirely
- Change detection driven purely by signals (`signal()`, `computed()`, `effect()`)
- `provideZonelessChangeDetection()` in app config
- All state in components and services uses Angular Signals

## Consequences
- **Better performance** — no unnecessary CD cycles, no monkey-patching overhead
- **Predictable rendering** — updates happen exactly when signals change, nothing more
- **Smaller bundle** — Zone.js (~35kB) removed
- **Constraint** — all reactive state must go through signals; `setTimeout`/`Promise` don't trigger CD automatically, which is the desired behavior
