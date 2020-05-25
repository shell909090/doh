package drivers

import (
	"encoding/json"
	"net/http"
	"net/url"
)

type DoHServer struct {
	CertFile         string
	KeyFile          string
	EdnsClientSubnet string
	scheme           string
	addr             string
	cli              Client
	mux              *http.ServeMux
}

func NewDoHServer(cli Client, URL string, body json.RawMessage) (srv *DoHServer, err error) {
	var u *url.URL
	u, err = url.Parse(URL)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	srv = &DoHServer{
		scheme: u.Scheme,
		addr:   u.Host,
		cli:    cli,
		mux:    http.NewServeMux(),
	}

	if body != nil {
		err = json.Unmarshal(body, &srv)
		if err != nil {
			logger.Error(err.Error())
			return
		}
	}

	rfc8484h, err := NewRfc8484Handler(cli, srv.EdnsClientSubnet)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	srv.mux.Handle("/dns-query", rfc8484h)

	googleh, err := NewGoogleHandler(cli, srv.EdnsClientSubnet)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	srv.mux.Handle("/resolve", googleh)

	dnspodh, err := NewDnsPodHandler(cli, srv.EdnsClientSubnet)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	srv.mux.Handle("/d", dnspodh)
	return
}

func (srv *DoHServer) Serve() (err error) {
	server := &http.Server{
		Addr:    srv.addr,
		Handler: srv.mux,
	}

	switch srv.scheme {
	case "http":
		err = server.ListenAndServe()
	case "https", "":
		err = server.ListenAndServeTLS(srv.CertFile, srv.KeyFile)
	}
	return
}
