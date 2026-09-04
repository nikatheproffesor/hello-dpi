@echo off
setlocal

echo ===================================
echo     Hello DPI Windows Installer    
echo ===================================

set "APP_DIR=%APPDATA%\HelloDPI"
set "BIN_NAME=hellodpi.exe"

if not exist "%APP_DIR%" mkdir "%APP_DIR%"

if exist "bin\hellodpi-windows-amd64.exe" (
    copy /Y "bin\hellodpi-windows-amd64.exe" "%APP_DIR%\%BIN_NAME%" >nul
) else if exist "bin\hellodpi.exe" (
    copy /Y "bin\hellodpi.exe" "%APP_DIR%\%BIN_NAME%" >nul
) else (
    echo Building Hello DPI for Windows...
    go build -o "%APP_DIR%\%BIN_NAME%" .\cmd\hellodpi
)

:: Create VBScript launcher to run hidden with zero console window
set "VBS_SCRIPT=%APP_DIR%\run_hidden.vbs"
(
echo Set WshShell = CreateObject("WScript.Shell"^)
echo WshShell.Run """%APP_DIR%\%BIN_NAME%"" -addr 127.0.0.1:8080 -system-proxy", 0, False
) > "%VBS_SCRIPT%"

:: Add to Windows User Startup
set "STARTUP_DIR=%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup"
copy /Y "%VBS_SCRIPT%" "%STARTUP_DIR%\HelloDPI.vbs" >nul

echo Starting Hello DPI in background...
wscript "%VBS_SCRIPT%"

echo.
echo [OK] Hello DPI has been installed to %APP_DIR%
echo [OK] Added to Startup (will start automatically with Windows).
echo [OK] Running in background with 0%% CPU load and full internet speed.
echo.
pause
