#!/bin/bash

protoc --go_out=. --go_opt=paths=source_relative user.proto
protoc --go_out=. --go_opt=paths=source_relative social.proto
protoc --go_out=. --go_opt=paths=source_relative message.proto
protoc --go_out=. --go_opt=paths=source_relative connect.proto
protoc --go_out=. --go_opt=paths=source_relative content.proto
protoc --go_out=. --go_opt=paths=source_relative history.proto

# 生成 gRPC 文件
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative user.grpc.proto
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative social.grpc.proto
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative message.grpc.proto
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative logic.grpc.proto
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative connect.grpc.proto