@echo off

set wsdir=%~dp0..
set pbdir=%wsdir%/pb
set apidir=%wsdir%/api/export

protoc -I=%apidir%^
 --go_out %pbdir%^
 --go_opt paths=source_relative^
 --go-grpc_out %pbdir%^
 --go-grpc_opt paths=source_relative^
 dfs.proto
