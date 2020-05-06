package drivers

import (
	"encoding/json"
)

type DriverHeader struct {
	Driver string
	URL    string
}

func (header *DriverHeader) CreateClient(body json.RawMessage) (cli Client, err error) {
	if URL, ok := (*Aliases)[header.URL]; ok {
		header.URL = URL
	}

	if header.Driver == "" {
		header.Driver, err = GuessDriver(header.URL)
		if err != nil {
			logger.Error(err.Error())
			return
		}
	}

	switch header.Driver {
	case "dns":
		cli, err = NewDnsClient(header.URL)
	case "google":
		cli, err = NewGoogleClient(header.URL, body)
	case "rfc8484":
		cli, err = NewRfc8484Client(header.URL, body)
	case "twin":
		cli, err = NewTwinClient(header.URL, body)
	default:
		err = ErrConfigParse
	}

	return
}

func (header *DriverHeader) CreateService(cli Client, body json.RawMessage) (srv Server, err error) {
	if header.Driver == "" {
		header.Driver, err = GuessDriver(header.URL)
		if err != nil {
			logger.Error(err.Error())
			return
		}
	}

	switch header.Driver {
	case "dns":
		srv, err = NewDnsServer(cli, header.URL, body)
	case "doh", "http", "https":
		srv, err = NewDoHServer(cli, header.URL, body)
	default:
		err = ErrConfigParse
		return
	}

	if err != nil {
		logger.Error(err.Error())
		return
	}
	return
}
