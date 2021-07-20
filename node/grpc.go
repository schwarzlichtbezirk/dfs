package main

import (
	"context"
	"io"
	"log"
	"sync"
	"time"

	"github.com/schwarzlichtbezirk/dfs/pb"
)

// Storage is singleton, files database
var storage sync.Map

type routeDataGuideServer struct {
	pb.UnimplementedDataGuideServer
	addr string
}

func (s *routeDataGuideServer) GetProp(ctx context.Context, arg *pb.FileID) (res *pb.Prop, err error) {
	if data, ok := storage.Load(arg.Id); ok {
		res = data.(pb.Chunk).Prop
		return
	}
	res = &pb.Prop{}
	return
}

func (s *routeDataGuideServer) Write(stream pb.DataGuide_WriteServer) error {
	var count int32
	var startTime = time.Now()
	for {
		var chunk, err = stream.Recv()
		if err == io.EOF {
			log.Printf("fetched %d items\n", count)
			var endTime = time.Now()
			return stream.SendAndClose(&pb.Summary{
				ChunkCount:  count,
				ElapsedTime: int32(endTime.Sub(startTime).Milliseconds()),
			})
		}
		if err != nil {
			return err
		}

		var data, ok = storage.LoadOrStore(chunk.Prop.Id, chunk)
		if ok {
			var file = data.(*pb.Chunk)
			file.Prop.To += int64(len(chunk.Value))
			file.Value = append(file.Value, chunk.Value...)
		}

		count++
	}
}
