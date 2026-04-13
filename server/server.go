package server

import (
	"fmt"
	"hulundb-kajus-dns/resolver"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func Start(addr string) error {

	//Listening on addr
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	fmt.Println("DNS server listening on", addr)

	//Closes server right
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		fmt.Println("Shutting down...")
		conn.Close()
	}()

	//Receives, sends to resolver, sends back
	buf := make([]byte, 512)
	for {
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return nil
			}
			fmt.Println("Error reading:", err)
			continue
		}

		fmt.Printf("Received %d bytes from %s\n", n, addr)

		// TODO: parser (hulundb)

		// TODO: forwarding (issue 2)
		response, err := resolver.Resolv(buf[:n])
		if err != nil {
			fmt.Println("Error resolving:", err)
			continue
		}
		_, err = conn.WriteTo(response, addr)
		if err != nil {
			fmt.Println("Error writing response:", err)
			return err
		}
	}
}
