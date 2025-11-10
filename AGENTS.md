# Repository Guidelines
This guide orients contributors to the dual Go/Vue workspace so changes stay consistent, reproducible, and production ready.

## Project Structure & Module Organization
`backend/` holds the Go API, with `config/`, `controllers/`, `services/`, `models/`, and `websocket/` mirroring the standard layering. `poker_score_frontend/` is the Vue 3 client (`src/` for views/stores, `public/` static assets, `dist/` sample build). Shared docs live in `docs/` (API, database, deployment notes), while integration helpers and legacy smoke scripts reside in `test/`. Keep SQLite artifacts (`backend/database.db`) out of commits unless schema changes are intentional.

## Build, Test & Development Commands
- `cd backend && go run .` launches the Gin server on `:8080` using the local SQLite file.
- `cd poker_score_frontend && npm run dev` serves the Vite UI on `:5173` with `/api` proxying.
- `cd backend && go test ./...` executes unit and controller integration tests; use before pushing backend changes.
- `cd poker_score_frontend && npm run build` produces optimized assets; `npm run preview` validates the static bundle.
- `cd test && python3 test_api.py --base-url http://localhost:8080` runs the legacy HTTP smoke suite once both services are up.

## Coding Style & Naming Conventions
Run `go fmt ./...` (or rely on your editor) before committing; exported Go symbols stay PascalCase, internal helpers camelCase, and JSON/database fields map via explicit `gorm`/`json` tags that use snake_case. Frontend code follows ESLint + Prettier (`npm run lint`, `npm run format`) with 2-space indentation, Composition API `<script setup>` components, and kebab-case file names under `src/components`. Favor descriptive room/action identifiers such as `useRoomStore` or `settleTransfer`.

## Testing Guidelines
Backend tests live primarily in `backend/controllers` and assert websocket-safe flows; add table-driven cases with `t.Run` and avoid hitting the real database file—use temporary paths via `os.CreateTemp`. Keep test names in the form `Test<Feature><Condition>`. Frontend work should at minimum pass `npm run type-check`; add Vitest components next to the source when practical. When touching settlement logic, run both `go test ./...` and `python3 test/test_api.py` to cover regression gaps.

## Commit & Pull Request Guidelines
History favors short, descriptive summaries in Chinese (e.g., “重构服务器初始化逻辑”) that bundle related touches; mirror that style and keep imperative voice. Each PR should describe scope, highlight DB migrations or config knobs, attach screenshots/GIFs for UI tweaks, and link tracker issues. Include verification notes (commands run, affected modules) so reviewers can replay steps quickly.

## Security & Configuration Tips
Environment defaults live in `backend/config`; override via `.env` or export when deploying. Always set `SERVER_ALLOWED_ORIGINS`, `SERVER_COOKIE_DOMAIN`, and `SERVER_COOKIE_SECURE=true` before exposing public endpoints. Never commit `.env`, `database.db`, or production `deploy.sh` secrets; scrub credentials from screenshots and logs prior to sharing.

