#!/bin/bash


# set up storage things
#setup_lvm
#allocate_volume $OPENSTACK_VOLUME 400g
#dev_path=/dev/$OPENSTACK_VOLUME_GROUP/$OPENSTACK_VOLUME

workaround_partition() {
  	parted /dev/sda mkpart primary 40GB 1200GB -s
	if [ $? -ne 0 ]; then
	  echo 'can not make data partition, assuming existed'
	fi
	parted /dev/sda print | awk '$1==2||$1==4||$1==3{print $1,$4}' | while read dataline
      do
	size=`echo $dataline | awk '{print $2}'`
	v=`python $SCRIPT_HOME/compare.py $size 100GB`
	if [ $v -eq 2 ]; then
	   part=`echo $dataline | awk '{print $1}'`
	   dev_path=/dev/sda$part
	   mkfs.ext4 $dev_path
	   mkdir -p $OPENSTACK_DATA_PATH
	   # this will be used as mysql and meta things
	   mount $dev_path $OPENSTACK_DATA_PATH
	   echo $dev_path'	'$OPENSTACK_DATA_PATH'	ext4	noatime	1	2' >> /etc/fstab
	fi
      done
}


#if [ is_c220g1 -eq 1 ]; then
  sudo fdisk -l | grep sda4
if [ $? -eq 0 ]; then
  dev_path=/dev/sda4
else
  dev_path=/dev/sdc4
fi
  mkfs.ext4 $dev_path
  mkdir -p $OPENSTACK_DATA_PATH
  # this will be used as mysql and meta things
  mount $dev_path $OPENSTACK_DATA_PATH
  echo '$dev_path        '$OPENSTACK_DATA_PATH'  ext4    noatime 1       2' >> /etc/fstab
#else
#  workaround_partition
#fi



