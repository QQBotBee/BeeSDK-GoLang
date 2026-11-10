@echo off
rem 使用项目固定的 go-legacy-win7 工具链执行 go 命令，例如：other\bee-go.bat test ./...
rem 无论从哪个目录调用，都会先切换到项目根目录再执行，./... 始终指整个项目。
setlocal EnableExtensions DisableDelayedExpansion
call "%~dp0go_env.bat"
if errorlevel 1 exit /b 1
pushd "%~dp0.."
"%BEE_GO_EXE%" %*
set "BEE_GO_RESULT=%ERRORLEVEL%"
popd
exit /b %BEE_GO_RESULT%
