package main

import (
	"github.com/miekg/dns"
)

type DnsClient struct {
	Type   string
	Net    string
	Server string
	client *dns.Client
}

func (client *DnsClient) Init() (err error) {
	client.client = &dns.Client{
		Net: client.Net,
	}
	return
}

func (client *DnsClient) Exchange(quiz *dns.Msg) (ans *dns.Msg, err error) {
	ans, _, err = client.client.Exchange(quiz, client.Server)
	return
}
