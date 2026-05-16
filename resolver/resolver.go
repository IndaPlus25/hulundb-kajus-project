package resolver

import (
	"fmt"
	"hulundb-kajus-dns/dns"
	"net"
	"time"
)

type TimeoutError struct {
	error
}

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

		if len(msg.Questions) == 0 {
			return nil, fmt.Errorf("malformed query: no questions")
		}

		if hasCNAMEOnly(msg.Answers, msg.Questions[0].Name, msg.Questions[0].Type) { //CNAME found
			cnameTarget, foundCNAME := getCNAMETarget(msg.Answers, msg.Questions[0].Name)
			if !foundCNAME {
				return nil, fmt.Errorf("malformed response from server: cname detected and not found")
			}

			matchingRecords := filterByNameAndType(msg.Answers, cnameTarget, msg.Questions[0].Type)
			if matchingRecords != nil { // correct target provided by server
				variableName1 := dns.Message{
					Header: dns.Header{
						ID:      msg.Header.ID,
						QR:      true,
						Opcode:  0,
						RD:      false,
						QDCount: 1,
						ANCount: uint16(len(matchingRecords)),
						NSCount: 0,
						ARCount: 0,
					},
					Questions:   msg.Questions,
					Answers:     matchingRecords,
					Authorities: []dns.RR{},
					Additionals: []dns.RR{},
				}
				return variableName1.Encode()
			} else {
				cnameQuery, err := dns.BuildQuery(cnameTarget, msg.Questions[0].Type)
				if err != nil {
					return nil, err
				}
				return Resolve(cnameQuery, depth+1)
			}

		} else {
			for _, rr := range msg.Answers {
				if rr.Header.Type == msg.Questions[0].Type {
					// Found the answer type we asked for
					return response, nil
				}
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
					nsQuery, err := dns.BuildQuery(ns.Name, 1)
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
	const maxRetries = 2
	const timeout = 3 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
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
		conn.SetReadDeadline(time.Now().Add(timeout))
		conn.SetWriteDeadline(time.Now().Add(timeout))

		//Recieves the answer
		buf := make([]byte, 4096)
		n, err := conn.Read(buf)
		if err != nil {
			if isTimeoutError(err) {
				fmt.Printf("Read timeout (attempt %d)\n", attempt+1)
			}
			continue
		}
		return buf[:n], nil
	}
	// All retries exhausted
	return nil, fmt.Errorf("nameserver %s failed after %d retries", target, maxRetries)
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	// Check if it's a net.Timeout error
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}
	return false
}
