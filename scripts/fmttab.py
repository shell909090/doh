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

def main():
    print(f'| vendor | protocol | latency | poisoned | edns client subnet | accuracy |')
    print(f'| ------ | -------- | ------- | -------- | ------------------ | -------- |')
    with open(sys.argv[1]) as fi:
        reader = csv.reader(fi)
        for name, prot, latency, poisoned, edns_subnet, accuracy in reader:
            print(f'| {name} | {prot} | {latency} | {poisoned} | {edns_subnet} | {accuracy} |')


if __name__ == '__main__':
    main()
