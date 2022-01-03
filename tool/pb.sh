#!/bin/bash
export pbdir=$GOPATH/src/github.com/schwarzlichtbezirk/dfs/pb
protoc -I=$pbdir\
 --go_out=$pbdir --go_opt paths=source_relative\
 --go-grpc_out=$pbdir --go-grpc_opt paths=source_relative\
 $pbdir/dfs.proto
