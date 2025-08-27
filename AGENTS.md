# AGENTS.md

## Project Overview
This file contains essential configuration and commands for LLM agents to work effectively with this codebase. It serves as a reference for automated agents and human developers.

## Build, Lint, Test Commands
- **Build**: `make node` (Go guardian), `npm run build` (JS/TS projects), `forge build` (Solidity)
- **Lint**: `./scripts/lint.sh lint` (Go + spellcheck), `npm run lint` (TypeScript projects)
- **Test**: `npm test` (JS/TS), `jest --verbose` (specific tests), `DEV=true NETWORK=DEVNET jest --verbose` (SDK tests)
- **Single test**: `jest path/to/test.js -t "test name"` or `npm test -- --testNamePattern="pattern"`
- **Format**: `./scripts/lint.sh format` (Go), `./scripts/lint.sh -w format` (auto-fix)

## Code Style Guidelines
- **Go**: Use `goimports`, follow `.golangci.yml` rules, camelCase for unexported, PascalCase for exported
- **TypeScript**: Use existing project conventions, prefer `import` over `require`, explicit types over `any`
- **Solidity**: Follow existing patterns in `ethereum/contracts/`, use Forge for builds
- **Imports**: Organize by standard lib, external packages, internal packages (Go); relative imports last (TS)
- **Error handling**: Always handle errors explicitly, use proper error types, no silent failures
- **Naming**: Descriptive names, avoid abbreviations, match existing patterns in codebase
- **Comments**: Follow [Go Doc Comments](https://go.dev/doc/comment) for Go, [TSDoc](https://tsdoc.org/) for TypeScript

## Development Workflow
- **Pre-commit**: Run `./scripts/lint.sh -d format` and `./scripts/lint.sh lint` before committing
- **Commits**: Small atomic commits with meaningful messages, prefix with component name (e.g., "node: fix bug")
- **PRs**: All features require GitHub issue discussion first, complex features need design docs
- **Quality focus**: Optimize for reading over writing, meaningful commit messages, useful comments
- **Dependencies**: Document benefits vs security/compatibility when updating dependencies

## Testing by Component
- **Guardian Node**: `cd node && make test` (tests in `./node/**/*_test.go`)
- **Ethereum**: `cd ethereum && make test` (tests in `./ethereum/test/`)  
- **Solana**: `cd solana && make test` (tests in `./solana/bridge/program/tests/`)
- **Cosmwasm/Algorand**: `cd {component} && make test`
