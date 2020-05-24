# Table of content

* [Abstract](#abstract)
* [Compile and Install](#compile-and-install)
* [Command line options and args](#command-line-options-and-args)
* [Config](#config)
  * [Client Config](#client-config)
* [Drivers and Protocols](#drivers-and-protocols)
  * [dns](#dns)
  * [doh](#doh)
* [Public recursive server](#public-recursive-server)
  * [114](#114)
  * [Cloudflare one](#cloudflare-one)
  * [Cloudflare doh](#cloudflare-doh)
  * [Google](#google)
  * [OpenDNS](#opendns)
  * [Quad9](#quad9)
  * [NextDNS](#nextdns)
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
* service-driver: optional. see "drivers and protocols". if empty, program will auto guess.
* service-url: required. see "drivers and protocols".
* cert-file: optional. cert file when use doh.
* key-file: optional. key file when use doh.
* edns-client-subnet: optional. it could be empty, means don't do anything. or "client", means read remote address and put it into edns-client-subnet. or an ip address/cidr subnet, means put this address into edns-client-subnet. empty by default.
* client: client settings.
* aliases: a dict. if url matches the key, it will be replaced by value.

## Client Config

* driver: optional. determine which system will be used as a client. see "drivers and protocols". if empty, the program will auto guess.
* url: required. see "drivers and protocols".
* insecure: optional. don't verify the certificate from the server.

# Drivers and Protocols

## dns

There have three different protocols in driver DNS:

* udp
* tcp
* tcp-tls

Here are some examples as output.

	doh -q --url udp://114.114.114.114:53 www.baidu.com
	doh -q --url tcp://114.114.114.114:53 www.baidu.com
	doh -q --url tcp-tls://one.one.one.one:853 www.baidu.com

Here are some examples as input.

	doh --config data/udp-rfc8484.json
	dig www.baidu.com @127.0.0.1 -p 5053

## doh

DoH means DNS over HTTPS. It include two drivers:

* rfc8484
* google

As an output driver, you should indicate which driver exactly. We will guess if you don't say it explicitly.

Here are some examples as output.

	doh -q --url https://security.cloudflare-dns.com/dns-query www.baidu.com
	doh -q --url https://dns.google.com/resolve www.baidu.com

As an input protocol, doh are fine. We support both protocols on the same http/https server.

Here are some examples as input.

	doh --config data/http.json &
	doh -q --url http://localhost:8053/dns-query www.baidu.com
	doh -q --url http://localhost:8053/resolve www.baidu.com

	doh --config data/https.json &
	doh -q --url https://localhost:8153/dns-query --insecure www.baidu.com
	doh -q --url https://localhost:8153/resolve --insecure www.baidu.com

# Public recursive server

* [Public Recursive Servers](data/public.csv)
* [Shanghai Telecom](data/sh-telecom.md), [csv](data/sh-telecom.csv)
* [Japan IDC](data/jp.md), [csv](data/jp.csv)

source:

* https://blog.skk.moe/post/which-public-dns-to-use/
* https://github.com/curl/curl/wiki/DNS-over-HTTPS
* https://www.publicdns.xyz/

# Suggestions

1. Ignore those has an accuracy less than 4.
2. Ignore those has a latency more than 30.
3. 114, alidns, dnspai, dnspod are acceptable in China. baidu, onedns are a bit more than 30.
4. alidns and dnspod support edns client subnet, and alidns has the most wide protocol supportive in China.

1. Don't use Quad9, NextDNS, and AdGuard. Wrong results means useless.
2. I won't suggest Cloudflare. Not the best result. Don't use it unless running out of other options.
3. My first option is Google with tcp-tls. Accessible, accurate, easy.
4. Secend option is Google/OpenDNS with udp. Find yourself a way to dodge the firewall.
4. Google with edns-client-subnet (proxy needed) are barely acceptable.
5. If you want tcp-tls, the first choice is Google, then Cloudflare.
6. If you want rfc8484, the first choice is OpenDNS, then Cloudflare.

# TODO

* cache
* record dns logs
* multiple outputs, load balance

