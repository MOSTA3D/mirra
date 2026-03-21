# ADR-002: Dark-First Design with CSS Token Architecture

**Date:** 2026-03-21
**Status:** Accepted

## Context
Mirra needs a premium, dark-first UI that can support multiple themes in the future without refactoring components.

## Decision
- V1 ships dark mode only
- All colors defined as CSS custom properties (design tokens) under `[data-theme]` selectors
- Tailwind configured to reference tokens, never raw color values
- Components use only token-mapped Tailwind classes — no hardcoded hex values ever
- Angular `ThemeService` manages `data-theme` attribute on `document.documentElement`

## Consequences
- Adding a new theme = one new CSS file defining the same token set
- Zero component changes when switching themes
- Slightly more upfront setup, pays off immediately on first theme addition
