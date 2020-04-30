package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
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
	Insecure  bool
	transport *http.Transport
}

func NewRfc8484Client(URL string, Insecure bool) (cli *Rfc8484Client) {
	cli = &Rfc8484Client{
		URL:      URL,
		Insecure: Insecure,
		transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}
	if Insecure {
		cli.transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return
}

func (cli *Rfc8484Client) Exchange(quiz *dns.Msg) (ans *dns.Msg, err error) {
	bquiz, err := quiz.Pack()
	if err != nil {
		logger.Error(err.Error())
		return
	}

	req, err := http.NewRequest("POST", cli.URL, bytes.NewBuffer(bquiz))
	if err != nil {
		logger.Error(err.Error())
		return
	}
	req.Header.Add("Accept", "application/dns-message")
	req.Header.Add("Content-Type", "application/dns-message")

	resp, err := cli.transport.RoundTrip(req)
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

	ans = &dns.Msg{}
	err = ans.Unpack(bbody)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	return
}

type Rfc8484Handler struct {
	cli Client
}

func (handler *Rfc8484Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var err error
	defer req.Body.Close()

	var bdns []byte
	switch req.Method {
	case "GET":
		err = req.ParseForm()
		if err != nil {
			logger.Error(err.Error())
			w.WriteHeader(400)
			return
		}

		b64dns := req.Form.Get("dns")
		bdns, err = base64.StdEncoding.DecodeString(b64dns)
		if err != nil {
			logger.Error(err.Error())
			w.WriteHeader(400)
			return
		}

	case "POST":
		bdns, err = ioutil.ReadAll(req.Body)
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
	err = quiz.Unpack(bdns)
	if err != nil {
		logger.Error(err.Error())
		w.WriteHeader(400)
		return
	}

	logger.Infof("rfc8484 server query: %s", quiz.Question[0].Name)

	ans, err := handler.cli.Exchange(quiz)
	if err != nil {
		logger.Error(err.Error())
		w.WriteHeader(502)
		return
	}

	bdns, err = ans.Pack()
	if err != nil {
		logger.Error(err.Error())
		w.WriteHeader(502)
		return
	}

	w.Header().Add("Content-Type", "application/dns-message")
	w.Header().Add("Cache-Control", "no-cache, max-age=0")
	w.WriteHeader(200)

	err = WriteFull(w, bdns)
	if err != nil {
		logger.Error(err.Error())
		w.WriteHeader(502)
		return
	}

	return
}

type DoHServer struct {
	Scheme   string
	Addr     string
	CertFile string
	KeyFile  string
	cli      Client
	mux      *http.ServeMux
}

func NewDoHServer(cli Client, Scheme, Addr, CertFile, KeyFile string) (srv *DoHServer) {
	srv = &DoHServer{
		Scheme:   Scheme,
		Addr:     Addr,
		CertFile: CertFile,
		KeyFile:  KeyFile,
		cli:      cli,
		mux:      http.NewServeMux(),
	}
	srv.mux.Handle("/dns-query", &Rfc8484Handler{cli: cli})
	return
}

func (srv *DoHServer) Run() (err error) {
	server := &http.Server{
		Addr:    srv.Addr,
		Handler: srv.mux,
	}
	switch srv.Scheme {
	case "http":
		err = server.ListenAndServe()
	case "https", "":
		err = server.ListenAndServeTLS(srv.CertFile, srv.KeyFile)
	}
	return
}
