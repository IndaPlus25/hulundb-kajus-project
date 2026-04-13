package resolver

import (
	"net"
	"time"
)

func Resolv(query []byte) ([]byte, error) {

	//Connects to Googles DNS
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	//Sends the question
	conn.Write(query)

	//Deadline for reading answer
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	//Recieves the answer
	buf := make([]byte, 512)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	return buf[:n], nil

}
