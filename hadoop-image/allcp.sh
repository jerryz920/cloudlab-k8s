#!/bin/bash
for m in $@; do
for n in `seq 1 4`; do
docker-machine scp $m hdfs-$n.latte.org:
done
done

