# Repository Guidelines

## Project Structure & Module Organization

This module is `github.com/calm195/go-template`. `cmd/server` assembles dependencies and manages the HTTP server lifecycle. `internal/config` loads YAML and environment configuration; `internal/platform` owns logging and middleware clients. `internal/httpapi` contains Gin handlers and middleware. `internal/user` demonstrates the service and repository boundary. Keep new features grouped by domain under `internal`. Configuration examples live in `configs`; database-specific SQL lives in `migrations`. Tests sit beside source files.

## Build, Test, and Development Commands

Use Go 1.25 or newer. Run `go mod tidy` after changing dependencies and commit both `go.mod` and `go.sum`. `make run` starts the API without external services by default. `make build` produces `bin/server`. `make fmt` applies gofmt; `make check` runs vet, race-enabled tests, coverage reporting, and formatting checks. `docker compose up -d --build app` starts the container. Start optional middleware using Compose profiles described in README.md.

## Coding Style & Naming Conventions

Use gofmt with Tab indentation. Group imports as standard library, third-party packages, then this module, separated by blank lines. Expand multi-field configuration literals across lines and separate top-level declarations with blank lines. Use standard Go naming: lowercase package names, MixedCaps identifiers, and descriptive exported API comments. Accept `context.Context` as the first parameter for I/O operations. Pass dependencies explicitly; avoid mutable global clients. Keep HTTP validation and status mapping in handlers, business rules in services, and GORM queries in repositories. Use `WithContext` for database calls. Wrap errors with useful context without exposing credentials or internal errors in HTTP responses.

## Testing Guidelines

Use the standard `testing` package and `net/http/httptest`. Name files `*_test.go` and tests `TestBehavior`; prefer table-driven cases for validation and HTTP failures. Unit tests must run without middleware. Add explicit integration tests when changing real database, Redis, or RocketMQ behavior, and document required services. No minimum coverage percentage is enforced. Run `make check` and `make build` before submitting changes.

## Commit & Pull Request Guidelines

The repository has no established commit history. Use Conventional Commits such as `feat: add user lookup` or `fix: close redis client`. PRs should describe the problem, resulting behavior, test commands and outcomes, linked issues, and any configuration or schema migration steps. Include screenshots only for visible UI changes.

## Configuration & Agent Instructions

Environment variables use the `APP_` prefix. Never commit secrets, local overrides, or production DSNs. Apply explicit database-specific migrations; do not add startup AutoMigrate. Codex should read this guide before editing, preserve unrelated changes, keep scope focused, and report checks that could not run. Never claim tests passed without execution.

## Required Checks After Every Change

After every modification, run `make fmt` and `make check` before declaring the work complete. `make check` must verify formatting without modifying files, run `go vet ./...`, and run `go test -race -cover ./...`. Also run `make build` after Go code or dependency changes. Fix all reported issues and rerun affected checks. If environment restrictions prevent a check, report the command and reason explicitly; do not skip it silently or claim it passed. Keep this workflow enforced in CI.

## Required Comments & Completion Logging

注释默认使用中文，包括包注释、导出符号文档、关键逻辑和测试说明。导出符号注释仍以对应的 Go 标识符开头；保留必要的技术术语及工具指令（如 `//go:build`），不修改生成代码中的注释。用户明确要求其他语言时遵循用户要求。

Every change must add or update comments at the affected critical points: business rules, lifecycle ordering, resource ownership, recovery, and non-obvious decisions. Explain why the code exists and keep comments accurate. Every changed runtime operation must emit a structured completion log using injected `*zap.Logger` (`go.uber.org/zap`), with request ID, duration, result, or safe identifiers as applicable. Log success only after the operation succeeds; record failures at the owning boundary. Never log credentials, DSNs, request bodies, or personal data. Preserve request completion logs on failure and recovery paths. Documentation-only changes must report completion and validation in the final response. Update relevant log assertions and run the required checks after every change.
