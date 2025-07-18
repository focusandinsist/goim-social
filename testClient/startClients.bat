@echo off
echo IM Test Client Launcher
echo.

echo Available startup modes:
echo 1. Normal mode - Full register/login authentication process
echo 2. Debug mode - Skip auth, use fixed user 1002
echo 3. Auto test - Start two auto-messaging clients (user 1002 and 1003)
echo 4. Single client - Start single debug mode client
echo.

set /p mode="Please select mode (1-4): "

if "%mode%"=="1" (
    echo Starting normal auth mode client...
    echo Note: Need to register or login to get real user ID
    start "IM-Client-Auth" cmd /k "go run main.go -target=1002"
    goto end
)

if "%mode%"=="2" (
    echo Starting debug mode client...
    echo Using fixed user 1001, target user 1002
    start "IM-Client-Debug" cmd /k "go run main.go -target=1002 -skip"
    goto end
)

if "%mode%"=="3" (
    echo Starting auto test mode...
    echo Client1: user 1001 auto send to user 1002
    start "IM-Auto-1001" cmd /k "go run main.go -target=1002 -skip -auto"
    timeout /t 2 /nobreak >nul
    echo Client2: user 1001 auto send to user 1001 (Note: need different users for real test)
    start "IM-Auto-1002" cmd /k "go run main.go -target=1001 -skip -auto"
    echo Two auto test clients started!
    echo   - Client1: user 1001 -^> user 1002
    echo   - Client2: user 1001 -^> user 1001 (same user, need register different users)
    goto end
)

if "%mode%"=="4" (
    echo Starting single client debug mode...
    echo Using fixed user 1001, target user 1002
    start "IM-Single-Client" cmd /k "go run main.go -target=1002 -skip"
    goto end
)

echo Invalid selection, starting default debug mode...
start "IM-Client-Default" cmd /k "go run main.go -target=1002 -skip"

:end
echo.
echo Clients started!
echo Tip: Type 'help' in client to see available commands
echo.
pause