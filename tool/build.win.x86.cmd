@echo off
go env -w GOOS=windows GOARCH=386
cd /d %GOPATH%/bin/
go build -o dfs.front.x86.exe -v github.com/schwarzlichtbezirk/dfs/front
go build -o dfs.node.x86.exe -v github.com/schwarzlichtbezirk/dfs/node
xcopy %GOPATH%\src\github.com\schwarzlichtbezirk\dfs\config dfs-config /f /d /i /s /e /k /y
