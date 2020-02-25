#!/bin/bash

# fix the hostname for kubernetes...
ipaddr=`hostname -i`
converted=`echo $ipaddr | sed 's/\./-/g'`
namespace=`cat /var/run/secrets/kubernetes.io/serviceaccount/namespace`
realhost=$converted.$namespace.pod

hostname $realhost

datadir=`echo $HDFS_CONF_dfs_datanode_data_dir | perl -pe 's#file://##'`
if [ ! -d $datadir ]; then
  echo "Datanode data directory not found: $dataedir"
  exit 2
fi

$HADOOP_PREFIX/bin/hdfs --config $HADOOP_CONF_DIR datanode
