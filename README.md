# Mirra

> Build a reflection of anyone.

Mirra is a persona distillation platform. Feed it resources about a person — social media, PDFs, text — and it extracts a structured, interactive persona you can export or deploy as an agent.

## Monorepo Structure

```
mirra/
├── backend/          # Go API + processing pipeline
├── frontend/         # Angular 18+ web app
├── mobile/
│   ├── android/      # Jetpack Compose
│   └── ios/          # SwiftUI
├── docs/             # Architecture, decisions, API specs
├── infra/            # Docker, CI/CD, deployment
└── .github/
    └── workflows/    # CI per project
```

## Stack

- **Backend:** Go
- **Frontend:** Angular 18+
- **Android:** Jetpack Compose
- **iOS:** SwiftUI
- **Database:** PostgreSQL
- **Cache/Queue:** Redis
- **Storage:** S3-compatible (Cloudflare R2)

## Branches

- `main` — always deployable, protected
- `dev` — integration branch, PRs merge here
- `feat/*`, `fix/*`, `chore/*` — feature branches

## Commit Convention

Follows [Conventional Commits](https://www.conventionalcommits.org/):
- `feat:` new feature
- `fix:` bug fix
- `chore:` tooling, config
- `docs:` documentation
- `wip:` work in progress (mid-feature checkpoint)

## Rules

- Every unit of work = one commit
- No hardcoded colors — design tokens only
- No business logic in controllers — services layer owns it
- All config via environment variables
- API versioned from day one: `/api/v1/...`
- Consistent response envelope: `{ data, error, meta }`
- No `TODO` without a linked issue
- Every significant technical decision gets an ADR in `docs/decisions/`
