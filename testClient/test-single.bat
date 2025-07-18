@echo off
echo Starting single IM test client...
echo.
go run main.go -user=1002 -target=1003 -skip
pause
