source ./env.sh
for n in $ALL_NODES; do
	  ssh -q -t $n $*
  done

