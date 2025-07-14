@echo off
protoc -I=. --go_out=. --go_opt=paths=source_relative user.proto
protoc -I=. --go_out=. --go_opt=paths=source_relative group.proto
protoc -I=. --go_out=. --go_opt=paths=source_relative message.proto
protoc -I=. --go_out=. --go_opt=paths=source_relative connect.proto
protoc -I=. --go_out=. --go_opt=paths=source_relative friend.proto
protoc -I=. --go_out=. --go-grpc_out=. --go_opt=paths=source_relative friend.grpc.proto
protoc -I=. --go_out=. --go-grpc_out=. --go_opt=paths=source_relative message.grpc.proto
protoc -I=. --go_out=. --go-grpc_out=. --go_opt=paths=source_relative connect.grpc.proto
protoc -I=. --go_out=. --go-grpc_out=. --go_opt=paths=source_relative group.grpc.proto
protoc -I=. --go_out=. --go-grpc_out=. --go_opt=paths=source_relative user.grpc.proto
