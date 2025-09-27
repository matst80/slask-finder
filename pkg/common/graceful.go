package common

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// ShutdownHook is a function executed after a termination signal is received
// but before the HTTP server begins its graceful shutdown. If a hook returns
// an error it will be logged; shutdown continues regardless.
type ShutdownHook func(ctx context.Context) error

// RunServerWithShutdown starts the provided *http.Server and blocks until a termination
// signal (SIGINT or SIGTERM) is received. It then runs any provided hooks (in order)
// with a context that shares the overall shutdown deadline, and finally gracefully
// shuts down the server. A per-hook timeout can be specified; if zero a default is used.
//
// Parameters:
//
//	server: configured *http.Server (Addr, Handler, timeouts etc.)
//	startupLog: message to log when the server starts listening
//	shutdownTimeout: total time allowed for all hooks + server shutdown
//	hookTimeout: individual timeout for each hook (if <=0 defaults to 5s)
//	hooks: optional ordered list of shutdown hooks
//
// Typical usage in main:
//
//	server := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 5*time.Second}
//	common.RunServerWithShutdown(server, "my service", 15*time.Second, 5*time.Second, saveHook)
func RunServerWithShutdown(server *http.Server, startupLog string, shutdownTimeout, hookTimeout time.Duration, hooks ...ShutdownHook) {
	if hookTimeout <= 0 {
		hookTimeout = 5 * time.Second
	}

	// Start server in goroutine
	go func() {
		log.Printf("starting %s on %s", startupLog, server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("%s listen error: %v", startupLog, err)
		}
	}()

	// Wait for signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Printf("shutdown signal received for %s", startupLog)

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Run hooks sequentially with individual timeouts
	for i, h := range hooks {
		if h == nil {
			continue
		}
		hCtx, hCancel := context.WithTimeout(ctx, hookTimeout)
		if err := h(hCtx); err != nil {
			log.Printf("shutdown hook %d failed: %v", i, err)
		}
		hCancel()
		if err := hCtx.Err(); err == context.DeadlineExceeded {
			log.Printf("shutdown hook %d timed out", i)
		}
	}

	// Finally shut down the HTTP server
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	} else {
		log.Printf("%s shutdown complete", startupLog)
	}
}

// TimeoutConfig holds server and shutdown related timeouts (all durations).
type TimeoutConfig struct {
	ReadHeader time.Duration
	Read       time.Duration
	Write      time.Duration
	Idle       time.Duration
	Shutdown   time.Duration
	Hook       time.Duration
}

// LoadTimeoutConfig reads environment variables (if present) to override defaults.
// Each env var is parsed as an integer number of seconds. If parsing fails or value <=0,
// the provided default is retained.
// Env variables:
//
//	READ_HEADER_TIMEOUT
//	READ_TIMEOUT
//	WRITE_TIMEOUT
//	IDLE_TIMEOUT
//	SHUTDOWN_TIMEOUT
//	HOOK_TIMEOUT
func LoadTimeoutConfig(defaults TimeoutConfig) TimeoutConfig {
	apply := func(curr *time.Duration, env string) {
		if v := os.Getenv(env); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				*curr = time.Duration(n) * time.Second
			}
		}
	}
	apply(&defaults.ReadHeader, "READ_HEADER_TIMEOUT")
	apply(&defaults.Read, "READ_TIMEOUT")
	apply(&defaults.Write, "WRITE_TIMEOUT")
	apply(&defaults.Idle, "IDLE_TIMEOUT")
	apply(&defaults.Shutdown, "SHUTDOWN_TIMEOUT")
	apply(&defaults.Hook, "HOOK_TIMEOUT")
	return defaults
}

// NewServerWithTimeouts attaches timeout settings to an existing *http.Server or creates a new one if nil.
func NewServerWithTimeouts(base *http.Server, cfg TimeoutConfig) *http.Server {
	if base == nil {
		return &http.Server{
			ReadHeaderTimeout: cfg.ReadHeader,
			ReadTimeout:       cfg.Read,
			WriteTimeout:      cfg.Write,
			IdleTimeout:       cfg.Idle,
		}
	}
	base.ReadHeaderTimeout = cfg.ReadHeader
	base.ReadTimeout = cfg.Read
	base.WriteTimeout = cfg.Write
	base.IdleTimeout = cfg.Idle
	return base
}
