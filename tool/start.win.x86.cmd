@echo off
cd /d %GOPATH%/bin/
start "front" dfs.front.x86.exe
start "node#1" dfs.node.x86.exe -p :50051
start "node#2" dfs.node.x86.exe -p :50052
