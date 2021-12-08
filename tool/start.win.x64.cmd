@echo off
start "DFS front" %GOPATH%/bin/dfs.front.x64.exe
start "DFS node#1" %GOPATH%/bin/dfs.node.x64.exe -p :50051
start "DFS node#2" %GOPATH%/bin/dfs.node.x64.exe -p :50052
