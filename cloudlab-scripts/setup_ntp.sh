#!/bin/bash
source $SCRIPT_HOME/functions.sh
myapt_install ntp

if [[ $1"x" != controllerx ]]; then
  sed 's/SERVER_NAME/controller/' $SCRIPT_HOME/etc/ntp.conf > /etc/ntp.conf
fi

service ntp stop
rm /var/lib/ntp/ntp.conf.dhcp
service ntp start
