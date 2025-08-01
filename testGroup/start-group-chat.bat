@echo off
echo ========================================
echo      Multi-Window Group Chat Client
echo ========================================
echo.
echo Features:
echo - Auto-fetch group members and online status
echo - Multi-window chat interface (max 5 windows)
echo - Real-time messaging across all windows
echo - Send messages as different users
echo.
echo Usage:
echo 1. Enter group ID
echo 2. System auto-fetches group info and members
echo 3. Chat windows open for each member
echo 4. Use @userID message format to send messages
echo 5. Type 'list' to view member status, 'quit' to exit
echo.
echo Press any key to start...
pause > nul

cd /d "%~dp0"
go run main.go

echo.
echo Program exited. Press any key to close window...
pause > nul
