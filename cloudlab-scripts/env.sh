# All the environment goes here


export USERID=qiangcao
export EMULAB_CONF=/var/emulab/boot/
export OPENSTACK_VOLUME_GROUP="openstack"
export OPENSTACK_VOLUME="openstack-data"
export OPENSTACK_DATA_PATH="/openstack"
export OPENSTACK_DEV_PATH="/openstack/srcs/"
export OPENSTACK_GIT_PATH="/openstack/git/"
export BOOTDIR=$EMULAB_CONF
export BOOT_DIR=$BOOTDIR
export EEID=`cat $BOOTDIR/nickname | cut -d . -f 2`
export EPID=`cat $BOOTDIR/nickname | cut -d . -f 3`
export tmpid=`cat $BOOTDIR/nickname | cut -d . -f 1`
export NODE_ID=${NODE_ID:-$tmpid}
export NODEID=$NODE_ID # for some typo in other scripts
export TOPOMAP=$BOOTDIR/topomap
export SWAPPER=`cat $BOOTDIR/swapper`
export OURDOMAIN=`cat $BOOTDIR/mydomain`
export NODES=`cat $TOPOMAP | grep -v '^#' | sed -n -e 's/^\([a-zA-Z0-9\-]*\),.*:.*$/\1/p' | xargs`
export COMPUTE_NODES=""
export COMPUTE_ILINKS=""
export COMPUTE_ELINKS=""
export STORAGE_NODES=""
export STORAGE_ILINKS=""
export STORAGE_ELINKS=""
for node in $NODES; do
  if [[ $node =~ compute* ]]; then
    export COMPUTE_NODES="$node $COMPUTE_NODES"
    export COMPUTE_ILINKS="$node-ilink $COMPUTE_ILINKS"
    export COMPUTE_ELINKS="$node-elink $COMPUTE_ELINKS"
  elif [[ $node =~ storage* ]]; then
    export STORAGE_NODES="$node $STORAGE_NODES"
    export STORAGE_ILINKS="$node-ilink $STORAGE_ILINKS"
    export STORAGE_ELINKS="$node-elink $STORAGE_ELINKS"
  fi
done
export TMCC=/usr/local/etc/emulab/tmcc
export HOSTNAME=`cat /var/emulab/boot/nickname | cut -f1 -d.`
#
# Grab our topomap so we can see how many nodes we have.
#
if [ ! -f $TOPOMAP ]; then
    $TMCC topomap | gunzip > $TOPOMAP
fi

export OPENSTACK_DEV_REPO='http://pages.cs.wisc.edu/~yanzhai/openstack_repos'

cp $SCRIPT_HOME/etc/user1rc /usr/bin/
cp $SCRIPT_HOME/etc/user2rc /usr/bin/
cp $SCRIPT_HOME/etc/user3rc /usr/bin/
cp $SCRIPT_HOME/etc/adminrc /usr/bin/


source $SCRIPT_HOME/functions.sh
export PATH=$PATH:$SCRIPT_HOME
export DEBIAN_FRONTEND=noninteractive


