package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/schwarzlichtbezirk/dfs/pb"
	"google.golang.org/grpc"
)

var (
	// channel to indicate about server shutdown
	exitchan = make(chan struct{})
	// wait group for all server goroutines
	exitwg sync.WaitGroup
)

var (
	grpcClient []pb.DataGuideClient
)

// Run launches server listeners.
func Run(gmux *Router) {
	var err error

	// get confiruration path
	if err = DetectConfigPath(); err != nil {
		log.Fatal(err)
	}
	log.Printf("config path: %s\n", ConfigPath)

	// load content of Config structure from YAML-file.
	if err = ReadYaml(cfgfile, &cfg); err != nil {
		log.Fatalf("can not read '%s' file: %v\n", cfgfile, err)
	}
	log.Printf("loaded '%s'\n", cfgfile)

	if err = ReadYaml(nodesfile, &nodes); err != nil {
		log.Fatal(err)
	}
	log.Printf("total %d nodes\n", len(nodes))
	grpcClient = make([]pb.DataGuideClient, len(nodes))

	// starts gRPC clients
	var grpcwg sync.WaitGroup
	for i, addr := range nodes {
		var i = i
		var addr = addr

		exitwg.Add(1)
		grpcwg.Add(1)
		go func() {
			defer exitwg.Done()

			var err error
			var conn *grpc.ClientConn
			var ok bool

			func() {
				defer grpcwg.Done()

				log.Printf("grpc connecting on %s\n", addr)
				var ctx, cancel = context.WithCancel(context.Background())
				defer cancel()

				if conn, err = grpc.DialContext(ctx, addr, grpc.WithInsecure()); err != nil {
					log.Fatalf("fail to dial on %s: %v", addr, err)
				}
				// wait until connect will be established or have got exit signal
				select {
				case <-ctx.Done():
					grpcClient[i] = pb.NewDataGuideClient(conn)
					log.Printf("grpc connected on %s\n", addr)
					ok = true
				case <-exitchan:
					log.Printf("grpc canceled for %s\n", addr)
				}
			}()

			if ok {
				// wait for exit signal
				<-exitchan

				if err = conn.Close(); err != nil {
					log.Printf("grpc disconnect on %s: %v\n", addr, err)
				} else {
					log.Printf("grpc disconnected on %s\n", addr)
				}
			}
		}()
	}
	// wait until all grpc starts
	grpcwg.Wait()

	// check on exit during grpc connecting
	select {
	case <-exitchan:
		return
	default:
	}

	// starts HTTP servers
	for _, addr := range cfg.PortHTTP {
		var addr = envfmt(addr) // localize
		exitwg.Add(1)
		go func() {
			defer exitwg.Done()

			var server = &http.Server{
				Addr:              addr,
				Handler:           gmux,
				ReadTimeout:       time.Duration(cfg.ReadTimeout) * time.Second,
				ReadHeaderTimeout: time.Duration(cfg.ReadHeaderTimeout) * time.Second,
				WriteTimeout:      time.Duration(cfg.WriteTimeout) * time.Second,
				IdleTimeout:       time.Duration(cfg.IdleTimeout) * time.Second,
				MaxHeaderBytes:    cfg.MaxHeaderBytes,
			}
			go func() {
				log.Printf("start http on %s\n", addr)
				if err := server.ListenAndServe(); err != http.ErrServerClosed {
					log.Fatalf("failed to serve: %v", err)
				}
			}()

			// wait for exit signal
			<-exitchan

			// create a deadline to wait for.
			var ctx, cancel = context.WithTimeout(
				context.Background(),
				time.Duration(cfg.ShutdownTimeout)*time.Second)
			defer cancel()

			server.SetKeepAlivesEnabled(false)
			if err := server.Shutdown(ctx); err != nil {
				log.Printf("shutdown http on %s: %v\n", addr, err)
			} else {
				log.Printf("stop http on %s\n", addr)
			}
		}()
	}

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
