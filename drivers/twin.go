package drivers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/miekg/dns"
	"github.com/shell909090/doh/iplist"
)

type TwinClient struct {
	Primary       json.RawMessage
	primary_cli   Client
	Secondary     json.RawMessage
	secondary_cli Client
	DirectRoutes  string `json:"direct-routes"`
	dir_routes    *iplist.IPList
}

func NewTwinClient(URL string, body json.RawMessage) (cli *TwinClient) {
	var err error
	cli = &TwinClient{}
	if body != nil {
		err = json.Unmarshal(body, &cli)
		if err != nil {
			panic(err.Error())
		}
	}

	var header DriverHeader
	err = json.Unmarshal(cli.Primary, &header)
	if err != nil {
		panic(err.Error())
	}
	cli.primary_cli = header.CreateClient(cli.Primary)
	logger.Debugf("primary: %+v", cli.primary_cli)

	err = json.Unmarshal(cli.Secondary, &header)
	if err != nil {
		panic(err.Error())
	}
	cli.secondary_cli = header.CreateClient(cli.Secondary)
	logger.Debugf("secondary: %+v", cli.secondary_cli)

	cli.dir_routes, err = iplist.ReadIPListFile(cli.DirectRoutes)
	if err != nil {
		panic(err.Error())
	}

	return
}

func (cli *TwinClient) Url() (u string) {
	return fmt.Sprintf("%s+%s", cli.primary_cli.Url(), cli.secondary_cli.Url())
}

func (cli *TwinClient) Exchange(ctx context.Context, quiz *dns.Msg) (ans *dns.Msg, err error) {
	ans, err = cli.primary_cli.Exchange(ctx, quiz)
	if err != nil {
		return
	}

	is_secondary := true
	for _, rr := range ans.Answer {
		switch v := rr.(type) {
		case *dns.A:
			if cli.dir_routes.Contain(v.A) {
				is_secondary = false
				break
			}
		}
	}

	if is_secondary {
		logger.Debugf("secondary query")
		ans, err = cli.secondary_cli.Exchange(ctx, quiz)
	}

	return
}
