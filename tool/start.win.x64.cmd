@echo off
cd /d %GOPATH%/bin/
start "front" dfs.front.x64.exe
start "node#1" dfs.node.x64.exe -p :50051
start "node#2" dfs.node.x64.exe -p :50052
