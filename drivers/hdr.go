package drivers

import (
	"encoding/json"
)

type DriverHeader struct {
	Driver string
	URL    string
}

func (header *DriverHeader) CreateClient(body json.RawMessage) (cli Client) {
	var err error
	if header.Driver == "" {
		header.Driver, err = GuessDriver(header.URL)
		if err != nil {
			panic(err.Error())
		}
	}

	switch header.Driver {
	case "dns":
		cli = NewDnsClient(header.URL)
	case "google":
		cli = NewGoogleClient(header.URL, body)
	case "rfc8484":
		cli = NewRfc8484Client(header.URL, body)
	case "dnspod":
		cli = NewDnsPodClient(header.URL, body)
	case "twin":
		cli = NewTwinClient(header.URL, body)
	default:
		err = ErrConfigParse
		panic(err.Error())
	}

	return
}

func (header *DriverHeader) CreateService(cli Client, body json.RawMessage) (srv Server) {
	var err error
	if header.Driver == "" {
		header.Driver, err = GuessDriver(header.URL)
		if err != nil {
			panic(err.Error())
		}
	}

	switch header.Driver {
	case "dns":
		srv = NewDnsServer(cli, header.URL, body)
	case "doh", "http", "https":
		srv = NewDoHServer(cli, header.URL, body)
	default:
		err = ErrConfigParse
		panic(err.Error())
	}

	return
}
