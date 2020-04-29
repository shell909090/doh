package main

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/miekg/dns"
)

var (
	ErrRequest = errors.New("failed to get response")
)

func WriteFull(w io.Writer, b []byte) (err error) {
	n, err := w.Write(b)
	if err != nil {
		return
	}
	if n != len(b) {
		return io.ErrShortWrite
	}
	return
}

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

type Rfc8484Handler struct {
	Client Client
}

func (handler *Rfc8484Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var err error
	defer req.Body.Close()

	var bbody []byte
	switch req.Method {
	case "GET":
	case "POST":
		bbody, err = ioutil.ReadAll(req.Body)
		if err != nil {
			logger.Error(err.Error())
			w.WriteHeader(400)
			return
		}
	default:
		w.WriteHeader(400)
		return
	}

	quiz := &dns.Msg{}
	err = quiz.Unpack(bbody)
	if err != nil {
		logger.Error(err.Error())
		w.WriteHeader(400)
		return
	}

	logger.Infof("rfc8484 server query: %s", quiz.Question[0].Name)

	ans, err := handler.Client.Exchange(quiz)
	if err != nil {
		logger.Error(err.Error())
		w.WriteHeader(502)
		return
	}

	bbody, err = ans.Pack()
	if err != nil {
		logger.Error(err.Error())
		w.WriteHeader(502)
		return
	}

	err = WriteFull(w, bbody)
	if err != nil {
		logger.Error(err.Error())
		w.WriteHeader(502)
		return
	}

	return
}

type DoHServer struct {
	Scheme string
	Addr   string
	Client Client
	mux    *http.ServeMux
}

func (srv *DoHServer) Init() (err error) {
	srv.mux = http.NewServeMux()
	srv.mux.Handle("/dns-query", &Rfc8484Handler{Client: srv.Client})
	return
}

func (srv *DoHServer) Run() (err error) {
	server := &http.Server{
		Addr:    srv.Addr,
		Handler: srv.mux,
	}
	err = server.ListenAndServe()
	return
}
