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

type DnsServer struct {
	Net    string
	Addr   string
	Client Client
}

func (srv *DnsServer) Init() (err error) {
	return
}

func (srv *DnsServer) ServeDNS(w dns.ResponseWriter, quiz *dns.Msg) {
	logger.Infof("dns server query: %s", quiz.Question[0].Name)
	ans, err := srv.Client.Exchange(quiz)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	if ans == nil {
		logger.Error("response is nil.")
		return
	}

	err = w.WriteMsg(ans)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	return
}

func (srv *DnsServer) Run() (err error) {
	server := &dns.Server{
		Net:     srv.Net,
		Addr:    srv.Addr,
		Handler: srv,
	}
	logger.Infof("dns server start.")
	err = server.ListenAndServe()
	return
}
