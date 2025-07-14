#!/bin/bash

protoc --go_out=. --go_opt=paths=source_relative user.proto
protoc --go_out=. --go_opt=paths=source_relative group.proto
protoc --go_out=. --go_opt=paths=source_relative message.proto
protoc --go_out=. --go_opt=paths=source_relative connect.proto
protoc --go_out=. --go_opt=paths=source_relative friend.proto