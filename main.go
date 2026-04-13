package main

import (
	"fmt"
	"hulundb-kajus-dns/server"
	"os"
)

func main() {
	err := server.Start("0.0.0.0:5355")

	if err != nil {
		fmt.Println("Failed to start server:", err)
		os.Exit(1)
	}
}
