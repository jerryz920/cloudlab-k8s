#!/bin/bash

bash pack.sh # make sure we pack the latest things
sleep 2
domain=${1:-utah}
config=${2:-config}
user=qiangcao
grep controller $config > controller
grep compute $config > compute
grep network $config > network
grep storage $config > storage

install_one()
{
  local addr=$1
  local nodetype=$2
  ssh -ttt $user@$addr.$domain.cloudlab.us "
		sudo mkdir -p /local/;
		cd /local;
		sudo rm *.tgz;"
  scp cloudlab-install.tgz $user@$addr.$domain.cloudlab.us:
  ssh -ttt $user@$addr.$domain.cloudlab.us "
		sudo mv cloudlab-install.tgz /local/;
		cd /local;
		sudo tar xf cloudlab-install.tgz;
		sudo bash setup_${nodetype}_dev.sh "
}

start_install()
{
  local addrs=`awk '{print $2}' $1`
  local nodetype=${2:-controller}

  for addr in $addrs; do
  	local name=`grep $addr $1 | awk '{print $1}'`
	echo 'installing' $nodetype on $name
	install_one $addr $nodetype >log-$name.out 2>&1 &
  done
}

upload_aws controller controller
start_install controller controller
start_install compute compute
start_install network network

  for p in `jobs -p`;
  do
    wait $p
  done
