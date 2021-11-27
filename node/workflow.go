package main

import (
	"context"
	"errors"
	"flag"
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
	// context to indicate about service shutdown
	exitctx context.Context
	exitfn  context.CancelFunc
	// wait group for all server goroutines
	exitwg sync.WaitGroup
)

// Init performs global data initialisation.
func Init() {
	log.Println("starts")

	flag.Parse()

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
				log.Println("shutting down by timeout")
			} else if errors.Is(exitctx.Err(), context.Canceled) {
				log.Println("shutting down by cancel")
			} else {
				log.Printf("shutting down by %s", exitctx.Err().Error())
			}
		case <-sigint:
			log.Println("shutting down by break")
		case <-sigterm:
			log.Println("shutting down by process termination")
		}
		signal.Stop(sigint)
		signal.Stop(sigterm)
	}()

	// get port
	if err := DetectPort(); err != nil {
		log.Printf("used default port %s", cfg.PortGRPC)
	}
}

// Run launches server listeners.
func Run() {
	// starts gRPC servers
	exitwg.Add(1)
	go func() {
		defer exitwg.Done()

		log.Printf("grpc server %s starts\n", cfg.PortGRPC)
		var err error
		var lis net.Listener
		if lis, err = net.Listen("tcp", cfg.PortGRPC); err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		var server = grpc.NewServer()
		pb.RegisterDataGuideServer(server, &routeDataGuideServer{addr: cfg.PortGRPC})
		go func() {
			if err := server.Serve(lis); err != nil {
				log.Fatalf("failed to serve: %v", err)
			}
		}()

		// wait for exit signal
		<-exitctx.Done()

		server.GracefulStop()

		log.Printf("grpc server %s closed\n", cfg.PortGRPC)
	}()

	log.Println("ready")
}

// Done performs graceful network shutdown,
// waits until all server threads will be stopped.
func Done() {
	// wait for exit signal
	<-exitctx.Done()
	// wait until all server threads will be stopped.
	exitwg.Wait()
	log.Println("shutting down complete.")
}
