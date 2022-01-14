package main

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/schwarzlichtbezirk/dfs/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

var (
	// context to indicate about service shutdown
	exitctx context.Context
	exitfn  context.CancelFunc
	// wait group for all server goroutines
	exitwg sync.WaitGroup
)

func init() {
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(os.Stdout, os.Stderr, os.Stderr))
}

// Init performs global data initialization.
func Init() {
	grpclog.Infoln("starts")

	// create context and wait the break
	exitctx, exitfn = context.WithCancel(context.Background())
	go func() {
		// Make exit signal on function exit.
		defer exitfn()

		var sigint = make(chan os.Signal, 1)
		var sigterm = make(chan os.Signal, 1)
		// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C) or SIGTERM (Ctrl+/)
		// SIGKILL, SIGQUIT will not be caught.
		signal.Notify(sigint, syscall.SIGINT)
		signal.Notify(sigterm, syscall.SIGTERM)
		// Block until we receive our signal.
		select {
		case <-exitctx.Done():
			if errors.Is(exitctx.Err(), context.DeadlineExceeded) {
				grpclog.Infoln("shutting down by timeout")
			} else if errors.Is(exitctx.Err(), context.Canceled) {
				grpclog.Infoln("shutting down by cancel")
			} else {
				grpclog.Infof("shutting down by %s", exitctx.Err().Error())
			}
		case <-sigint:
			grpclog.Infoln("shutting down by break")
		case <-sigterm:
			grpclog.Infoln("shutting down by process termination")
		}
		signal.Stop(sigint)
		signal.Stop(sigterm)
	}()
}

// Run launches server listeners.
func Run() {
	var grpcctx, grpccancel = context.WithCancel(context.Background())

	// starts gRPC servers
	exitwg.Add(1)
	go func() {
		defer exitwg.Done()

		grpclog.Infof("grpc server %s starts\n", cfg.PortGRPC)
		var err error
		var lis net.Listener
		if lis, err = net.Listen("tcp", cfg.PortGRPC); err != nil {
			grpclog.Fatalf("failed to listen: %v", err)
		}
		var server = grpc.NewServer()
		pb.RegisterDataGuideServer(server, &routeDataGuideServer{addr: cfg.PortGRPC})
		go func() {
			grpccancel()
			if err := server.Serve(lis); err != nil {
				grpclog.Fatalf("failed to serve: %v", err)
			}
		}()

		// wait for exit signal
		<-exitctx.Done()

		server.GracefulStop()

		grpclog.Infof("grpc server %s closed\n", cfg.PortGRPC)
	}()

	select {
	case <-grpcctx.Done():
		grpclog.Infoln("grpc ready")
	case <-exitctx.Done():
		return
	}
}

// Done performs graceful network shutdown,
// waits until all server threads will be stopped.
func Done() {
	// wait for exit signal
	<-exitctx.Done()
	// wait until all server threads will be stopped.
	exitwg.Wait()
	grpclog.Infoln("shutting down complete.")
}
