@echo off
setlocal EnableExtensions DisableDelayedExpansion
if not defined GOPROXY set "GOPROXY=https://proxy.golang.org,direct"
if not defined TEST_TIMEOUT set "TEST_TIMEOUT=60s"

:: Called by each example's wrapper with its module as the working directory.
if "%~1"=="" goto :Help
if /I "%~1"=="help" goto :Help
set "ACTION="
for %%A in (env deps get update build fmt vet lint clean test) do if /I "%~1"=="%%A" set "ACTION=%%A"
if not defined ACTION (
    echo [Error] Unknown command: %~1
    exit /b 1
)
set "FIX="
if "%ACTION%"=="lint" if /I "%~2"=="--fix" set "FIX=--fix"
if not "%~2"=="" if not defined FIX goto :InvalidArgs
if not "%~3"=="" goto :InvalidArgs
if not exist go.mod (
    echo [Error] Run this command from an example module.
    exit /b 1
)
where go >nul 2>nul || (
    echo [Error] Go is not installed or not in PATH.
    exit /b 1
)
echo [%ACTION%] Processing: .
goto :%ACTION%

:env
call go env
exit /b

:deps
:: Use the active workspace, including GOWORK=off or an external go.work.
set "WORKSPACE_FILE=%TEMP%\socketio-workspace-%RANDOM%-%RANDOM%.txt"
call go env GOWORK >"%WORKSPACE_FILE%"
set "WORKSPACE_RESULT=%errorlevel%"
set "WORKSPACE="
set /p "WORKSPACE=" <"%WORKSPACE_FILE%"
del /q "%WORKSPACE_FILE%" >nul 2>nul
if not "%WORKSPACE_RESULT%"=="0" exit /b %WORKSPACE_RESULT%
call go mod tidy || exit /b
if not defined WORKSPACE goto :ModuleVendor
if "%WORKSPACE%"=="off" goto :ModuleVendor
call go work sync || exit /b
call go work vendor
exit /b

:ModuleVendor
call go mod vendor
exit /b

:get
call go get ./...
exit /b

:update
call go get -u -v ./... || exit /b
call :deps
exit /b

:build
call go build ./...
exit /b

:fmt
call go fmt ./...
exit /b

:clean
call go clean -v -r ./...
exit /b

:vet
call :deps || exit /b
call go vet ./...
exit /b

:lint
where golangci-lint >nul 2>nul || (
    echo [Error] golangci-lint is not installed.
    exit /b 1
)
call :deps || exit /b
call golangci-lint run --timeout=5m %FIX% ./... <nul
exit /b

:test
call :deps || exit /b
call go test -count=1 -timeout=%TEST_TIMEOUT% -race -cover -covermode=atomic ./... <nul
exit /b

:InvalidArgs
echo [Error] Only lint accepts an option: --fix.
exit /b 1

:Help
echo Usage: make.bat [env ^| deps ^| get ^| update ^| build ^| fmt ^| vet ^| lint ^| clean ^| test]
echo deps refreshes tidy and vendor, syncing the active workspace when enabled.
echo test, vet and lint refresh deps first; update runs go get -u first.
echo Options: lint --fix; environment: GOPROXY, TEST_TIMEOUT ^(default: 60s^).
exit /b 0
