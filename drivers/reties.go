package drivers

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/miekg/dns"
)

var (
	ErrEmptyClients = errors.New("empty client list")
)

type RetiesClient struct {
	Tries   int
	Clients []json.RawMessage
	clis    []Client
}

func NewRetiesClient(URL string, body json.RawMessage) (cli *RetiesClient) {
	var err error
	cli = &RetiesClient{}
	if body != nil {
		err = json.Unmarshal(body, &cli)
		if err != nil {
			panic(err.Error())
		}
	}

	var header DriverHeader
	for _, cfg := range cli.Clients {
		err = json.Unmarshal(cfg, &header)
		if err != nil {
			panic(err.Error())
		}
		cli.clis = append(cli.clis, header.CreateClient(cfg))
	}

	return
}

func (cli *RetiesClient) AddClient(client Client) {
	cli.clis = append(cli.clis, client)
}

func (cli *RetiesClient) Url() (u string) {
	return cli.clis[0].Url()
}

func (cli *RetiesClient) Exchange(ctx context.Context, quiz *dns.Msg) (ans *dns.Msg, err error) {
	if len(cli.clis) == 0 {
		panic(ErrEmptyClients.Error())
	}

	for i := 0; i < cli.Tries; i++ {
		cur := cli.clis[i%len(cli.clis)]
		ans, err = cur.Exchange(ctx, quiz)
		if err == nil {
			return
		}
		logger.Info(err.Error())
	}
	return
}
