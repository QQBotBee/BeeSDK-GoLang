# Repository Guidelines

## Project Structure & Module Organization

This is a Go (go-legacy-win7 1.27.3-1) Bee plugin SDK/template for Windows. Primary plugin work happens in `plugin_main.go`, which defines plugin metadata, lifecycle hooks, and message handlers. `bee_sdk.go` contains the Bee API wrapper, event constants, and IPC client. `settings.go` contains the native Windows settings window. Low-level runtime and build support live in `other/`: the C Bee shell, export definition file, worker runtime, and build metadata generator. Project docs are in `docs/`; generated `build/` and `temp/` outputs should not be committed.

## Build, Test, and Development Commands

- `other\bee-go.bat test ./...`: runs all Go package tests and is the baseline local check.
- `other\bee-go.bat vet ./...`: runs Go static analysis before submitting shared changes.
- `build.bat`: builds the Windows 32-bit plugin DLL using Go and Zig.
- `build.bat MyPlugin.dll`: builds the DLL with an explicit output name.

The build expects Windows CMD, Windows PowerShell, and Zig in `PATH`, and produces `build\<plugin-name>.dll`. The Go toolchain is pinned to the go-legacy-win7 1.27.3-1 fork (https://github.com/thongtech/go-legacy-win7), never the official Go or any `go.exe` in `PATH`. `other/go_env.bat` downloads the release ZIP into `.toolchain/`, verifies its pinned SHA-256, and sets `GOROOT`, `GOTOOLCHAIN=local` and `GOWORK=off`. Run every Go command through `other\bee-go.bat` so tests and vet use the same toolchain. Do not switch to the official Go or change the fork version without updating the hashes in `other/go_toolchain.ps1`: compatibility with Windows 7 and Windows Server 2012 R2 requires this fork. The fork reports the same `go version` as upstream, so `build.bat` also checks the worker PE subsystem version (6.1 for the fork, 10.0 for official Go) before embedding it. `go.mod` declares `go 1.27.3` (Go 1.27.3 language semantics); it does not select the compiler, and must not gain a `toolchain` line.

## Coding Style & Naming Conventions

Use standard Go formatting: run `gofmt` on changed `.go` files and keep package-level names clear and idiomatic. Keep business logic in `plugin_main.go` unless it becomes large enough to justify a focused helper file. Use descriptive handler names that match Bee callback concepts, such as `onGroupMessage` or `onSettings`. Preserve Windows/386 assumptions and do not move Go runtime code into the Bee DLL.

## Testing Guidelines

Add tests as `*_test.go` files next to the code under test. Prefer table-driven tests for parsing, message handling helpers, and SDK behavior. `other\bee-go.bat test ./...` should pass before every PR. Full plugin lifecycle, message callbacks, settings windows, worker shutdown, and DLL unload stability still require verification in a real Bee environment.

## Commit & Pull Request Guidelines

The current Git history is minimal, so use concise imperative commit subjects, for example `feat: add group reply helper` or `fix: preserve GBK export encoding`. PRs should include a short description, testing performed, Bee runtime verification when relevant, and screenshots for settings-window changes.

## Security & Configuration Tips

Keep `build.bat` and `other/BeePlugin.def` in their required encodings; `.gitattributes` marks them binary to prevent line-ending or encoding damage. Store plugin data through `GetAppDataDir()` rather than beside the DLL or in temporary plugin directories.
