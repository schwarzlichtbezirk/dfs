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
)

var (
	// channel to indicate about server shutdown
	exitchan chan struct{}
	// wait group for all server goroutines
	exitwg sync.WaitGroup
)

var (
	grpcClient pb.DataGuideClient
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
		log.Println(cfg)
		//log.Fatalf("can not read '%s' file: %v\n", cfgfile, err)
	}
	log.Printf("loaded '%s'\n", cfgfile)

	// inits exit channel
	exitchan = make(chan struct{})

	// helps to start HTTP only after initial load to prevent call to uninitialized data
	var dataready = make(chan struct{})

	for _, addr := range cfg.PortHTTP {
		var addr = addr // localize
		exitwg.Add(1)
		go func() {
			defer exitwg.Done()

			log.Printf("start http on %s\n", addr)
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
				if err := server.ListenAndServe(); err != http.ErrServerClosed {
					log.Fatalf("failed to serve on %s: %v", addr, err)
					return
				}
			}()

			// wait for exit signal
			<-exitchan

			// create a deadline to wait for.
			var ctx, cancel = context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
			defer cancel()

			server.SetKeepAlivesEnabled(false)
			if err := server.Shutdown(ctx); err != nil {
				log.Printf("shutdown http on %s: %v\n", addr, err)
			} else {
				log.Printf("stop http on %s\n", addr)
			}
		}()
	}

	// load initial list (database), and serve after it
	exitwg.Add(1)
	go func() {
		defer exitwg.Done()

		// data is ready, so HTTP can safely serve
		close(dataready)
	}()
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
