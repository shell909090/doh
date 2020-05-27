package rrtree

import (
	"fmt"

	"github.com/miekg/dns"
)

type ResourceRecord struct {
	Name string
	Type uint16
	// Expire time.Time
	RR dns.RR
}

type Node struct {
	Name      string
	Subs      map[string]*Node
	Resources map[uint16][]*ResourceRecord
}

func NewNode(name string) (node *Node) {
	node = &Node{
		Name:      name,
		Subs:      make(map[string]*Node, 0),
		Resources: make(map[uint16][]*ResourceRecord, 0),
	}
	return
}

type RRTree struct {
	*Node
}

func NewRRTree() (tree *RRTree) {
	tree = &RRTree{
		Node: NewNode(""),
	}
	return
}

func (tree *RRTree) AddRecord(rr dns.RR) {
	record := &ResourceRecord{
		Name: rr.Header().Name,
		Type: rr.Header().Rrtype,
		// Expire: time.Now().Add(time.Duration(rr.Header().Ttl) * time.Second),
		RR: rr,
	}

	labels := dns.SplitDomainName(record.Name)
	fmt.Println(labels)

	node := tree.Node
	for i := len(labels) - 1; i >= 0; i-- {
		sub, ok := node.Subs[labels[i]]
		if !ok {
			sub = NewNode(labels[i])
			node.Subs[labels[i]] = sub
		}
		node = sub
	}

	node.Resources[record.Type] = append(node.Resources[record.Type], record)
	return
}

func (tree *RRTree) GetNode(name string) (node *Node, partial bool) {
	labels := dns.SplitDomainName(name)
	fmt.Println(labels)

	node = tree.Node
	for i := len(labels) - 1; i >= 0; i-- {
		sub, ok := node.Subs[labels[i]]
		if !ok {
			partial = true
			return
		}
		node = sub
	}

	return
}

var DefaultTree *RRTree = NewRRTree()

var (
	ROOTDNS = []string{
		"a.root-servers.net. 300 IN A 198.41.0.4",
		"b.root-servers.net. 300 IN A 199.9.14.201",
		"c.root-servers.net. 300 IN A 192.33.4.12",
		"d.root-servers.net. 300 IN A 199.7.91.13",
		"e.root-servers.net. 300 IN A 192.203.230.10",
		"f.root-servers.net. 300 IN A 192.5.5.241",
		"g.root-servers.net. 300 IN A 192.112.36.4",
		"h.root-servers.net. 300 IN A 198.97.190.53",
		"i.root-servers.net. 300 IN A 192.36.148.17",
		"j.root-servers.net. 300 IN A 192.58.128.30",
		"k.root-servers.net. 300 IN A 193.0.14.129",
		"l.root-servers.net. 300 IN A 199.7.83.42",
		"m.root-servers.net. 300 IN A 202.12.27.33",
	}
)

func init() {
	for _, s := range ROOTDNS {
		rr, err := dns.NewRR(s)
		if err != nil {
			panic(err.Error())
		}
		DefaultTree.AddRecord(rr)
	}
	return
}
