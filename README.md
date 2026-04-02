# hulundb-kajus — Recursive DNS Resolver in Go

> **DD1349 Project** | Build a functioning recursive DNS resolver from scratch in Go, capable of answering real queries by walking the DNS hierarchy — no external resolver libraries.

---

## Goal

A fully recursive DNS resolver that handles real-world queries end-to-end: parsing the DNS wire format, walking the hierarchy from root servers down to authoritative nameservers, caching results with TTL respect, and serving concurrent clients. Validated with `dig @127.0.0.1 -p 5353 google.com`.

---

## Milestones

### Milestone 1 — Protocol & Stub Resolver (~25hrs each)

Implement DNS wire format parsing/encoding and a UDP server that forwards queries to an upstream resolver (e.g. `8.8.8.8`). This validates packet parsing before building recursive logic.

**Issues:**
- Parse DNS message headers (12-byte binary structure, flags, counts)
- Parse and encode question, answer, authority, and additional sections
- Parse all resource record types: A, AAAA, NS, CNAME, MX, SOA
- Implement DNS name compression (pointer labels, RFC 1035 §4.1.4)
- UDP server listening on port 5353
- Forward queries to upstream resolver and relay responses
- Integration test: `dig @127.0.0.1 -p 5353 google.com` returns a real answer

**Work split:** One person owns wire format (parsing/encoding); the other owns the UDP server and forwarding logic. Integrate at end of M1.

---

### Milestone 2 — Recursive Resolution (~30hrs each)

Replace forwarding with a full recursive walk of the DNS tree: starting from root nameservers, following referrals through TLD servers to authoritative servers.

**Issues:**
- Hardcode the 13 root nameserver addresses (from IANA `named.cache`)
- Implement the recursive lookup loop: roots → referral → TLD → referral → authoritative → answer
- Handle A, AAAA, CNAME, NS, MX record types in resolution
- CNAME chain following (e.g. `www.github.com` resolves via a CNAME)
- Correctly distinguish referrals from authoritative answers via response flags
- Handle glue records (nameserver addresses in the additional section of referrals)

---

### Milestone 3 — Caching & Robustness (~25hrs each)

Make it production-worthy: TTL-respecting cache, failure handling, and concurrent query serving.

**Issues:**
- In-memory cache keyed on `(name, type)`, respecting TTL from responses
- TTL decrement as time passes; evict expired entries
- Cache negative responses (NXDOMAIN) to avoid hammering servers for nonexistent names
- Timeout and retry logic for unresponsive nameservers
- Concurrent query handling via goroutines (`sync.Map` or mutex-protected cache)
- Basic metrics: cache hit rate, query latency, upstream queries per second

---

## Must-Haves / Nice-to-Haves / Risks

| Category | Item |
|---|---|
| **Must-have** | DNS wire format parser & encoder (RFC 1035 compliant) |
| **Must-have** | UDP server on port 5353 |
| **Must-have** | Recursive resolution from root → TLD → authoritative |
| **Must-have** | A, AAAA, NS, CNAME, MX record type support |
| **Must-have** | CNAME chain following |
| **Must-have** | TTL-respecting in-memory cache |
| **Must-have** | NXDOMAIN negative caching |
| **Must-have** | Concurrent query handling with safe shared state |
| **Must-have** | Timeout/retry for unresponsive nameservers |
| **Nice-to-have** | Basic metrics (cache hit rate, latency, upstream QPS) |
| **Nice-to-have** | DNS over TCP for responses > 512 bytes |
| **Nice-to-have** | CLI tool to inspect cache state |
| **Nice-to-have** | Zone file parsing |
| **Nice-to-have** | DNS over TLS (DoT) to `1.1.1.1` |
| **Risk** | Name compression edge cases (pointer loops, forward pointers) |
| **Risk** | Glue record handling in referrals |
| **Risk** | CNAME chains that are circular or excessively long |
| **Risk** | Race conditions in concurrent cache access |
| **Risk** | Upstream rate limiting / blocking during development |
| **Risk** | Integration complexity at the M1 boundary (wire format ↔ server) |

---

## Requirements

### Business Requirements
- A developer should be able to resolve arbitrary domain names against our server and get correct, cached answers with no dependency on a third-party resolver library.

### System Requirements
- The server must implement the DNS wire format per RFC 1035.
- The server must walk the DNS hierarchy from root servers to authoritative servers without delegating to an upstream resolver.
- The server must cache responses respecting TTL values and serve cached answers to repeated queries.
- The server must handle concurrent queries safely.

### Functional Requirements
- Parse and encode all DNS message sections (header, question, answer, authority, additional).
- Implement DNS name compression encoding and decoding.
- Hardcode root nameserver hints from `https://www.internic.net/domain/named.cache`.
- Recursive resolution loop: query roots → follow referrals → return authoritative answer.
- Cache keyed on `(name, type)` with TTL-based expiry; evict on read if expired.
- Each incoming query runs in its own goroutine; cache protected by `sync.Map` or `sync.Mutex`.

---

## Architecture

```
Client (dig, browser)
        │  query
        ▼
  ┌─────────────────────────┐
  │     Your DNS Server     │
  │                         │
  │  Cache ──► Resolver     │──► Root servers (a.root-servers.net …)
  │  TTL-aware  recursive   │         │ referral
  │  in-memory  walk logic  │◄────────┘
  └─────────────────────────┘
        │  answer           ──► TLD servers (.com, .se, .org …)
        ▼                            │ referral
     Client                         ▼
                             Auth. servers (ns1.google.com …)
                                     │ answer
                                     ▼
                              (back to resolver)
```

---

## Key References

- [RFC 1035](https://www.rfc-editor.org/rfc/rfc1035) — Sections 3 (data formats) and 4 (messages)
- [IANA root hints](https://www.internic.net/domain/named.cache) — the 13 root nameserver addresses to hardcode
- [miekg/dns](https://github.com/miekg/dns) — full Go DNS library; useful for reading source when stuck on parsing edge cases (do not use as a dependency)
- `dig +trace google.com` — makes dig perform its own recursive resolution, printing every step; use as ground truth for what your resolver should do

---

## Stretch Goals

- Zone file parsing
- DNS over TCP for large responses (> 512 bytes)
- Small CLI tool to inspect cache contents
- DNS over TLS (DoT) to `1.1.1.1`
