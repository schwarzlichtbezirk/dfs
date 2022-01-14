#!/bin/bash
wsdir=$(dirname $0)/..
protoc --proto_path=$wsdir\
 --go_out=$wsdir --go_opt paths=source_relative\
 --go-grpc_out=$wsdir --go-grpc_opt paths=source_relative\
 $wsdir/pb/dfs.proto
