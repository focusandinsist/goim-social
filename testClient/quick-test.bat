@echo off
echo Quick Test - IM Client
echo.
echo Starting debug mode client...
echo Current user: 1002 (fixed, skip auth)
echo Target user: 1003
echo.
go run main.go -user=1002 -target=1003 -skip
pause
