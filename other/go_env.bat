@echo off
rem 固定 Go 工具链：go-legacy-win7 1.27.3-1（保留 Windows 7/8/8.1/Server 2012 R2 支持的 Go fork）。
rem 由 build.bat 和 bee-go.bat 调用，在调用方环境中设置 GOROOT、PATH 和 BEE_GO_EXE。
rem 不使用 PATH 中已安装的任何 Go；首次运行时自动下载到项目 .toolchain 目录并校验 SHA-256。
set "BEE_GO_FORK_VERSION=1.27.3-1"
set "REQUIRED_GO_VERSION=go1.27.3"
for %%I in ("%~dp0..") do set "BEE_PROJECT_DIR=%%~fI"
set "BEE_TOOLCHAIN_DIR=%BEE_PROJECT_DIR%\.toolchain\go-legacy-win7-%BEE_GO_FORK_VERSION%"
set "BEE_GO_EXE=%BEE_TOOLCHAIN_DIR%\go\bin\go.exe"
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0go_toolchain.ps1" -Action Install -ToolchainDir "%BEE_TOOLCHAIN_DIR%"
if errorlevel 1 goto :toolchain_fail
set "GOROOT=%BEE_TOOLCHAIN_DIR%\go"
set "GOTOOLCHAIN=local"
set "GOWORK=off"
set "PATH=%BEE_TOOLCHAIN_DIR%\go\bin;%PATH%"
set "BEE_GO_VERSION="
for /f "tokens=3" %%V in ('call "%BEE_GO_EXE%" version 2^>nul') do set "BEE_GO_VERSION=%%V"
if not "%BEE_GO_VERSION%"=="%REQUIRED_GO_VERSION%" goto :version_fail
exit /b 0

:version_fail
echo 错误：固定工具链版本异常，期望 %REQUIRED_GO_VERSION%，实际 %BEE_GO_VERSION%。
echo 工具链路径：%BEE_GO_EXE%
echo 可删除 .toolchain 目录后重新运行，以重新下载 go-legacy-win7 %BEE_GO_FORK_VERSION%。
exit /b 1

:toolchain_fail
echo 错误：无法准备 go-legacy-win7 %BEE_GO_FORK_VERSION% 工具链。
echo 发布页：https://github.com/thongtech/go-legacy-win7/releases/tag/v%BEE_GO_FORK_VERSION%
exit /b 1
