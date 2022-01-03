package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/schwarzlichtbezirk/dfs/pb"
)

// API error codes.
// Each error code have unique source code point,
// so this error code at service reply exactly points to error place.
const (
	AECnull = iota
	AECbadbody
	AECnoreq
	AECbadjson
	AECpanic

	// upload
	AECuploadform
	AECuploadwrite
	AECuploadbuf1
	AECuploadsend1
	AECuploadbuf2
	AECuploadsend2
	AECuploadreply

	// download
	AECdownloadbadid
	AECdownloadnoarg
	AECdownloadabsent

	// fileinfo
	AECfileinfonoarg

	// remove
	AECremovenoarg
	AECremoveabsent
	AECremovegrpc

	// clear
	AECcleargrpc

	// addnode
	AECaddnodenodata
	AECaddnodehas
)

// HTTP error messages
var (
	ErrNoJSON   = errors.New("data not given")
	ErrNoData   = errors.New("data is empty")
	ErrNotFound = errors.New("404 file not found")

	ErrArgBadID = errors.New("file ID can not be parsed as an integer")
	ErrNodeHas  = errors.New("node with given addres already present")
)

// pingAPI is ping helper to check transactions latency and webserver health.
func pingAPI(w http.ResponseWriter, r *http.Request) {
	var body, _ = io.ReadAll(r.Body)
	w.WriteHeader(http.StatusOK)
	WriteJSONHeader(w)
	w.Write(body)
}

// nodesizeAPI returns array with sum size of all chunks on each nodes.
func nodesizeAPI(w http.ResponseWriter, r *http.Request) {
	storage.nodmux.RLock()
	var ret = make([]int64, len(storage.Nodes))
	for i, node := range storage.Nodes {
		ret[i] = node.SumSize
	}
	storage.nodmux.RUnlock()

	WriteOK(w, ret)
}

