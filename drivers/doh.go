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

func NewDoHServer(cli Client, URL string, body json.RawMessage) (srv *DoHServer) {
	u, err := url.Parse(URL)
	if err != nil {
		panic(err.Error())
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
			panic(err.Error())
		}
	}

	srv.mux.Handle("/dns-query", NewRfc8484Handler(cli, srv.EdnsClientSubnet))
	srv.mux.Handle("/resolve", NewGoogleHandler(cli, srv.EdnsClientSubnet))
	srv.mux.Handle("/d", NewDnsPodHandler(cli, srv.EdnsClientSubnet))
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
