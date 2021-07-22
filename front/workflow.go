package main

import (
	"context"
	"log"
	"net/http"
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

	var nodes []string
	if err = ReadYaml(nodesfile, &nodes); err != nil {
		log.Fatal(err)
	}
	log.Printf("total %d nodes\n", len(nodes))
	Nodes = make([]NodeInfo, len(nodes))

	// starts gRPC clients
	var grpcwg sync.WaitGroup
	for i, addr := range nodes {
		var i = i
		var addr = addr

		exitwg.Add(1)
		grpcwg.Add(1)
		go func() {
			defer exitwg.Done()

			var conn *grpc.ClientConn
			var ok bool

			func() {
				defer grpcwg.Done()

				var err error
				log.Printf("grpc connection wait on %s\n", addr)
				var ctx, cancel = context.WithCancel(context.Background())
				go func() {
					defer cancel()
					if conn, err = grpc.DialContext(ctx, addr, grpc.WithInsecure(), grpc.WithBlock()); err != nil {
						log.Fatalf("fail to dial on %s: %v", addr, err)
					}
					Nodes[i].Client = pb.NewDataGuideClient(conn)
					Nodes[i].Addr = addr
					Nodes[i].SumSize = 0
				}()
				// wait until connect will be established or have got exit signal
				select {
				case <-ctx.Done():
					log.Printf("grpc connection established on %s\n", addr)
					ok = true
				case <-exitchan:
					log.Printf("grpc connection canceled on %s\n", addr)
				}
			}()

			if ok {
				defer conn.Close()
				// wait for exit signal
				<-exitchan

				if err := conn.Close(); err != nil {
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
				ReadTimeout:       cfg.ReadTimeout,
				ReadHeaderTimeout: cfg.ReadHeaderTimeout,
				WriteTimeout:      cfg.WriteTimeout,
				IdleTimeout:       cfg.IdleTimeout,
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
				cfg.ShutdownTimeout)
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
