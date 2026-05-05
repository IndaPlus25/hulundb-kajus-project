package resolver

import (
	"hulundb-kajus-dns/dns"
)

// checks if the answer section has a CNAME for `name` but NOT the requested qtype
// this is your detection condition
func hasCNAMEOnly(answers []dns.RR, name string, qtype uint16) bool {
	foundCNAME := false
	foundQType := false

	for _, rr := range answers {
		if rr.Header.Name != name {
			continue
		}
		if rr.Header.Type == dns.TypeCNAME {
			foundCNAME = true
		}
		if rr.Header.Type == qtype {
			foundQType = true
		}
	}

	return foundCNAME && !foundQType

}

// extracts the CNAME target name from the answer section
func getCNAMETarget(answers []dns.RR, name string) (string, bool) {

	for _, rr := range answers {
		if rr.Header.Name != name {
			continue
		}
		if rr.Header.Type == dns.TypeCNAME {
			if data, ok := rr.Data.(dns.CNAMERecord); ok {
				return data.Name, true
			}
		}
	}

	return "", false
}

// returns all records matching name AND qtype
// used to check if the chain's final answer was bundled in the same response
func filterByNameAndType(answers []dns.RR, name string, qtype uint16) []dns.RR {
	var matchingRecords []dns.RR

	for _, rr := range answers {
		if rr.Header.Name != name {
			continue
		}
		if rr.Header.Type == qtype {
			matchingRecords = append(matchingRecords, rr)
		}
	}

	return matchingRecords
}
