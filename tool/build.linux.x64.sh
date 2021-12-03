#!/bin/bash
go env -w GOOS=linux GOARCH=amd64
cd $GOPATH/src/github.com/schwarzlichtbezirk/dfs
go build -o $GOPATH/bin/dfs.front.x64 -v ./front
go build -o $GOPATH/bin/dfs.node.x64 -v ./node
cp -r -u ./config $GOPATH/bin/dfs-config
