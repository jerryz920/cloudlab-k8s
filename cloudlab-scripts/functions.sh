


to_upper()
{
  echo $* | awk '{print toupper($0)}'
}

to_lower()
{
  echo $* | awk '{print tolower($0)}'
}

get_mygw()
{
  #`ip route show default | sed -n -e 's/^default via \([0-9]*.[0-9]*.[0-9]*.[0-9]*\).*$/\1/p'`
  cat $EMULAB_CONF/routerip
}

get_myip0()
{
  grep "${NODE_ID}-ilink" /etc/hosts | awk '{print $1}'
}

get_myip1()
{
  # use as tunnel
  grep "${NODE_ID}-elink" /etc/hosts | awk '{print $1}'
}

get_ip0()
{
  grep "${1:-NODE_ID}-ilink" /etc/hosts | awk '{print $1}'
}

get_ip1()
{
  # use as tunnel
  grep "${1:-NODE_ID}-elink" /etc/hosts | awk '{print $1}'
}

get_mypublicip()
{
  cat $EMULAB_CONF/myip
}

get_mypublicmask()
{
  cat $EMULAB_CONF/mynetmask
}

ip_to_eth()
{
  local mygw=`get_mygw`
  local gateway_ip=${1:-$mygw}
  ifconfig | grep -B1 $gateway_ip | awk '{print $1; exit}'
}

fix_hostnames()
{
# adjust hosts file to make controller/network/compute... align to ilink, aka ip0
for n in $NODES; do
  node_ip0=`get_ip0 $n`
  node_ip1=`get_ip1 $n`
  sed -i -e '/'$node_ip0'/d' -e '/'$node_ip1'/d' /etc/hosts
  echo "$node_ip0 ${n}-ilink ${n}-0 ${n}" >> /etc/hosts
  echo "$node_ip1 ${n}-elink ${n}-1" >> /etc/hosts
done
}

# make sure env.sh has been sourced before call this

ssh_wait_for_event()
{
  local event=$1
  shift 1
  mkdir -p /tmp/ssh_events/
  chmod 777 /tmp/ssh_events/
  rm -f /tmp/ssh_events/$event
  for node in $*; do # wait for given node on certain event
    SUCCESS=1
    fqdn=$node.$EEID.$EPID.$OURDOMAIN
    while [ $SUCCESS -ne 0 ] ; do
	sleep 1
	scp -o ConnectTimeout=1 -o PasswordAuthentication=No -o NumberOfPasswordPrompts=0 -o StrictHostKeyChecking=No \
	  $fqdn:$HOME/ssh_events/$event /tmp/ssh_events/ 2>/dev/null
	if test -f /tmp/ssh_events/$event; then
	  SUCCESS=0
	fi
	echo 'still waiting for '$event 'on ' $node
    done
done
}

ssh_make_event()
{
  local event=$1
  mkdir -p $HOME/ssh_events/
  touch $HOME/ssh_events/$event
}

myapt_install()
{
local packages=$*
#while true; do
  sudo apt-get install -q -y $packages
#  if [ $? -eq 0 ]; then
#    break
#  fi
#  echo "apt error, sleep 5 sec, retry and wait for installing $packages"
#  sleep 5
#done
}

is_c220g1()
{
  local name=`cat /var/emulab/boot/tmcc/nodeid`
  if [[ $t =~ c220g1 ]]; then
    return 1
  fi
  return 0
}

