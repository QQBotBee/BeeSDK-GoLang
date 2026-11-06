# Repository Guidelines

## Project Structure & Module Organization

This is a Go 1.20.14 Bee plugin SDK/template for Windows. Primary plugin work happens in `plugin_main.go`, which defines plugin metadata, lifecycle hooks, and message handlers. `bee_sdk.go` contains the Bee API wrapper, event constants, and IPC client. `settings.go` contains the native Windows settings window. Low-level runtime and build support live in `other/`: the C Bee shell, export definition file, worker runtime, and build metadata generator. Project docs are in `docs/`; generated `build/` and `temp/` outputs should not be committed.

## Build, Test, and Development Commands

- `go test ./...`: runs all Go package tests and is the baseline local check.
- `go vet ./...`: runs Go static analysis before submitting shared changes.
- `build.bat`: builds the Windows 32-bit plugin DLL using Go and Zig.
- `build.bat MyPlugin.dll`: builds the DLL with an explicit output name.

The build expects Windows CMD, exactly Go 1.20.14 as the first `go.exe` in `PATH`, Zig in `PATH`, and produces `build\<plugin-name>.dll`. Run Go tests and vet with Go 1.20.14 as well. Do not upgrade the release compiler: compatibility with Windows Server 2012 R2 requires this pinned toolchain. `go.mod` declares the language version; `build.bat` enforces the exact compiler version and verifies the built worker before embedding it.

## Coding Style & Naming Conventions

Use standard Go formatting: run `gofmt` on changed `.go` files and keep package-level names clear and idiomatic. Keep business logic in `plugin_main.go` unless it becomes large enough to justify a focused helper file. Use descriptive handler names that match Bee callback concepts, such as `onGroupMessage` or `onSettings`. Preserve Windows/386 assumptions and do not move Go runtime code into the Bee DLL.

## Testing Guidelines

Add tests as `*_test.go` files next to the code under test. Prefer table-driven tests for parsing, message handling helpers, and SDK behavior. `go test ./...` should pass before every PR. Full plugin lifecycle, message callbacks, settings windows, worker shutdown, and DLL unload stability still require verification in a real Bee environment.

## Commit & Pull Request Guidelines

The current Git history is minimal, so use concise imperative commit subjects, for example `feat: add group reply helper` or `fix: preserve GBK export encoding`. PRs should include a short description, testing performed, Bee runtime verification when relevant, and screenshots for settings-window changes.

## Security & Configuration Tips

Keep `build.bat` and `other/BeePlugin.def` in their required encodings; `.gitattributes` marks them binary to prevent line-ending or encoding damage. Store plugin data through `GetAppDataDir()` rather than beside the DLL or in temporary plugin directories.
