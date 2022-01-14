package main

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/schwarzlichtbezirk/dfs/pb"
	"google.golang.org/grpc/grpclog"
)

// Storage is singleton, files database with fileID/Chunk keys/values.
var storage sync.Map

var (
	// ErrOutRange is "bounds out of the range" error message.
	ErrOutRange = errors.New("bounds out of the range")
)

type routeDataGuideServer struct {
	pb.UnimplementedDataGuideServer
	addr string
}

func (s *routeDataGuideServer) Read(ctx context.Context, arg *pb.Range) (res *pb.Chunk, err error) {
	if data, ok := storage.Load(arg.FileId); ok {
		var chunk = data.(*pb.Chunk)
		if arg.From < chunk.Range.From || arg.To > chunk.Range.To {
			err = ErrOutRange
			return
		}
		res = &pb.Chunk{
			Range: arg,
			Value: chunk.Value[arg.From-chunk.Range.From : arg.To-chunk.Range.From],
		}
	} else {
		res = &pb.Chunk{}
	}
	return
}

func (s *routeDataGuideServer) Write(stream pb.DataGuide_WriteServer) error {
	var count int32
	var startTime = time.Now()
	for {
		var chunk, err = stream.Recv()
		if err == io.EOF {
			grpclog.Infof("fetched %d items\n", count)
			var endTime = time.Now()
			return stream.SendAndClose(&pb.Summary{
				ChunkCount:  count,
				ElapsedTime: int64(endTime.Sub(startTime)),
			})
		}
		if err != nil {
			return err
		}

		var data, ok = storage.LoadOrStore(chunk.Range.FileId, chunk)
		if ok {
			var file = data.(*pb.Chunk)
			file.Range.To += int64(len(chunk.Value))
			file.Value = append(file.Value, chunk.Value...)
		}

		count++
	}
}

func (s *routeDataGuideServer) GetRange(ctx context.Context, arg *pb.FileID) (res *pb.Range, err error) {
	if data, ok := storage.Load(arg.Id); ok {
		res = data.(*pb.Chunk).Range
		return
	}
	res = &pb.Range{}
	return
}

func (s *routeDataGuideServer) Remove(ctx context.Context, arg *pb.FileID) (res *pb.Range, err error) {
	if data, ok := storage.LoadAndDelete(arg.Id); ok {
		res = data.(*pb.Chunk).Range
		return
	}
	res = &pb.Range{}
	return
}

func (s *routeDataGuideServer) Purge(ctx context.Context, arg *pb.Empty) (res *pb.Empty, err error) {
	storage = sync.Map{}
	res = &pb.Empty{}
	return
}
