
hostname | grep wisc > /dev/null
if [ $? -eq 0 ]; then
	# on wisconsin we can determine it via vlan and lldp. Not sure why it does not work on Utah

	linknames=`sudo lldpctl -f json | jq '.lldp.interface | .[] | keys[0]' | tr -d \"`

	for l in $linknames; do
		vlan=`sudo lldpctl -f json $l | jq '.lldp.interface.'$l'.vlan."vlan-id"'  | tr -d \"`
		echo "$vlan $l"
	done
else

	linknames=`ls /sys/class/net`
	for l in $linknames; do
		ip addr show dev $l | grep 10.10.1 > /dev/null
		if [ $? -eq 0 ]; then
			echo "10.10.1 $l"
		fi
	done
fi


