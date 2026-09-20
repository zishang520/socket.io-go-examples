@echo off
setlocal EnableExtensions DisableDelayedExpansion
pushd "%~dp0" || exit /b 1
set "MAKE_ARGS=%*"
call "%~dp0..\scripts\module.bat" %%MAKE_ARGS%%
set "RESULT=%errorlevel%"
popd
exit /b %RESULT%
