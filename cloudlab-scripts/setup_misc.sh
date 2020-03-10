#!/bin/bash

source $SCRIPT_HOME/functions.sh
#setup rabbit mq
#myapt_install   rabbitmq-server
#rabbitmqctl add_user openstack sonar # i know this is bad, but we are not doing industrial deployment here.
## just in case of configure error
#rabbitmqctl add_user stack sonar 
#rabbitmqctl set_permissions openstack ".*" ".*" ".*"
#rabbitmqctl set_permissions stack ".*" ".*" ".*"

bash $SCRIPT_HOME/setup_java.sh


