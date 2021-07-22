package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/schwarzlichtbezirk/dfs/pb"
	"google.golang.org/grpc"
)

var (
	// channel to indicate about server shutdown
	exitchan = make(chan struct{})
	// wait group for all server goroutines
	exitwg sync.WaitGroup
)

// Run launches server listeners.
func Run() {
	var err error

	// get port
	if err = DetectPort(); err != nil {
		log.Printf("used default port %s", cfg.PortGRPC)
	}

	// starts gRPC servers
	exitwg.Add(1)
	go func() {
		defer exitwg.Done()

		var err error
		var lis net.Listener

		log.Printf("grpc server %s starts\n", cfg.PortGRPC)
		if lis, err = net.Listen("tcp", cfg.PortGRPC); err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		var server = grpc.NewServer()
		pb.RegisterDataGuideServer(server, &routeDataGuideServer{addr: cfg.PortGRPC})
		go func() {
			if err = server.Serve(lis); err != nil {
				log.Fatalf("failed to serve: %v", err)
			}
		}()

		// wait for exit signal
		<-exitchan

		server.GracefulStop()

		log.Printf("grpc server %s closed\n", cfg.PortGRPC)
	}()

	log.Printf("ready")
}

// WaitBreak blocks goroutine until Ctrl+C will be pressed.
func WaitBreak() {
	var sigint = make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C) or SIGTERM (Ctrl+/)
	// SIGKILL, SIGQUIT will not be caught.
	signal.Notify(sigint, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive our signal.
	<-sigint
	// Make exit signal.
	close(exitchan)
}

// WaitExit performs graceful network shutdown,
// waits until all server threads will be stopped.
func WaitExit() {
	exitwg.Wait()
}
