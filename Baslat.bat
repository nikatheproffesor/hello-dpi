@echo off
chcp 65001 >nul
title Hello DPI
powershell -Command "Unblock-File -Path '%~dp0HelloDPI-Windows.exe' -ErrorAction SilentlyContinue"
start "" "%~dp0HelloDPI-Windows.exe"
exit