// uploadAPI uploads some file.
func uploadAPI(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)

	var file, handler, err = r.FormFile("datafile")
	if err != nil {
		WriteError400(w, err, AECuploadform)
		return
	}
	defer file.Close()

	var info = storage.MakeFileInfo(handler)
	log.Printf("upload file: %s, size: %d, mime: %s\n", handler.Filename, handler.Size, info.MIME)

	storage.nodmux.RLock()
	var nn = int64(len(storage.Nodes)) // nodes number
	storage.nodmux.RUnlock()
	var cn int64 // chunks number
	var cr int64 // chunks remainder
	if cfg.MinNodeChunkSize == 0 {
		cn = 1000000 // any maximum possible value
	} else {
		cn = handler.Size / cfg.MinNodeChunkSize
		cr = handler.Size % cfg.MinNodeChunkSize
		if cr > 0 {
			cn++
		}
	}
	if cn <= nn {
		info.Chunks = make([]*pb.Range, cn)
		for i := int64(0); i < cn; i++ {
			info.Chunks[i] = &pb.Range{
				NodeId: i,
				FileId: info.FileID,
				From:   cfg.MinNodeChunkSize * i,
				To:     cfg.MinNodeChunkSize * (i + 1),
			}
		}
		// last chunk will have remainder
		if cr > 0 {
			var last = info.Chunks[cn-1]
			last.To = last.From + cr
		}
	} else if cfg.NodeFluidFill && nn > 1 {
		var sizes = make([]int64, nn)
		var volume int64
		storage.nodmux.RLock()
		for i := int64(0); i < nn; i++ {
			sizes[i] = storage.Nodes[i].SumSize
			volume += sizes[i]
		}
		storage.nodmux.RUnlock()

		// calculate fluid chunk sizes
		var fsum int64
		for i := int64(0); i < nn; i++ {
			var percent float64
			if volume > 0 {
				percent = float64(sizes[i]) / float64(volume)
			} else {
				percent = 1 / float64(nn)
			}
			var portion = (1 - percent) / float64(nn-1)
			sizes[i] = int64(float64(handler.Size) * portion)
			fsum += sizes[i]
			log.Printf("node#%d, portion=%f, size=%d", i, portion, sizes[i])
		}
		// store remainder to first node
		if fsum < handler.Size {
			sizes[0] += handler.Size - fsum
		} else if fsum > handler.Size {
			// there is something wrong
			panic("negative remainder received for file " + handler.Filename)
		}

		var pos int64
		info.Chunks = make([]*pb.Range, nn)
		for i := int64(0); i < nn; i++ {
			info.Chunks[i] = &pb.Range{
				NodeId: i,
				FileId: info.FileID,
				From:   pos,
				To:     pos + sizes[i],
			}
			pos += sizes[i]
		}
	} else {
		info.Chunks = make([]*pb.Range, nn)
		var cs = handler.Size / nn // chunk size
		for i := int64(0); i < nn; i++ {
			info.Chunks[i] = &pb.Range{
				NodeId: i,
				FileId: info.FileID,
				From:   cs * i,
				To:     cs * (i + 1),
			}
		}
		// last chunk will have remainder
		var last = info.Chunks[nn-1]
		last.To += handler.Size % nn
	}

	// send to nodes
	// sequential algorithm is faster than with parallelism
	// on files for several MB and nodes on same hardware
	// to make parallelism uncomment lines in followed code

	var errs = make([]error, len(info.Chunks))
	//var fmux sync.Mutex
	//var wg sync.WaitGroup
	//wg.Add(len(info.Chunks))
	func() {
		for i, rng := range info.Chunks {
			var i = i     // localize
			var rng = rng // localize
			//go func() {
			//defer wg.Done()

			var err error
			var ctx = context.Background() // no any limits
			var stream pb.DataGuide_WriteClient
			storage.nodmux.RLock()
			var node = storage.Nodes[rng.NodeId]
			storage.nodmux.RUnlock()
			if stream, err = node.Client.Write(ctx); err != nil {
				errs[i] = MakeAjaxErr(err, AECuploadwrite)
				return
			}
			var cs = rng.To - rng.From
			var cn = cs / cfg.StreamChunkSize
			var cr = cs % cfg.StreamChunkSize
			var buf = make([]byte, cfg.StreamChunkSize)
			// write serie of chunks
			for j := int64(0); j < cn; j++ {
				//fmux.Lock()
				file.Seek(rng.From+j*cfg.StreamChunkSize, io.SeekStart)
				_, err = file.Read(buf)
				//fmux.Unlock()

				if err != nil {
					errs[i] = MakeAjaxErr(err, AECuploadbuf1)
					return
				}
				var chunk = pb.Chunk{
					Range: &pb.Range{
						FileId: rng.FileId,
						NodeId: rng.NodeId,
						From:   rng.From + j*cfg.StreamChunkSize,
						To:     rng.From + (j+1)*cfg.StreamChunkSize,
					},
					Value: buf,
				}
				if err := stream.Send(&chunk); err != nil {
					errs[i] = MakeAjaxErr(err, AECuploadsend1)
					return
				}
			}
			// write remainder
			if cr > 0 {
				buf = buf[:cr]
				//fmux.Lock()
				file.Seek(rng.From+cn*cfg.StreamChunkSize, io.SeekStart)
				_, err = file.Read(buf)
				//fmux.Unlock()

				if err != nil {
					errs[i] = MakeAjaxErr(err, AECuploadbuf2)
					return
				}
				var chunk = pb.Chunk{
					Range: &pb.Range{
						FileId: rng.FileId,
						NodeId: rng.NodeId,
						From:   rng.From + cn*cfg.StreamChunkSize,
						To:     rng.From + cn*cfg.StreamChunkSize + cr,
					},
					Value: buf,
				}
				if err := stream.Send(&chunk); err != nil {
					errs[i] = MakeAjaxErr(err, AECuploadsend2)
					return
				}
			}
			var reply *pb.Summary
			if reply, err = stream.CloseAndRecv(); err != nil {
				errs[i] = MakeAjaxErr(err, AECuploadreply)
				return
			}
			log.Printf("chunk %d, size %d, time %v", i, cs, time.Duration(reply.ElapsedTime))
			//}()
		}
	}()
	//wg.Wait()

	// check for error at any thread
	for _, err := range errs {
		if err != nil {
			// try to remove all stored chunks to prevent garbage accumulation
			for _, rng := range info.Chunks {
				ctx, cancel := context.WithTimeout(context.Background(), cfg.ApiTimeout)
				defer cancel()
				storage.nodmux.RLock()
				var node = storage.Nodes[rng.NodeId]
				storage.nodmux.RUnlock()
				// do not get a new error, it's already failed state
				node.Client.Remove(ctx, &pb.FileID{Id: rng.FileId})
			}
			// write error 500
			WriteJSON(w, http.StatusInternalServerError, err)
			return
		}
	}

	// save file information at last to get ready for full access after it
	storage.AddFileInfo(info)

	WriteOK(w, info)
}

