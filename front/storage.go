package main

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"sync"
	"sync/atomic"

	"github.com/schwarzlichtbezirk/dfs/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/grpclog"
)

// FileInfo is file information about chunks placed at nodes.
type FileInfo struct {
	FileID int64       `json:"file_id"`
	Name   string      `json:"name"`
	Size   int64       `json:"size"`
	MIME   string      `json:"mime"`
	Chunks []*pb.Range `json:"chunks"`
}

type NodeInfo struct {
	// Client is gRPC client.
	Client pb.DataGuideClient
	// Addr is client address:port, used for read-only after initialization.
	Addr string
	// SumSize is total size of all chunks saved on node, atomic increments.
	SumSize int64
	// NumChunks is number of chunks saved on node.
	NumChunks int
}

type Storage struct {
	// idconter is files ID counter.
	// Each stored file will have unique ID, and can have not unique file name.
	idconter int64
	// Nodes is list of available nodes with information about them.
	Nodes []*NodeInfo
	// mutex for Nodes array access.
	nodmux sync.RWMutex
	// FIMap is files database with fileID/FileInfo keys/values.
	FIMap sync.Map
}

// Storage is singleton
var storage Storage

// RunGRPC establishes gRPC connection for given node.
func (node *NodeInfo) RunGRPC() {
	grpcwg.Add(1)
	exitwg.Add(1)
	go func() {
		defer exitwg.Done()

		var conn *grpc.ClientConn
		var err error
		var ctx, cancel = context.WithCancel(context.Background())
		go func() {
			defer grpcwg.Done()
			defer cancel()

			grpclog.Infof("grpc connection wait on %s\n", node.Addr)
			var options = []grpc.DialOption{
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithBlock(),
			}
			conn, err = grpc.DialContext(ctx, node.Addr, options...)
			node.Client = pb.NewDataGuideClient(conn)
		}()
		// wait until connect will be established or have got exit signal
		select {
		case <-ctx.Done():
		case <-exitctx.Done():
			grpclog.Infof("grpc connection canceled on %s\n", node.Addr)
			return
		}

		if err != nil {
			grpclog.Errorf("fail to dial on %s: %v", node.Addr, err)
			exitfn()
			return
		}
		grpclog.Infof("grpc connection established on %s\n", node.Addr)

		// wait for exit signal
		<-exitctx.Done()

		if err := conn.Close(); err != nil {
			grpclog.Errorf("grpc disconnect on %s: %v\n", node.Addr, err)
		} else {
			grpclog.Infof("grpc disconnected on %s\n", node.Addr)
		}
	}()
}

func (s *Storage) MakeFileInfo(handler *multipart.FileHeader) (info *FileInfo) {
	// make file ID
	var fid = atomic.AddInt64(&s.idconter, 1)
	// extract MIME type
	var mime = "N/A"
	if ct, ok := handler.Header["Content-Type"]; ok && len(ct) > 0 {
		mime = ct[0]
	}
	// inits file info
	info = &FileInfo{
		FileID: fid,
		Name:   handler.Filename,
		Size:   handler.Size,
		MIME:   mime,
	}
	return
}

func (s *Storage) NewReader(fi *FileInfo) io.ReadSeeker {
	return &NodesReader{s, fi, 0}
}

// AddFileInfo adds file information to nodes storage.
func (s *Storage) AddFileInfo(fi *FileInfo) {
	// update statistics
	s.nodmux.Lock()
	for _, rng := range fi.Chunks {
		s.Nodes[rng.NodeId].NumChunks++
		s.Nodes[rng.NodeId].SumSize += rng.To - rng.From
	}
	s.nodmux.Unlock()

	// add itself
	s.FIMap.Store(fi.FileID, fi)
}

// DelFileInfo deletes file information from nodes storage.
func (s *Storage) DelFileInfo(fi *FileInfo) {
	// delete itself
	s.FIMap.Delete(fi.FileID)

	// update statistics
	s.nodmux.Lock()
	for _, rng := range fi.Chunks {
		s.Nodes[rng.NodeId].NumChunks--
		s.Nodes[rng.NodeId].SumSize -= rng.To - rng.From
	}
	s.nodmux.Unlock()
}

