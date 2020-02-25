#!/bin/bash

# Set RIAK_IP
RIAK_IP=`hostname -i`

curl -XPUT  http://${RIAK_IP}:8098/types/safesets/buckets/safe/keys/b5SCs-dUqRWMvs1GbwvwRC9Pi9yHYuSVj6oxLSU8wXs  -H 'Content-Type: text/plain'   -d 'herzlich willkommen'

curl http://${RIAK_IP}:8098/types/safesets/buckets/safe/keys/b5SCs-dUqRWMvs1GbwvwRC9Pi9yHYuSVj6oxLSU8wXs
# Expected respoonse: herzlich willkommen
