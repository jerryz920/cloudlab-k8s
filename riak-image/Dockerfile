from ubuntu:14.04

RUN apt-get update && apt-get install -y build-essential
RUN apt-get install -y gdb
RUN apt-get install -y default-jdk
RUN apt-get install -y git curl make htop vim emacs wget

RUN apt-get install -y libssl1.0.0
RUN apt-get install -y libpam0g-dev
RUN apt-get install -y libssl-dev

COPY start.sh   /root/
COPY test.sh    /root/

# install riak
RUN wget http://s3.amazonaws.com/downloads.basho.com/riak/2.1/2.1.4/ubuntu/trusty/riak_2.1.4-1_amd64.deb
RUN dpkg -i riak_2.1.4-1_amd64.deb

ENTRYPOINT ["/bin/bash", "/root/start.sh"]
#ENTRYPOINT ["/bin/sh", "-c"]
