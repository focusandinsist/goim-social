@echo off
setlocal enabledelayedexpansion
title IM Chat Client Launcher

echo.
echo ==========================================
echo           IM Chat Client Launcher
echo ==========================================
echo.

echo Select authentication mode:
echo 1. Register new user
echo 2. Login existing user
echo 3. Debug mode (skip authentication)
echo.

set /p auth_choice="Enter your choice (1-3): "

if "%auth_choice%"=="1" (
    set auth_mode=register
) else if "%auth_choice%"=="2" (
    set auth_mode=login
) else if "%auth_choice%"=="3" (
    set auth_mode=debug
) else (
    echo Invalid choice, defaulting to debug mode.
    set auth_mode=debug
)

if "%auth_mode%"=="debug" (
    echo.
    echo Debug mode configuration:

    set user_id=
    set /p user_id="Enter your user ID (default 1001): "
    if not defined user_id set user_id=1001

    set target_id=
    set /p target_id="Enter chat target user ID (default 1002): "
    if not defined target_id set target_id=1002

    echo.
    echo Launching chat client...
    echo User: !user_id!  <->  Target: !target_id!
    echo.

    start "" cmd /v:on /k "go run main.go -user=!user_id! -target=!target_id! -skip"
) else (
    set target_id=
    set /p target_id="Enter chat target user ID (default 1002): "
    if not defined target_id set target_id=1002

    echo.
    echo Launching chat client...
    echo Auth mode: %auth_mode%  <->  Target: !target_id!
    echo.

    start "" cmd /v:on /k "go run main.go -target=!target_id!"
)

echo.
echo Chat window started!
echo You can run this script multiple times to launch multiple chat sessions.
echo.
pause
