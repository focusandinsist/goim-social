@echo off
echo 按依赖关系顺序启动微服务...
echo.

REM 第一层：基础服务（无依赖）
echo 启动基础服务层...
set HTTP_PORT=21001&& set GRPC_PORT=22001&& start "1-User-Service" cmd /k "set HTTP_PORT=21001&& set GRPC_PORT=22001&& go run apps\user-service\cmd\main.go"
timeout /t 3 /nobreak >nul

set HTTP_PORT=21002&& set GRPC_PORT=22002&& start "2-Group-Service" cmd /k "set HTTP_PORT=21002&& set GRPC_PORT=22002&& go run apps\group\cmd\main.go"
timeout /t 3 /nobreak >nul

REM 第二层：业务服务（依赖基础服务）
echo 启动业务服务层...
set HTTP_PORT=21003&& set GRPC_PORT=22003&& start "3-Friend-Service" cmd /k "set HTTP_PORT=21003&& set GRPC_PORT=22003&& go run apps\friend\cmd\main.go"
timeout /t 3 /nobreak >nul

set HTTP_PORT=21004&& set GRPC_PORT=22004&& start "4-Message-Service" cmd /k "set HTTP_PORT=21004&& set GRPC_PORT=22004&& go run apps\message\cmd\main.go"
timeout /t 3 /nobreak >nul

REM 第三层：接入服务（依赖所有业务服务）
echo 启动接入服务层...
set HTTP_PORT=21005&& set GRPC_PORT=22005&& start "5-Connect-Service" cmd /k "set HTTP_PORT=21005&& set GRPC_PORT=22005&& go run apps\connect\cmd\main.go"

echo.
echo 所有微服务已按依赖顺序启动完成！
echo.
echo 端口分配：
echo 1. User Service    - HTTP:21001, gRPC:22001
echo 2. Group Service   - HTTP:21002, gRPC:22002
echo 3. Friend Service  - HTTP:21003, gRPC:22003
echo 4. Message Service - HTTP:21004, gRPC:22004
echo 5. Connect Service - HTTP:21005, gRPC:22005
echo.
pause