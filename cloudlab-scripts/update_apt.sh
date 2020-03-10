#!/bin/bash

killall -KILL apt-get
killall -KILL apt-get
killall -KILL apt-get
killall -KILL apt-get
killall -KILL dpkg
killall -KILL dpkg
killall -KILL dpkg
killall -KILL dpkg
killall -KILL dpkg
killall -KILL dpkg
killall -KILL dpkg
killall -KILL dpkg
killall -KILL dpkg
killall -KILL dpkg
killall -KILL dpkg
killall -KILL dpkg
killall -KILL dpkg
killall -KILL dpkg
sleep 5 

apt-get update

apt-get install -y ubuntu-cloud-keyring
apt-get install -y apt-transport-https ca-certificates
apt-get install -y curl git

curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu bionic stable"
apt-get install -y docker-ce


apt-get update
apt-get install -y syslog-ng-core
apt-get install -y syslog-ng
# harden the server
apt-get install -y fail2ban
apt-get install -y tmux bridge-utils

echo 'export BUILD_TIME="2020-02-15 17:16:49"' > /usr/bin/buildtime
chmod +x /usr/bin/buildtime

apt-get install -y python-pip libpython-dev
apt-get install -y python-dev
apt-get install -y libboost-python-dev

service fail2ban start
