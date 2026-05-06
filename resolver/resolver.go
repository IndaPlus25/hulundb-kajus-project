package resolver

import (
	"fmt"
	"hulundb-kajus-dns/dns"
	"math/rand"
	"net"
	"time"
)

func Resolve(query []byte, depth int) ([]byte, error) {

	if depth > 10 {
		return nil, fmt.Errorf("too many redirects")
	}

	target, err := queryRootServers(query)
	if err != nil {
		return nil, err
	}

	for {
		response, err := sendDNS(query, target)
		if err != nil {
			return nil, err
		}

		// Decode answer
		msg, err := dns.DecodeMessage(response)
		if err != nil {
			return nil, err
		}

		for _, rr := range msg.Answers {
			if rr.Header.Type == msg.Questions[0].Type {
				// Found the answer type we asked for
				return response, nil
			}
			if rr.Header.Type == 5 { // CNAME
				return nil, nil // TODO: Hugo CNAME
			}
		}

		// Try 1 — IP?
		nextTarget := ""
		for _, rr := range msg.Additionals {
			if a, ok := rr.Data.(dns.ARecord); ok {
				nextTarget = a.IP.String()
				break
			}
		}

		// Try 2 — Name?
		if nextTarget == "" {
			for _, rr := range msg.Authorities {
				if ns, ok := rr.Data.(dns.NSRecord); ok {
					nsQuery, err := buildQuery(ns.Name, 1)
					if err != nil {
						continue
					}

					nsResponse, err := Resolve(nsQuery, depth+1)
					if err != nil {
						continue
					}

					nsMsg, err := dns.DecodeMessage(nsResponse)
					if err != nil {
						continue
					}

					for _, nsRR := range nsMsg.Answers {
						if a, ok := nsRR.Data.(dns.ARecord); ok {
							nextTarget = a.IP.String()
							break
						}
					}
					if nextTarget != "" {
						break
					}
				}
			}
		}

		//Nothing was found
		if nextTarget == "" {
			return nil, fmt.Errorf("could not find next nameserver")
		}

		target = nextTarget
	}
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

func buildQuery(name string, qtype uint16) ([]byte, error) {
	id := uint16(rand.Intn(65536))
	msg := dns.Message{
		Header: dns.Header{
			ID:      id,    // random id
			QR:      false, // query
			Opcode:  0,     // standard query
			RD:      false, // no recursion desired
			QDCount: 1,
			ANCount: 0,
			NSCount: 0,
			ARCount: 0,
		},
		Questions: []dns.Question{
			{Name: name, Type: qtype, Class: 1},
		},
		Answers:     []dns.RR{},
		Authorities: []dns.RR{},
		Additionals: []dns.RR{},
	}

	return msg.Encode()
}
