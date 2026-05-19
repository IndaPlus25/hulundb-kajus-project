package main

import (
	"fmt"
	"hulundb-kajus-dns/cache"
	"hulundb-kajus-dns/resolver"
	"hulundb-kajus-dns/server"
	"hulundb-kajus-dns/web"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Initialize cache and resolver from dev branch
	c := cache.NewCache()
	c.StartEviction(60 * time.Second)
	r := resolver.New(c)

	// Error channel for both servers
	errs := make(chan error, 2)

	// Start DNS server
	go func() {
		errs <- server.Start("0.0.0.0:53", r)
	}()

	// Start web interface
	go func() {
		errs <- web.Start("0.0.0.0:8080")
	}()

	// Signal handling for graceful shutdown
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		fmt.Println("\nShutting down...")
		os.Exit(0)
	}()

	// Wait for error from either server
	err := <-errs
	if err != nil {
		fmt.Println("Server error:", err)
		os.Exit(1)
	}
}
