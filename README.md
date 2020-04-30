# Abstract

[DNS over HTTPS](https://en.wikipedia.org/wiki/DNS_over_HTTPS) utils written by golang.

# Protocol and URL

## dns

There have three different protocols in DNS:

* udp
* tcp
* tcp-tls

## doh

DoH means DNS over HTTPS. It include two protocols:

* rfc8484
* google

As output protocol, you should indicate rfc8484 or google, to specify which exactly protocol we actually use.

As input protocol, doh are fine. We support both protocols on the same http/https server.

# Examples

	doh -q --protocol dns --url udp://114.114.114.114:53 www.baidu.com
	doh -q --protocol dns --url tcp://114.114.114.114:53 www.baidu.com
	doh -q --protocol dns --url tcp-tls://one.one.one.one:853 www.baidu.com

	doh -q --protocol rfc8484 --url https://security.cloudflare-dns.com/dns-query www.baidu.com
	# with proxy
	doh -q --protocol google --url https://dns.google.com/resolve www.baidu.com

	doh --config udp.json
	dig www.baidu.com @127.0.0.1 -p 5053

	doh --config doh.json
	doh -q --protocol rfc8484 --url http://localhost:8053/dns-query www.baidu.com

	doh --config dohs.json
	doh -q --protocol rfc8484 --url https://localhost:8053/dns-query --insecure www.baidu.com

# TODO

* cache
* edns-client-subnet
* record dns logs
* DoHServer support google protocol