func downloadAPI(w http.ResponseWriter, r *http.Request) {
	var err error

	// get arguments
	var fid int64
	if s := r.FormValue("id"); len(s) > 0 {
		if fid, err = strconv.ParseInt(s, 10, 64); err != nil {
			WriteError400(w, ErrArgBadID, AECdownloadbadid)
			return
		}
	}
	var name string
	if s := r.FormValue("name"); len(s) > 0 {
		name = s
	}

	if fid == 0 && name == "" {
		WriteError400(w, ErrNoData, AECdownloadnoarg)
		return
	}

	var info *FileInfo
	if info = storage.FindFileInfo(fid, name); info == nil {
		WriteError(w, http.StatusNotFound, ErrNotFound, AECdownloadabsent)
		return
	}

	w.Header().Set("Content-Type", info.MIME)
	http.ServeContent(w, r, info.Name, time.Time{}, storage.NewReader(info))
}

func fileinfoAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		Name string `json:"name,omitempty"`
		ID   int64  `json:"id,omitempty"`
	}
	var ret *FileInfo

	// get arguments
	if err = AjaxGetArg(w, r, &arg); err != nil {
		return
	}
	if arg.ID == 0 && arg.Name == "" {
		WriteError400(w, ErrNoData, AECfileinfonoarg)
		return
	}

	ret = storage.FindFileInfo(arg.ID, arg.Name)

	WriteOK(w, ret)
}

// removeAPI deletes all chunks of pointed file from nodes.
// returns file info of removed file.
func removeAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		Name string `json:"name,omitempty"`
		ID   int64  `json:"id,omitempty"`
	}
	var ret *FileInfo

	// get arguments
	if err = AjaxGetArg(w, r, &arg); err != nil {
		return
	}
	if arg.ID == 0 && arg.Name == "" {
		WriteError400(w, ErrNoData, AECremovenoarg)
		return
	}

	if ret = storage.FindFileInfo(arg.ID, arg.Name); ret == nil {
		WriteError(w, http.StatusNotFound, ErrNotFound, AECremoveabsent)
		return
	}

	storage.DelFileInfo(ret) // file data can not be accessed after it
	// try to remove all chunks
	for _, rng := range ret.Chunks {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.ApiTimeout)
		defer cancel()
		storage.nodmux.RLock()
		var node = storage.Nodes[rng.NodeId]
		storage.nodmux.RUnlock()
		if _, err1 := node.Client.Remove(ctx, &pb.FileID{Id: rng.FileId}); err1 != nil {
			err = err1 // save error for future break
		}
	}
	if err != nil {
		WriteError500(w, err, AECremovegrpc)
		return
	}

	WriteOK(w, ret)
}

// clearAPI deletes all data at storage, purge nodes, and sets files ID counter to 0.
func clearAPI(w http.ResponseWriter, r *http.Request) {
	var err error

	storage.Clear()

	// Purge all nodes in locked state.
	// So users can not add some files or do anything during this operation.
	func() {
		// Unlock put to defer, if panic will be caused
		storage.nodmux.Lock()
		defer storage.nodmux.Unlock()

		// Try to purge all nodes
		for _, node := range storage.Nodes {
			ctx, cancel := context.WithTimeout(context.Background(), cfg.ApiTimeout)
			defer cancel()
			if _, err1 := node.Client.Purge(ctx, &pb.Empty{}); err1 != nil {
				err = err1 // save error for future break
			}
		}
	}()

	if err != nil {
		WriteError500(w, err, AECcleargrpc)
		return
	}

	log.Println("content is cleared")

	WriteOK(w, nil)
}

// addnodeAPI adds new node to composition in runtime and waits until connection will be established.
func addnodeAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		Addr string `json:"addr"`
	}
	var ret int // index of added node

	// get arguments
	if err = AjaxGetArg(w, r, &arg); err != nil {
		return
	}
	if arg.Addr == "" {
		WriteError400(w, ErrNoData, AECaddnodenodata)
		return
	}

	var found bool
	storage.nodmux.RLock()
	for _, node := range storage.Nodes {
		if node.Addr == arg.Addr {
			found = true
			break
		}
	}
	storage.nodmux.RUnlock()
	if found {
		WriteError400(w, ErrNodeHas, AECaddnodehas)
		return
	}

	var node = &NodeInfo{
		Addr:    arg.Addr,
		SumSize: 0,
	}

	storage.nodmux.Lock()
	ret = len(storage.Nodes) // get size, it will be index
	storage.Nodes = append(storage.Nodes, node)
	storage.nodmux.Unlock()

	node.RunGRPC()
	grpcwg.Wait()

	WriteOK(w, ret)
}
