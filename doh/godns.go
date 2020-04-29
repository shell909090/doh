package main

import (
	"time"

	"github.com/miekg/dns"
)

type DnsProfile struct {
	Type   string
	Net    string
	Server string
	client *dns.Client
}

func (prof *DnsProfile) Init() (err error) {
	prof.client = &dns.Client{
		Net: prof.Net,
	}
	return
}

func (prof *DnsProfile) Exchange(quiz *dns.Msg) (ans *dns.Msg, rtt time.Duration, err error) {
	ans, rtt, err = prof.client.Exchange(quiz, prof.Server)
	return
}
