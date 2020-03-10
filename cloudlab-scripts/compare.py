#!/usr/bin/python

import sys
import os

suffix_map = {
    "gb": 1000**3,
    "g": 1000**3,
    "mb":  1000**2,
    "m": 1000**2,
    "kb": 1000,
    "k": 1000,
    }


def convert(s):
    for suffix, ratio in suffix_map.iteritems():
        if s.endswith(suffix):
            return int(s[:-len(suffix)]) * ratio
    if s.endswith('b'):
        return int(s[:-1])
    return int(s)


if __name__ == "__main__":
    if len(sys.argv) < 3:
        print 0
        sys.exit(1)
    s1 = convert(sys.argv[1].lower())
    s2 = convert(sys.argv[2].lower())
    if s1 > s2:
        print 2
    elif s1 == s2:
        print 1
    else:
        print 0


