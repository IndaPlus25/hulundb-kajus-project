package main

import (
	"hulundb-kajus-dns/cache"
	"hulundb-kajus-dns/resolver"
	"hulundb-kajus-dns/server"
	"time"
)

func main() {
	c := cache.NewCache()
	c.StartEviction(60 * time.Second)
	r := resolver.New(c)
	server.Start("0.0.0.0:53", r)
}
