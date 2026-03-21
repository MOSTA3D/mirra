# ADR-003: REST API Design Conventions

**Date:** 2026-03-21
**Status:** Accepted

## Context
Need consistent API design that can evolve without breaking clients.

## Decision
- REST, versioned from day one: `/api/v1/...`
- All responses follow a consistent envelope:
  ```json
  {
    "data": {},
    "error": null,
    "meta": {
      "requestId": "...",
      "timestamp": "..."
    }
  }
  ```
- Errors follow:
  ```json
  {
    "data": null,
    "error": {
      "code": "PERSONA_NOT_FOUND",
      "message": "The requested persona does not exist"
    },
    "meta": {}
  }
  ```
- Breaking changes bump the version (`/api/v2/...`), never modify existing contracts
- All endpoints documented in `docs/api/` before implementation

## Consequences
- Clients can always expect the same shape
- Error handling is predictable
- Version bumps are rare but clean when needed
