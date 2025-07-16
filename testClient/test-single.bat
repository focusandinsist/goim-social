@echo off
echo 启动单个测试客户端...
echo.

REM 检查参数
if "%1"=="" (
    echo 用法: test-single.bat [用户ID] [目标用户ID]
    echo 示例: test-single.bat 1001 1002
    echo.
    echo 使用默认参数启动用户1001...
    go run main.go -user=1001 -target=1002
) else if "%2"=="" (
    echo 启动用户%1，目标用户1002...
    go run main.go -user=%1 -target=1002
) else (
    echo 启动用户%1，目标用户%2...
    go run main.go -user=%1 -target=%2
)

pause
