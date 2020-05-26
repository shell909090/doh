# Table of content

* [Abstract](#abstract)
* [Compile and Install](#compile-and-install)
* [Command line options and args](#command-line-options-and-args)
* [Config](#config)
  * [Client Config](#client-config)
* [Drivers and Protocols](#drivers-and-protocols)
  * [dns](#dns)
  * [rfc8484](#rfc8484)
  * [google](#google)
  * [doh/http/https](#doh/http/https)
  * [twin](twin)
* [Public recursive server](#public-recursive-server)
  * [Summary in China](#summary-in-china)
  * [Summary outside China](#summary-outside-china)
* [Suggestions](#suggestions)
* [TODO](#todo)

# Abstract

[DNS over HTTPS](https://en.wikipedia.org/wiki/DNS_over_HTTPS) utils written by golang.

# Compile and Install

	make

The executable file is placed under `bin/`. Copy it to wherever you like. Enjoy.

# Command line options and args

See `doh --help`.

# Config

Defaultly doh will try to read configs from `doh.json;~/.doh.json;/etc/doh.json`.

* logfile: optional. indicate which file log should be written to. empty means stdout. empty by default.
* loglevel: optional. log level. warning by default.
* service: service config
  * driver: driver to use.
  * url: url to driver.
  * ... the rest of the config depends on the driver.
* client: client config
  * driver: driver to use.
  * url: url to driver.
  * ... the rest of the config depends on the driver.

## aliases

Defaultly doh will try to read aliases from `doh-aliases.json;~/.doh-aliases.json`.

If the server from the command line matches the key, the value will be used.

## Client Config

* driver: optional. determine which system will be used as a client. see "drivers and protocols". if empty, the program will auto guess.
* url: required. see "drivers and protocols".
* insecure: optional. don't verify the certificate from the server.

# Drivers and Protocols

## dns

There are three different protocols in driver `dns`:

* udp: default port 53
* tcp: default port 53
* tcp-tls: default port 853

This driver can be used in both client and server settings.

Client Config:

* timeout: as its name.

Server Config:

* ednsclientsubnet: as its name.
* certfile: file path of certificates.
* certkeyfile: file path of key.

## rfc8484

There is one protocol in driver `rfc8484`. 

* https: default port 443, default path is `/dns-query`.

This driver can only be used in client setting.

Client Config:

* insecure: don't check the certificates.

## google

There is one protocol in driver `rfc8484`. 

* https: default port 443, default path is `/resolve`.

This driver can only be used in client setting.

Client Config:

* insecure: don't check the certificates.

## doh/http/https

This driver can only be used in server setting. It supports both `rfc8484` and `google`.

Server Config:

* ednsclientsubnet: as its name. if it's `client`, then the actual client IP address will be put into the field.
* certfile: file path of the certificates.
* keyfile: file path of the key.

## twin

This driver can only be used in client setting.

Client Config:

* primary: another client config.
* secondary: another client config.
* direct-routes: a route file.

The quiz will be sent to the primary. If none of the answers match any routes in `direct-routes`, the quiz will be sent to the secondary and we return the answers from the secondary. Otherwise the answers from the primary will be used.

# Public recursive server

* [Public Recursive Servers](data/public.csv)
* [Shanghai Telecom](data/sh-telecom.md), [csv](data/sh-telecom.csv)
* [Shanghai Unicom](data/sh-unicom.md), [csv](data/sh-unicom.csv)
* [Japan IDC](data/jp.md), [csv](data/jp.csv)
* [Seattle IDC](data/seattle.md), [csv](data/seattle.csv)

source:

* https://blog.skk.moe/post/which-public-dns-to-use/
* https://github.com/curl/curl/wiki/DNS-over-HTTPS
* https://www.publicdns.xyz/

## Summary in China

| name | sh tc latency | sh tc accuracy | sh uc latency | sh uc accuracy |
| ---- | ------------- | -------------- | ------------- | -------------- |
| 114 | 17.2 | 12.2 | 274.2 | 20.3 |
| alidns | 9.2 | 19.7 | 259.2 | 18.0 |
| baidu | 41.2 | 17.8 | 287.8 | 19.3 |
| cnnic | 12.2 | 28.3 | 277.0 | 33.6 |
| dnspai | 6.8 | 17.7 | 265.8 | 19.2 |
| dnspod | 8.2 | 14.3 | 274.6 | 18.2 |
| dyn | 36.2 | 42.7 | 256.4 | 80.6 |
| google | 52.4 | 19.3 | 265.2 | 23.5 |
| onedns | 39.4 | 13.2 | 257.0 | 18.0 |
| opendns | 59.2 | 15.8 | 253.8 | 36.4 |

* Ignore those that have an accuracy more than 30.
* Ignore those that have a latency more than 30.
* alidns, dnspod, and google support edns client subnet, at least in some way.
* cnnic hasn't been poisoned, at least not to twitter.
* alidns has the most wide protocol supportive in China.
* In China Unicom, sometime the latency of TCP are less than the latency of UDP. Have no idea why it been like that.

## Summary outside China

| name | jp latency | jp accuracy | seattle latency | seattle accuracy |
| ---- | ---------- | ----------- | --------------- | ---------------- |
| adguard | 0.0 | 13.8 | 0.0 | 214.4 |
| cloudflare | 1.6 | 16.1 | 0.8 | 215.6 |
| containerpi | 71.8 | 16.3 | 367.4 | 1454.7 |
| google | 4.4 | 13.4 | 41.4 | 225.511 |
| he | 0.2 | 18.2 | 4.4 | 11.7 |
| nextdns | 0.4 | 12.9 | 0.6 | 14864.2 |
| opendns | 3.4 | 13.4 | 36.8 | 219.376 |
| quad9 | 110.0 | 1194.3 | 0.6 | 214.9 |
| safedns | 72.6 | 12.9 | 0.8 | 213.2 |

* Ignore those that have an accuracy more than 30.
* Ignore those that have a latency more than 30.
* google support edns client subnet.
* Most of the dns get a dramatically high score in seattle accuracy test, because they get the same result for tmall. `47.246.24.233`. `he` get a very close result, which is `47.246.18.236`. The latency of `47.246.24.233` is 66.7, and the latency of `47.246.18.236` is 0.6.

# Suggestions

4. adguard, cloudflare, comodo, google, he, nextdns, opennic, safedns are acceptable in Japan. dnspod are almost 50. opendns are more than 40.
5. adguard, google support edns client subnet, and they have the most wide protocol supportive in China. cloudflare, nextdns also support 4 protocols, except they don't support edns client subnet.
6. Seattle has almost the same situation as Japan. Except dyn becomes acceptable, and opennic becomes unacceptable.
7. If you are in China. alidns is the best choice you have. And if you are not in China, adguard and google are the best.

# TODO

* cache
* record dns logs
* multiple outputs, load balance

