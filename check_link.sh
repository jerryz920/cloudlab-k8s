linknames=`sudo lldpctl -f json | jq '.lldp.interface | .[] | keys[0]' | tr -d \"`

for l in $linknames; do
  vlan=`sudo lldpctl -f json $l | jq '.lldp.interface.'$l'.vlan."vlan-id"'  | tr -d \"`
  echo "$vlan $l"
done

