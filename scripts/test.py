#!/usr/bin/python3
# -*- coding: utf-8 -*-
'''
@date: 2020-05-24
@author: Shell.Xu
@copyright: 2020, Shell.Xu <shell909090@gmail.com>
@license: BSD-3-clause
'''
import sys
import csv
import random
import ipaddress
import subprocess

import pprint

from multiprocessing.pool import ThreadPool

# 101.0.0.0/8 // sh telecom
# 52.88.0.0/13 // aws canada
# 104.244.40.0/21 // twitter

accuracy_domains = [
    'www.taobao.com', 'www.tmall.com', 'www.qq.com', 'www.baidu.com', 'www.sohu.com', 'www.jd.com',
    'www.amazon.com', 'www.bing.com', 'www.linkedin.com', 'www.weibo.com', 'www.meituan.com']


def result_parse(p):
    for rec in p.stdout.decode('utf-8').splitlines():
        try:
            rec = rec.strip()
            yield ipaddress.ip_address(rec)
        except ValueError:
            if 'ERROR' in rec or 'WARNING' in rec:
                print(rec, file=sys.stderr)


def repeat_ips(num, cmd):
    ips = set()
    for _ in range(num):
        p = subprocess.run(cmd, capture_output=True)
        ips |= set(result_parse(p))
    return ips


ping_cache = {}
def ping(ip):
    if ip in ping_cache:
        return ping_cache[ip]
    p = subprocess.run(
        ['ping', '-qc', '5', '-i', '0.2', str(ip)],
        capture_output=True)
    for line in p.stdout.decode('utf-8').splitlines():
        line = line.strip()
        if not line.startswith('rtt'):
            continue
        rtt = line.split('=')[1].split('/')
        ping_cache[ip] = float(rtt[0])  # take the min
        break
    if ip not in ping_cache:
        print(f'ping ip {ip} has no response', file=sys.stderr)
        ping_cache[ip] = 10000
    return ping_cache[ip]


def test_available(row):
    name, prot, url = row
    driver = prot
    if prot in ('udp', 'tcp', 'tls'):
        driver = 'dns'
    ips = repeat_ips(2, ['bin/doh', '-short', '-driver', driver, '-s', url, 'www.amazon.com'])
    if not ips:
        writer.writerow((name, prot, 'not available'))
    else:
        return (name, prot, driver, url)


accuracies = {}
def test_accuracy(domain):
    global accuracies
    rslt = {}
    avgs, mins = [], []
    for row in servers:
        name, prot, driver, url = row
        ips = repeat_ips(2, ['bin/doh', '-short', '-insecure', '-driver', driver, '-s', url, domain])
        if not ips:
            print(name, prot, domain, 'fuck off', file=sys.stderr)
            mins.append(10000)
            avgs.append(10000)
            continue
        latency = [ping(ip) for ip in ips]
        mins.append(min(latency))
        avgs.append(sum(latency) / len(latency))
    min_latency = min(mins)
    for row, latency in zip(servers, avgs):
        rslt[row] = latency/min_latency
    accuracies[domain] = rslt


def test_latency(driver, url):
    latency = []
    for _ in range(5):
        p = subprocess.run(
            ['bin/doh', '-driver', driver, '-s', url, 'www.amazon.com'],
            capture_output=True)
        for rec in p.stdout.decode('utf-8').splitlines():
            if not rec.startswith(";; Query time:"):
                continue
            latency.append(int(rec.strip().split(':', 1)[1].split()[0]))
    if len(latency) != 0:
        return sum(latency)/len(latency)
    else:
        print(f"{driver} {url} has no query time: {p.stdout}", file=sys.stdout)


def check_poisoned(driver, url):
    cidr = ipaddress.ip_network('104.244.40.0/21')
    p = subprocess.run(
        ['bin/doh', '-short', '-insecure', '-driver', driver, '-s', url, 'www.twitter.com'],
        capture_output=True)
    return not all([(ip in cidr) for ip in result_parse(p)])


def test_edns_subnet(driver, url):
    ips1 = repeat_ips(3, ['bin/doh', '-short', '-insecure', '-driver', driver, '-s', url,
                          '-subnet', '101.80.0.0', 'www.amazon.com'])
    ips2 = repeat_ips(3, ['bin/doh', '-short', '-insecure', '-driver', driver, '-s', url,
                          '-subnet', '52.88.0.0', 'www.amazon.com'])
    print(f'edns {url} {ips1} {ips2}', file=sys.stderr)
    return bool(ips1) and bool(ips2) and not bool(ips1 & ips2)


def test_all(row):
    name, prot, driver, url = row
    latency = test_latency(driver, url)
    if latency is None:
        writer.writerow((name, prot, 'not available'))
        return
    poisoned = 'Yes' if check_poisoned(driver, url) else 'No'
    edns_subnet = 'Yes' if test_edns_subnet(driver, url) else 'No'
    accuracy = [accuracies[domain][row] for domain in accuracy_domains]
    rslt = [name, prot, latency, poisoned, edns_subnet, '%0.3f' % sum(accuracy)]
    writer.writerow(rslt + ['%0.2f' % score for score in accuracy])


def main():
    global writer
    writer = csv.writer(sys.stdout)

    global servers
    with open(sys.argv[1]) as fi:
        servers = list(csv.reader(fi))
    random.shuffle(servers)

    pool = ThreadPool(3)
    servers = [row for row in pool.map(test_available, servers) if row]
    pool.map(test_accuracy, accuracy_domains)
    pool.map(test_all, servers)


if __name__ == '__main__':
    main()
