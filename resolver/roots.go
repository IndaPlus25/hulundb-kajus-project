package resolver

type RootServer struct {
	Name string
	IPv4 string
}

var RootServers = []RootServer{
	{Name: "a.root-servers.net", IPv4: "198.41.0.4"},
	{Name: "b.root-servers.net", IPv4: "170.247.170.13"},
	{Name: "c.root-servers.net", IPv4: "192.33.4.12"},
	{Name: "d.root-servers.net", IPv4: "199.7.91.13"},
	{Name: "e.root-servers.net", IPv4: "192.203.230.10"},
	{Name: "f.root-servers.net", IPv4: "192.5.5.241"},
	{Name: "g.root-servers.net", IPv4: "192.112.36.4"},
	{Name: "h.root-servers.net", IPv4: "198.97.190.53"},
	{Name: "i.root-servers.net", IPv4: "192.36.148.17"},
	{Name: "j.root-servers.net", IPv4: "192.58.128.30"},
	{Name: "k.root-servers.net", IPv4: "193.0.14.129"},
	{Name: "l.root-servers.net", IPv4: "199.7.83.42"},
	{Name: "m.root-servers.net", IPv4: "202.12.27.33"},
}
