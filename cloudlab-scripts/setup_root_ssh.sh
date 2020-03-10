#!/bin/sh

##
## Setup a root ssh key on the calling node, and broadcast it to all the
## other nodes' authorized_keys file.
##

set -x

# Gotta know the rules!
if [ $EUID -ne 0 ] ; then
    echo "This script must be run as root" 1>&2
    exit 1
fi

# Grab our libs
#. "`dirname $0`/setup-lib.sh"

# Make ourself a keypair; this gets copied to other roots' authorized_keys
#if [ ! -f /root/.ssh/id_rsa ]; then
#	we will make sure it uses our keys, since this is also used for git access
#
	echo 'StrictHostKeyChecking no' >> ~/.ssh/config
	echo 'StrictHostKeyChecking no' >> /root/.ssh/config
	chmod 600 ~/.ssh/config
	chmod 600 /root/.ssh/config
	cp $SCRIPT_HOME/keys/id_rsa ~/.ssh/id_rsa
	cp $SCRIPT_HOME/keys/id_rsa.pub ~/.ssh/id_rsa.pub
	cp $SCRIPT_HOME/keys/id_rsa /root/.ssh/id_rsa
	cp $SCRIPT_HOME/keys/id_rsa.pub /root/.ssh/id_rsa.pub
	chmod 600 ~/.ssh/id_rsa
	chmod 600 /root/.ssh/id_rsa
	chown -R $USERID ~$USERID/.ssh
#fi

if [ "$SWAPPER" = "geniuser" ]; then
    SHAREDIR=/proj/$EPID/exp/$EEID/tmp

    cp /root/.ssh/id_rsa.pub $SHAREDIR/$HOSTNAME

    for node in $NODES ; do
	while [ ! -f $SHAREDIR/$node ]; do
            sleep 1
	done
	echo $node is up
	cat $SHAREDIR/$node >> /root/.ssh/authorized_keys
    done
else
    for node in $NODES ; do
	if [ "$node" != "$HOSTNAME" ]; then 
	    fqdn="$node.$EEID.$EPID.$OURDOMAIN"
	    SUCCESS=1
	    while [ $SUCCESS -ne 0 ]; do
		su -c "$SSH  -l $SWAPPER $fqdn sudo tee -a /root/.ssh/authorized_keys" $SWAPPER < /root/.ssh/id_rsa.pub
		SUCCESS=$?
		sleep 1
	    done
	fi
    done
fi

exit 0
