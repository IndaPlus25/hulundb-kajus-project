package main

import (
	"fmt"
	"hulundb-kajus-dns/server"
	"hulundb-kajus-dns/web"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	errs := make(chan error, 2)

	go func() {
		errs <- server.Start("0.0.0.0:53")
	}()

	go func() {
		errs <- web.Start("0.0.0.0:8080")
	}()

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		fmt.Println("\nShutting down...")
		os.Exit(0)
	}()

	err := <-errs
	if err != nil {
		fmt.Println("Server error:", err)
		os.Exit(1)
	}
}
