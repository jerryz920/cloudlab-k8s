

service docker stop
mkdir $OPENSTACK_DATA_PATH/docker
if [ -d /var/lib/docker ] ; then
  mv /var/lib/docker/* $OPENSTACK_DATA_PATH/docker/
fi
rm -rf /var/lib/docker
if [ $? -ne 0 ]; then
# a weird workaround
umount /var/lib/docker
rm -rf /var/lib/docker
fi
ln -s $OPENSTACK_DATA_PATH/docker /var/lib/docker
service docker start
