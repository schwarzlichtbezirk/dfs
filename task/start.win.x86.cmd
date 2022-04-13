@echo off
start "DFS front" %GOPATH%/bin/dfs.front.x86.exe
start "DFS node#1" %GOPATH%/bin/dfs.node.x86.exe -p :50051
start "DFS node#2" %GOPATH%/bin/dfs.node.x86.exe -p :50052
