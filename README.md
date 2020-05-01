# Table of content

* [Abstract](#abstract)

# Abstract

[DNS over HTTPS](https://en.wikipedia.org/wiki/DNS_over_HTTPS) utils written by golang.

# Compile

	make

# Install

# Command line options and args

# Config

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

	doh --config udp-rfc8484.json
	dig www.baidu.com @127.0.0.1 -p 5053

	doh --config doh.json
	doh -q --protocol rfc8484 --url http://localhost:8053/dns-query www.baidu.com

	doh --config dohs.json
	doh -q --protocol rfc8484 --url https://localhost:8153/dns-query --insecure www.baidu.com

# 114

* Domain: public1.114dns.com
* IP: 114.114.114.114
* Accept protocols: udp/tcp
* Don't accept edns-client-subnet, in any protocols.
* No proxy needed in China.

# Cloudflare one

* Domain: one.one.one.one
* IP: 1.1.1.1/1.0.0.1
* Accept protocols: udp/tcp/tcp-tls
* Don't accept edns-client-subnet, in any protocols.
* No proxy needed in China.
* Accuracy: not best result in China.

# Cloudflare doh

* Domain: security.cloudflare-dns.com
* IP: 104.18.2.55/104.18.3.55
* Accept protocols: rfc8484
* Don't accept edns-client-subnet, in any protocols.
* No proxy needed in China.
* Accuracy: not best result in China.

# Google

* Domain: dns.google.com
* IP: 8.8.8.8/8.8.4.4
* Accept protocols: udp/tcp/tcp-tls/google.
* Accept edns-client-subnet with protocol google.
* A proxy will be needed in China.

# OpenDNS

* Domain: dns.opendns.com
* IP: 208.67.222.222/208.67.220.220
* Accept protocols: udp/tcp
* Don't accept edns-client-subnet, in any protocols.
* No proxy needed in China.

# Quad9

* Domain: dns.quad9.net
* IP: 9.9.9.9/149.112.112.112
* Accept protocols: udp/tcp/tcp-tls/rfc8484
* Don't accept edns-client-subnet, in any protocols.
* No proxy needed in China.
* Accuracy: wrong result in China (taobao and baidu).

# Suggested in China

1. Don't use Quad9. Wrong result means useless.
2. I won't suggest Cloudflare. Not the best result. Don't use it unless running out of other options.
3. Google/OpenDNS with udp (direct connect, will be interfered by the GFW). Find yourself a way to dodge the firewall.
4. Google with edns-client-subnet (proxy needed).
5. If you want tcp-tls, the first choice is Google (proxy needed in China), then Cloudflare (no proxy needed in China).
6. If you want rfc8484, the only option here is Cloudflare. Don't use Quad9.

# TODO

* cache
* record dns logs
* multiple outputs, load balance
