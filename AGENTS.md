# Development Guidelines

## Project

- This is a Wails v2 desktop application.
- The backend uses Go.
- The frontend uses Svelte and Vite.
- Keep the application lightweight and avoid unnecessary dependencies.

## Development

- Run `wails dev` for local development.
- Run `wails build` to build the desktop application.
- Run `npm run build` from `frontend/` to validate frontend changes.
- Run `go test ./...` for Go changes.
- Do not commit `.DS_Store`, `node_modules/`, or generated build output.

## Commits

- Use Conventional Commits format, for example:
  - `feat: add account management`
  - `fix: handle provider errors`
  - `chore: update dependencies`
- Use real newlines for multi-line commit messages.
