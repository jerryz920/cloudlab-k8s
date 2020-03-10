#!/bin/bash

export SCRIPT_FULL_PATH=`readlink -f $0`
DETECTED_SCRIPT_HOME=`dirname $0`
export SCRIPT_HOME=${SCRIPT_HOME:-$DETECTED_SCRIPT_HOME}

. $SCRIPT_HOME/env.sh # this will set the script home
. $SCRIPT_HOME/functions.sh
fix_hostnames
# setup ntp time of controller
bash $SCRIPT_HOME/update_apt.sh
bash $SCRIPT_HOME/setup_root_ssh.sh

# setup lvm and default storage
bash $SCRIPT_HOME/setup_base.sh

bash $SCRIPT_HOME/setup_ntp.sh ${NODE_ID:-controller}

bash $SCRIPT_HOME/move_docker.sh
