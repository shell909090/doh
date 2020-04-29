package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/miekg/dns"
)

var (
	ErrRequest = errors.New("failed to get response")
)

type Rfc8484Client struct {
	URL       string
	transport http.RoundTripper
}

func (client *Rfc8484Client) Init() (err error) {
	client.transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	return
}

func (client *Rfc8484Client) Exchange(quiz *dns.Msg) (ans *dns.Msg, err error) {
	bquiz, err := quiz.Pack()
	if err != nil {
		logger.Error(err.Error())
		return
	}

	req, err := http.NewRequest("POST", client.URL, bytes.NewBuffer(bquiz))
	if err != nil {
		logger.Error(err.Error())
		return
	}
	req.Header.Add("Accept", "application/dns-message")
	req.Header.Add("Content-Type", "application/dns-message")

	resp, err := client.transport.RoundTrip(req)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err = ErrRequest
		return
	}

	bbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	// logger.Debugf("%s", string(bbody))

	ans = &dns.Msg{}
	err = ans.Unpack(bbody)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	return
}
