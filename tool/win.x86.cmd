@echo off
go env -w GOOS=windows GOARCH=386
go build -o %GOPATH%\bin\dfs.front.x86.exe -v github.com/schwarzlichtbezirk/dfs/front
go build -o %GOPATH%\bin\dfs.node.x86.exe -v github.com/schwarzlichtbezirk/dfs/node