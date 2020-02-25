#!/bin/bash

for n in `seq 1 8`; do
docker-machine ssh worker-$n "$@"
done
docker-machine ssh master "$@"

