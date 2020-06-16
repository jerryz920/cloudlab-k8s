
hostname | grep wisc > /dev/null
	linknames=`ls /sys/class/net`
	for l in $linknames; do
		ip addr show dev $l | grep 10.10.1 > /dev/null
		if [ $? -eq 0 ]; then
			echo "10.10.1 $l"
		fi
	done

