package resolver

import (
	"fmt"
	"hulundb-kajus-dns/dns"
	"net"
	"time"
)

func Resolve(query []byte, depth int) ([]byte, error) {

	target, err := queryRootServers(query)
	if err != nil {
		return nil, err
	}

	response, err := sendDNS(query, target)
	if err != nil {
		return nil, err
	}

	// Decode answer
	msg, err := dns.DecodeMessage(response)
	if err != nil {
		return nil, err
	}

	if len(msg.Answers) > 0 {
		for _, rr := range msg.Answers {
			if rr.Header.Type == msg.Questions[0].Type {
				// Found the answer type we asked for
				return response, nil
			}
			if rr.Header.Type == 5 { // CNAME
				return nil, nil // TODO: Hugo CNAME
			}
		}
	}
	if len(msg.Answers) == 0 {
		// Försök 1 — finns IP:n direkt?
		for _, rr := range msg.Additionals {
			if a, ok := rr.Data.(dns.ARecord); ok {
				target = a.IP.String()
				break
			}
		}

		// Försök 2 — finns bara namnet?
		if target == "" {
			for _, rr := range msg.Authorities {
				if ns, ok := rr.Data.(dns.NSRecord); ok {
					// resolve ns.Name för att få IP
				}
			}
		}

		// Inget hittades alls
		if target == "" {
			return nil, fmt.Errorf("could not find next nameserver")
		}
	}

	return Resolve(query, depth+1)

}

func queryRootServers(query []byte) (string, error) {
	for _, root := range RootServers {
		_, err := sendDNS(query, root.IPv4)
		if err != nil {
			continue // test next
		}
		return root.IPv4, nil
	}
	return "", fmt.Errorf("no root server responded")
}

func sendDNS(query []byte, target string) ([]byte, error) {
	//Connects to Root
	conn, err := net.Dial("udp", target+":53")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	//Sends the query
	_, err = conn.Write(query)
	if err != nil {
		return nil, err
	}

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
