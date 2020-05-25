#!/usr/bin/python3
# -*- coding: utf-8 -*-
'''
@date: 2020-05-24
@author: Shell.Xu
@copyright: 2020, Shell.Xu <shell909090@gmail.com>
@license: BSD-3-clause
'''
import re
import sys
import csv

accuracy_domains = [
    'taobao', 'tmall', 'qq', 'baidu', 'sohu', 'jd',
    'amazon', 'bing', 'linkedin', 'weibo', 'meituan']

def main():
    re_name = re.compile('[a-z\-]')
    s = f'| vendor | protocol | latency | poisoned | edns-client-subnet | accuracy | %s |' % ' | '.join(accuracy_domains)
    print(s)
    print(re_name.sub('-', s))
    with open(sys.argv[1]) as fi:
        reader = csv.reader(fi)
        for row in reader:
            print('| %s |' % ' | '.join(row))


if __name__ == '__main__':
    main()
