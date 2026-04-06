### Work split
M = Milestone

M1 — hulundb: wire format (message parsing/encoding, name compression, all RR types). kajus: UDP server, forwarding logic, integration with upstream resolver.
Integration PR at end of M1: kajus opens a PR on dev that wires both pieces together. hulundb reviews.

M2 — kajus: recursive resolution loop, referral vs authoritative detection, glue records. hulundb: CNAME chain following, record type handling in the resolution path.

M3 — hulundb: cache (data structure, TTL logic, eviction, NXDOMAIN). kajus: concurrency (goroutines, sync.Map, timeout/retry), metrics.

### Branch strategy

main - has to pass tests, and work

dev - merge feature branches here, and make sure things work

feat/ — you're adding something new (a feature)
fix/ — you're correcting something broken (a bug fix)

EX: feat/12-dns-message-parser (Branch for adding the feature dns message parser, that is connected to issue 12) 


### Naming conventions

Ref: https://go.dev/doc/effective_go

MessageParser — capital first letter, exported (visible outside the package)

messageParser — lowercase first letter, unexported (package-private)

camelCase

The Go convention for single-method interfaces is to name them as the method name plus -er

Package - lowercase, one word  EX: dns

hulundb-kajus-dns/
├── go.mod
├── go.sum
├── main.go                    # Entry point. Wires everything together and starts the server.
│
├── dns/                       # Core DNS protocol — wire format only, no IO.
│   ├── message.go             # Message struct, header parsing/encoding.
│   ├── question.go            # Question section parsing/encoding.
│   ├── record.go              # Resource record types (A, AAAA, NS, CNAME, MX, SOA).
│   ├── name.go                # Name parsing, encoding, and compression/decompression.
│   └── message_test.go        # Table-driven tests for round-trip parsing.
│
├── server/                    # UDP listener. Accepts queries, hands off to resolver.
│   ├── server.go              # ListenAndServe, goroutine-per-query dispatch.
│   └── server_test.go
│
├── resolver/                  # The recursive walk logic. No parsing, no caching here.
│   ├── resolver.go            # Resolve() — the main recursive loop.
│   ├── roots.go               # Hardcoded root nameserver addresses.
│   └── resolver_test.go
│
├── cache/                     # TTL-aware in-memory cache. No DNS logic here.
│   ├── cache.go               # Get/Set/evict, TTL decrement, NXDOMAIN entries.
│   └── cache_test.go
│
└── metrics/                   # Counters and timers. Thin wrapper, no dependencies.
    └── metrics.go             # CacheHits, Latency, UpstreamQueries.