// Clear performs safe and quick delete of all stored data.
func (s *Storage) Clear() {
	// below assignment to absolute values, so lock performs to whole content
	s.nodmux.Lock()
	defer s.nodmux.Unlock()

	// reset files info map
	s.FIMap = sync.Map{}

	// update statistics
	for _, node := range s.Nodes {
		node.NumChunks = 0
		node.SumSize = 0
	}

	// no needs for atomic on locked content
	s.idconter = 0
}

// FindByName returns ID of first founded file record with given name, or 0 if it is not found.
func (s *Storage) FindIdByName(name string) (fid int64) {
	s.FIMap.Range(func(key interface{}, value interface{}) bool {
		_ = key
		var fi = value.(*FileInfo)
		if fi.Name == name {
			fid = fi.FileID
			return false
		}
		return true
	})
	return
}

// FindFileInfo searches file record by given `fid`, or by `name` if `fid` is zero.
// Returns founded record, or nil if it not found.
func (s *Storage) FindFileInfo(fid int64, name string) (info *FileInfo) {
	if fid == 0 {
		fid = s.FindIdByName(name)
	}
	if data, ok := s.FIMap.Load(fid); ok {
		info = data.(*FileInfo)
	}
	return
}

var (
	ErrNRBadWhence = errors.New("NodesReader.Seek: invalid whence")
	ErrNRPosNeg    = errors.New("NodesReader.Seek: negative position")
	ErrNROffNeg    = errors.New("NodesReader.ReadAt: negative offset")
)

type NodesReader struct {
	storage *Storage
	info    *FileInfo
	pos     int64 // current reading index
}

// Size returns the original length of the file.
// Size is the number of bytes available for reading via ReadAt.
// The returned value is always the same and is not affected by calls
// to any other method.
func (r *NodesReader) Size() int64 {
	return r.info.Size
}

// Seek implements the io.Seeker interface.
func (r *NodesReader) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = r.pos + offset
	case io.SeekEnd:
		abs = r.info.Size + offset
	default:
		return 0, ErrNRBadWhence
	}
	if abs < 0 {
		return 0, ErrNRPosNeg
	}
	if abs > r.info.Size {
		return 0, io.EOF
	}
	r.pos = abs
	return abs, nil
}

// readRange reads chunk of file with given range, from `off` position to `end` position.
// Length of this range must not be larger than `b` length.
func (r *NodesReader) readRange(off, end int64, b []byte) (n int, err error) {
	for _, rng := range r.info.Chunks {
		if rng.From < end && rng.To > off {
			var from = off
			if rng.From > off {
				from = rng.From
			}
			var to = end
			if rng.To < end {
				to = rng.To
			}
			var ctx = context.Background() // no any limits
			var in = &pb.Range{
				NodeId: rng.NodeId,
				FileId: rng.FileId,
				From:   from,
				To:     to,
			}
			var chunk *pb.Chunk
			r.storage.nodmux.RLock()
			var node = r.storage.Nodes[rng.NodeId]
			r.storage.nodmux.RUnlock()
			if chunk, err = node.Client.Read(ctx, in); err != nil {
				return
			}
			n += copy(b[chunk.Range.From-off:], chunk.Value)
		}
	}
	r.pos = end
	return
}

// Read implements the io.Reader interface.
func (r *NodesReader) Read(b []byte) (n int, err error) {
	if r.pos >= r.info.Size {
		return 0, io.EOF
	}

	var off = r.pos
	var end = off + int64(len(b))
	if end > r.info.Size {
		end = r.info.Size
	}
	return r.readRange(off, end, b)
}

// ReadAt implements the io.ReaderAt interface.
func (r *NodesReader) ReadAt(b []byte, off int64) (n int, err error) {
	// cannot modify state - see io.ReaderAt
	if off < 0 {
		return 0, ErrNROffNeg
	}
	if off >= r.info.Size {
		return 0, io.EOF
	}

	var end = off + int64(len(b))
	if end > r.info.Size {
		end = r.info.Size
	}
	if n, err = r.readRange(off, end, b); err != nil {
		return
	}
	if n < len(b) {
		err = io.EOF
	}
	return
}
