# Development Guidelines

## Project

- This is a Wails v2 desktop application.
- The backend uses Go.
- The frontend uses Svelte, Vite, and Tailwind CSS v4.
- Keep the application lightweight and avoid unnecessary dependencies.
- Account display names, email addresses, avatars, and plan types come from Codex OAuth userinfo/JWT data; do not add manual account-name input.
- Account credentials belong in the macOS Keychain; account metadata and profiles use the Mux application-support directory.

## Development

- Run `wails dev` for local development.
- Run `wails build` to build the desktop application.
- Run `npm run build` from `frontend/` to validate frontend changes.
- Run `go test ./...` for Go changes.
- Run `go vet ./...` for static analysis.
- Run `golangci-lint run ./...` for lint checks.
- Run `go list ./... | xargs go run golang.org/x/tools/go/analysis/passes/modernize/cmd/modernize@latest -test=false` for Go modernization checks.
- Do not commit `.DS_Store`, `node_modules/`, or generated build output.
- Frontend hover hints must use the custom instant tooltip component; do not use the browser `title` attribute because its display is delayed.

## Commits

- Use Conventional Commits format, for example:
  - `feat: add account management`
  - `fix: handle provider errors`
  - `chore: update dependencies`
- Use real newlines for multi-line commit messages.
