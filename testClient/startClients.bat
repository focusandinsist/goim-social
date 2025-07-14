@echo off
echo 启动多个测试客户端...
echo.

REM 启动用户1001的客户端，发给用户1002
start "Client-1001" cmd /k "go run main.go -user=1001 -target=1002"

REM 启动用户1002的客户端，发给用户1001  
start "Client-1002" cmd /k "go run main.go -user=1002 -target=1001"

REM 启动用户1003的客户端，发给用户1001
start "Client-1003" cmd /k "go run main.go -user=1003 -target=1001"

REM 启动自动模式客户端1004，发给用户1001
start "Auto-Client-1004" cmd /k "go run main.go -user=1004 -target=1001 -auto"

echo.
echo 所有客户端已启动！
echo 客户端配置：
echo   - 用户1001 发给 用户1002
echo   - 用户1002 发给 用户1001  
echo   - 用户1003 发给 用户1001
echo   - 用户1004 自动模式发给 用户1001
echo.
echo 按任意键退出...
pause 