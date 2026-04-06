package main

import (
	"fmt"
	"hulundb-kajus-dns/server"
	"os"
)

func main() {
	if err := server.Start("0.0.0.0:5353"); err != nil {
		fmt.Println("Failed to start server:", err)
		os.Exit(1)
	}
}
