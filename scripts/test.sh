#!/bin/bash

function basic() {
    echo "version"
    bin/doh -version
    echo "resolv.conf"
    bin/doh www.baidu.com
    echo "short"
    bin/doh -short www.baidu.com
    echo "json"
    bin/doh -json www.baidu.com | jq -r '.Answer[].data'
    echo "url"
    bin/doh -short -s udp://114.114.114.114 www.baidu.com
    echo "tcp"
    bin/doh -short -tcp -s 114.114.114.114 www.baidu.com
    echo "alias"
    bin/doh -short -s 114 www.baidu.com
    echo "at server"
    bin/doh -short @114 www.baidu.com
    echo "retries"
    bin/doh -short -tries 3 @114 @114t www.baidu.com
    echo "QType"
    bin/doh -short -t NS @114 baidu.com
    bin/doh -short @114 NS baidu.com
    echo "QClass"
    bin/doh -short -c IN @114 www.baidu.com
    bin/doh -short @114 IN www.baidu.com
    echo "timeout"
    bin/doh -s 114 -timeout 1 www.baidu.com
    echo "query with config"
    bin/doh -short -q -config data/twin.json www.baidu.com
}

function drivers() {
    echo "drivers, except google"
    bin/doh -short -s 114 www.baidu.com
    bin/doh -short -s 114t www.baidu.com
    bin/doh -short -s cftls www.baidu.com
    bin/doh -short -s cfdoh www.baidu.com
    bin/doh -short -s gdnstls www.baidu.com
    bin/doh -short -s http://119.29.29.29/d www.baidu.com
}

function edns() {
    bin/doh -short -s alidns www.taobao.com
    bin/doh -short -s alidns -subnet 101.80.0.0 www.taobao.com
    bin/doh -short -s alidns -subnet 104.244.42.1 www.taobao.com
    bin/doh -short -s dnspod www.taobao.com
    bin/doh -short -s dnspod -subnet 101.80.0.0 www.taobao.com
    bin/doh -short -s dnspod -subnet 104.244.42.1 www.taobao.com
}

function rfc8484() {
    bin/doh -config data/rfc8484.json &
    sleep 1
    bin/doh -short -s udp://127.0.0.1:5053 www.taobao.com
    killall doh
}

function google() {
    bin/doh -config data/google.json &
    sleep 1
    bin/doh -short -s udp://127.0.0.1:5153 www.taobao.com
    killall doh
}

function dnspod() {
    bin/doh -config data/dnspod.json &
    sleep 1
    bin/doh -short -s udp://127.0.0.1:5253 www.taobao.com
    killall doh
}

function http() {
    bin/doh -config data/http.json &
    sleep 1
    bin/doh -short -s http://localhost:8053/dns-query www.baidu.com
    bin/doh -short -s http://localhost:8053/resolve www.baidu.com
    bin/doh -short -s http://localhost:8053/d www.baidu.com
    curl -s "http://localhost:8053/resolve?name=www.baidu.com" | jq -r '.Answer[].data'
    curl -s "http://localhost:8053/d?dn=www.baidu.com&ttl=1"
    echo ""
    killall doh
}

function https() {
    bin/doh -config data/https.json &
    sleep 1
    bin/doh -short -s https://localhost:8153/dns-query -insecure www.baidu.com
    bin/doh -short -s https://localhost:8153/resolve -insecure www.baidu.com
    curl -s -k "https://localhost:8153/resolve?name=www.baidu.com" | jq -r '.Answer[].data'
    curl -s -k "https://localhost:8153/d?dn=www.baidu.com&ttl=1"
    echo ""
    killall doh
}

$@
