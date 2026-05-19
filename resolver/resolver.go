package resolver

import (
	"fmt"
	"hulundb-kajus-dns/cache"
	"hulundb-kajus-dns/dns"
	"net"
	"time"
)

type TimeoutError struct {
	error
}

type Resolver struct {
	cache *cache.Cache
}

func New(c *cache.Cache) *Resolver {
	return &Resolver{cache: c}
}

func (r *Resolver) Resolve(query []byte, depth int) ([]byte, error) {

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

		// Check RCODE - handles non RCODE = 0
		if msg.Header.RCode != dns.RCodeNoError {

			if msg.Header.RCode == dns.RCodeNameError {
				// Cache NXDOMAIN
				qname := msg.Questions[0].Name
				qtype := msg.Questions[0].Type
				r.cache.SetNegative(qname, qtype, 300)
				return response, nil
			}

			if len(msg.Authorities) > 0 {
				nextTarget, found := tryAllNameservers(r, query, msg, depth)
				if found {
					target = nextTarget
					continue
				}
			}

			// No NS to test, returns server response
			return response, nil
		}

		cacheResult := r.cache.Get(msg.Questions[0].Name, msg.Questions[0].Type)

		if cacheResult.Found {
			cacheMessage := dns.Message{
				Header: dns.Header{
					ID:      msg.Header.ID,
					QR:      true,
					Opcode:  0,
					RD:      false,
					QDCount: 1,
					ANCount: uint16(len(cacheResult.Records)),
					NSCount: 0,
					ARCount: 0,
				},
				Questions:   msg.Questions,
				Answers:     cacheResult.Records,
				Authorities: []dns.RR{},
				Additionals: []dns.RR{},
			}
			return cacheMessage.Encode()
		}
		// if cacheResult.Negative {
		//

		// }

		if hasCNAMEOnly(msg.Answers, msg.Questions[0].Name, msg.Questions[0].Type) { //CNAME found
			cnameTarget, foundCNAME := getCNAMETarget(msg.Answers, msg.Questions[0].Name)
			if !foundCNAME {
				return nil, fmt.Errorf("malformed response from server: cname detected and not found")
			}

			matchingRecords := filterByNameAndType(msg.Answers, cnameTarget, msg.Questions[0].Type)
			if matchingRecords != nil { // correct target provided by server
				cnameMessage := dns.Message{
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
				r.cache.Set(msg.Questions[0].Name, msg.Questions[0].Type, matchingRecords)
				return cnameMessage.Encode()
			} else {
				cnameQuery, err := dns.BuildQuery(cnameTarget, msg.Questions[0].Type)
				if err != nil {
					return nil, err
				}
				return r.Resolve(cnameQuery, depth+1)
			}

		} else {
			hasAnswer := false
			for _, rr := range msg.Answers {
				if rr.Header.Type == msg.Questions[0].Type {
					hasAnswer = true
					break
				}
			}
			if hasAnswer {
				r.cache.Set(msg.Questions[0].Name, msg.Questions[0].Type, msg.Answers)
				return response, nil
			}
		}

		// Try all authorative NS
		nextTarget, found := tryAllNameservers(r, query, msg, depth)
		if !found {
			// all failed NS returns - SERVFAIL
			return buildServfailResponse(msg), nil
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

// tryAllNameservers - returns the next IP
func tryAllNameservers(r *Resolver, query []byte, msg dns.Message, depth int) (string, bool) {
	// Try 1 — Already an IP?
	for _, rr := range msg.Additionals {
		if a, ok := rr.Data.(dns.ARecord); ok {
			return a.IP.String(), true
		}
	}

	// Try 2 — loopa ALL NS
	for _, rr := range msg.Authorities {
		if ns, ok := rr.Data.(dns.NSRecord); ok {
			// Try to get IP for NS
			nsQuery, err := dns.BuildQuery(ns.Name, 1)
			if err != nil {
				continue
			}

			nsResponse, err := r.Resolve(nsQuery, depth+1)
			if err != nil {
				continue
			}

			nsMsg, err := dns.DecodeMessage(nsResponse)
			if err != nil {
				continue
			}

			// Was an A-record found for this NS?
			for _, nsRR := range nsMsg.Answers {
				if a, ok := nsRR.Data.(dns.ARecord); ok {
					return a.IP.String(), true
				}
			}
		}
	}

	// Ingen NS fungerade
	return "", false
}

// Contruct DNS - respons for Serverfail instead of returning error
func buildServfailResponse(msg dns.Message) []byte {
	response := dns.Message{
		Header: dns.Header{
			ID:      msg.Header.ID,
			QR:      true,
			Opcode:  0,
			RCode:   dns.RCodeServerFailure, // SERVFAIL = 2
			QDCount: uint16(len(msg.Questions)),
			ANCount: 0,
			NSCount: 0,
			ARCount: 0,
		},
		Questions:   msg.Questions,
		Answers:     []dns.RR{},
		Authorities: []dns.RR{},
		Additionals: []dns.RR{},
	}

	encoded, err := response.Encode()
	if err != nil {
		return nil
	}
	return encoded
}
