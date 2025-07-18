@echo off
echo Start two-user conversation test
echo.

echo Description:
echo Since debug mode uses fixed user 1001, to test two-user conversation need:
echo 1. First client: normal mode, register user A
echo 2. Second client: normal mode, register user B
echo Or
echo 1. First client: debug mode, user 1001
echo 2. Second client: normal mode, register real user, target set to 1001
echo.

echo Choose test method:
echo 1. Two normal mode clients (need register two different users)
echo 2. One debug mode + one normal mode
echo 3. Manual specify parameters
echo.

set /p choice="Please select (1-3): "

if "%choice%"=="1" (
    echo Starting two normal auth mode clients...
    echo Client1: need register/login user A, target user set to registered user B ID
    start "IM-User-A" cmd /k "go run main.go -target=1002"
    timeout /t 2 /nobreak >nul
    echo Client2: need register/login user B, target user set to registered user A ID
    start "IM-User-B" cmd /k "go run main.go -target=1001"
    goto end
)

if "%choice%"=="2" (
    echo Starting mixed mode test...
    echo Client1: debug mode, user 1001, target user 1002
    start "IM-Debug-1001" cmd /k "go run main.go -target=1002 -skip"
    timeout /t 2 /nobreak >nul
    echo Client2: normal mode, need register user, suggest target user set to 1001
    start "IM-Normal-User" cmd /k "go run main.go -target=1001"
    goto end
)

if "%choice%"=="3" (
    echo Manual mode:
    echo Please run in two command line windows separately:
    echo.
    echo Window1: go run main.go -target=target_user_id -skip
    echo Window2: go run main.go -target=target_user_id
    echo.
    echo Or both use normal mode:
    echo Window1: go run main.go -target=target_user_id
    echo Window2: go run main.go -target=target_user_id
    goto end
)

echo Invalid selection, starting default mixed mode...
start "IM-Debug-1001" cmd /k "go run main.go -target=1002 -skip"
timeout /t 2 /nobreak >nul
start "IM-Normal-User" cmd /k "go run main.go -target=1001"

:end
echo.
echo Test clients started!
echo.
echo Important tips:
echo - Debug mode uses fixed user ID 1001
echo - Normal mode need register/login to get real user ID
echo - Ensure two clients target user IDs correspond to each other
echo - Type 'help' in client to see available commands
echo.
pause
