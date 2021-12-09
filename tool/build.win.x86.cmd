@echo off
cd /d %GOPATH%\src\github.com\schwarzlichtbezirk\dfs
xcopy .\config %GOPATH%\bin\dfs-config /f /d /i /e /k /y
go env -w GOOS=windows GOARCH=386
go build -o %GOPATH%/bin/dfs.front.x86.exe -v ./front
go build -o %GOPATH%/bin/dfs.node.x86.exe -v ./node
