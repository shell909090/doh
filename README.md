# Table of content

* [Abstract](#abstract)
* [Compile and Install](#compile-and-install)
* [Command line options and args](#command-line-options-and-args)
* [Config](#config)
* [Protocol and URL](#protocol-and-url)
  * [dns](#dns)
  * [doh](#doh)
* [Public recursive server](#public-recursive-server)
  * [114](#114)
  * [Cloudflare one](#cloudflare-one)
  * [Cloudflare doh](#cloudflare-doh)
  * [Google](#google)
  * [OpenDNS](#OpenDNS)
  * [Quad9](#Quad9)
* [Config](#config)
* [Suggestions in China](#suggestions-in-china)
* [TODO](#todo)

# Abstract

[DNS over HTTPS](https://en.wikipedia.org/wiki/DNS_over_HTTPS) utils written by golang.

# Compile and Install

	make

The executable file is placed under `bin/`. Copy it to wherever you like. Enjoy.

# Command line options and args

See `doh --help`.

# Config

* logfile: optional. indicate which file log should be written to. empty means stdout. empty by default.
* loglevel: optional. log level. warning by default.
* input-protocol: optional. see "protocol and url". if empty, program will auto guess.
* input-url: required. see "protocol and url".
* input-cert-file: optional. cert file when use doh.
* input-key-file: optional. key file when use doh.
* edns-client-subnet: optional. it could be empty, means don't do anything. or "client", means read remote address and put it into edns-client-subnet. or an ip address/cidr subnet, means put this address into edns-client-subnet. empty by default.
* output-protocol: optional. see "protocol and url". if empty, program will auto guess.
* output-url: required. see "protocol and url".
* output-insecure: optional. don't verify the certificate from the server.

# Protocol and URL

## dns

There have three different protocols in DNS:

* udp
* tcp
* tcp-tls

Here are some examples as output.

	doh -q --protocol dns --url udp://114.114.114.114:53 www.baidu.com
	doh -q --protocol dns --url tcp://114.114.114.114:53 www.baidu.com
	doh -q --protocol dns --url tcp-tls://one.one.one.one:853 www.baidu.com

Here are some examples as input.

	doh --config udp-rfc8484.json
	dig www.baidu.com @127.0.0.1 -p 5053
	doh --config udp-google.json
	dig www.baidu.com @127.0.0.1 -p 5153

## doh

DoH means DNS over HTTPS. It include two protocols:

* rfc8484
* google

As an output protocol, you should indicate rfc8484 or google, to specify which exactly protocol we actually use.

Here are some examples as output.

	doh -q --protocol rfc8484 --url https://security.cloudflare-dns.com/dns-query www.baidu.com
	doh -q --protocol google --url https://dns.google.com/resolve www.baidu.com

As an input protocol, doh are fine. We support both protocols on the same http/https server.

Here are some examples as input.

	doh --config doh.json
	doh -q --protocol rfc8484 --url http://localhost:8053/dns-query www.baidu.com
	doh -q --protocol google --url http://localhost:8053/resolve www.baidu.com

	doh --config dohs.json
	doh -q --protocol rfc8484 --url https://localhost:8153/dns-query --insecure www.baidu.com
	doh -q --protocol google --url https://localhost:8153/resolve --insecure www.baidu.com

# Public recursive server

## 114

* Domain: public1.114dns.com
* IP: 114.114.114.114
* Accept protocols: udp/tcp
* Response time in Shanghai: udp:12, tcp:22
* Don't accept edns-client-subnet, in any protocols.
* No proxy needed in China.

## Cloudflare one

* Domain: one.one.one.one
* IP: 1.1.1.1/1.0.0.1
* Accept protocols: udp/tcp/tcp-tls
* Response time in Shanghai: udp:78, tcp:157, tcp-tls:250
* Response time in Japan IDC: udp:2, tcp:2, tcp-tls:22
* Don't accept edns-client-subnet, in any protocols.
* No proxy needed in China.
* Accuracy: not best result in China.

## Cloudflare doh

* Domain: security.cloudflare-dns.com
* IP: 104.18.2.55/104.18.3.55
* Accept protocols: rfc8484
* Response time in Shanghai: rfc8484:83
* Response time in Japan IDC: rfc8484:4
* Don't accept edns-client-subnet, in any protocols.
* No proxy needed in China.
* Accuracy: not best result in China.

## Google

* Domain: dns.google.com
* IP: 8.8.8.8/8.8.4.4
* Accept protocols: udp/tcp/tcp-tls/google.
* Response time in Shanghai: udp:75, tcp:90, tcp-tls:250
* Response time in Japan IDC: udp:1, tcp:2, tcp-tls:40, google:5-100
* Accept edns-client-subnet with protocol google.
* A proxy will be needed for protocol google in China.

## OpenDNS

* Domain: dns.opendns.com
* IP: 208.67.222.222/208.67.220.220
* Accept protocols: udp/tcp
* Response time in Shanghai: udp:78, tcp:156
* Response time in Japan IDC: udp:1, tcp:2
* Don't accept edns-client-subnet, in any protocols.
* No proxy needed in China.

## Quad9

* Domain: dns.quad9.net
* IP: 9.9.9.9/149.112.112.112
* Accept protocols: udp/tcp/tcp-tls/rfc8484
* Response time in Shanghai: udp:85, tcp:170, tcp-tls:280, rfc8484: 80
* Response time in Japan IDC: udp:115, tcp:250, tcp-tls:370, rfc8484: 1
* Don't accept edns-client-subnet, in any protocols.
* No proxy needed in China.
* Accuracy: wrong result in China (taobao and baidu).

# Suggestions in China

1. Don't use Quad9. Wrong results means useless.
2. I won't suggest Cloudflare. Not the best result. Don't use it unless running out of other options.
3. Google/OpenDNS with udp (direct connect, will be interfered by the GFW). Find yourself a way to dodge the firewall.
4. Google with edns-client-subnet (proxy needed).
5. If you want tcp-tls, the first choice is Google (proxy needed), then Cloudflare (don't need proxy).
6. If you want rfc8484, the only option here is Cloudflare. Don't use Quad9.

# TODO

* cache
* record dns logs
* multiple outputs, load balance
