#!/bin/bash

export SCRIPT_HOME=/local
source /local/env.sh
cd $SCRIPT_HOME

wget https://dl.google.com/go/go1.13.8.linux-amd64.tar.gz -O $OPENSTACK_DATA_PATH/gobinary.tar.gz
sudo apt-get install -y cmake

pushd .

cd $OPENSTACK_DATA_PATH
mkdir -p goroot
tar xf $OPENSTACK_DATA_PATH/gobinary.tar.gz -Cgoroot
mkdir -p go

echo "export GOROOT=$OPENSTACK_DATA_PATH/goroot/go" >> /usr/bin/goenv
echo "export GOPATH=$OPENSTACK_DATA_PATH/go" >> /usr/bin/goenv
echo "export PATH=\$PATH:/.local/bin:\$PATH:\$GOROOT/bin/:\$GOPATH/bin/" >> /usr/bin/goenv

echo "export GOROOT=$OPENSTACK_DATA_PATH/goroot/go" >> /root/.bashrc
echo "export GOPATH=$OPENSTACK_DATA_PATH/go" >> /root/.bashrc
echo "export PATH=\$PATH:/.local/bin:\$PATH:\$GOROOT/bin/:\$GOPATH/bin/" >> /root/.bashrc

export GOROOT=$OPENSTACK_DATA_PATH/goroot/go
export GOPATH=$OPENSTACK_DATA_PATH/go
export PATH=$PATH:$GOROOT/bin/:$GOPATH/bin/

go get github.com/biogo/store/interval
go get github.com/sirupsen/logrus
go get github.com/gorilla/mux
go get github.com/stretchr/testify
go get github.com/opencontainers/runc

