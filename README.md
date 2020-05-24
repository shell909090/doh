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

Server Config:

* ednsclientsubnet: as its name.

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
* [Japan IDC](data/jp.md), [csv](data/jp.csv)
* [Seattle IDC](data/seattle.md), [csv](data/seattle.csv)

source:

* https://blog.skk.moe/post/which-public-dns-to-use/
* https://github.com/curl/curl/wiki/DNS-over-HTTPS
* https://www.publicdns.xyz/

# Suggestions

1. Ignore those that have an accuracy less than 4. Ignore those that have a latency more than 30.
2. 114, alidns, dnspai, dnspod are acceptable in China. baidu, onedns are a bit more than 30.
3. alidns and dnspod support edns client subnet, and alidns has the most wide protocol supportive in China.
4. adguard, cloudflare, comodo, google, he, nextdns, opennic, safedns are acceptable in Japan. dnspod are almost 50. opendns are more than 40.
5. adguard, google support edns client subnet, and they have the most wide protocol supportive in China. cloudflare, nextdns also support 4 protocols, except they don't support edns client subnet.
6. Seattle has almost the same situation as Japan. Except dyn becomes acceptable, and opennic becomes unacceptable.
7. If you are in China. alidns is the best choice you have. And if you are not in China, adguard and google are the best.

# TODO

* cache
* record dns logs
* multiple outputs, load balance

