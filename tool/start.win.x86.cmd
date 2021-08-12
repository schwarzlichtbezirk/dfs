@echo off
cd /d %GOPATH%/bin/
start "DFS front" dfs.front.x86.exe
start "DFS node#1" dfs.node.x86.exe -p :50051
start "DFS node#2" dfs.node.x86.exe -p :50052
