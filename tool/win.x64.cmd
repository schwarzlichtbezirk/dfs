@echo off
go env -w GOOS=windows GOARCH=amd64
go build -o %GOPATH%\bin\dfs.front.x64.exe -v github.com/schwarzlichtbezirk/dfs/front
go build -o %GOPATH%\bin\dfs.node.x64.exe -v github.com/schwarzlichtbezirk/dfs/node
