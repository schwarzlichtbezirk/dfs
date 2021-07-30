@echo off
go env -w GOOS=windows GOARCH=amd64
cd /d %GOPATH%/bin/
go build -o dfs.front.x64.exe -v github.com/schwarzlichtbezirk/dfs/front
go build -o dfs.node.x64.exe -v github.com/schwarzlichtbezirk/dfs/node
