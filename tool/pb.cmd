@echo off
set pbimport=github.com/schwarzlichtbezirk/dfs/pb
protoc -I=%GOPATH%/src/%pbimport%/^
 --go_out=%GOPATH%/src/ --go-grpc_out=%GOPATH%/src/^
 %GOPATH%/src/%pbimport%/dfs.proto
 