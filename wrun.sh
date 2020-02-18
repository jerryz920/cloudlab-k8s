source ./env.sh
for n in $W_NODES; do
	  ssh -q -t $n $*
  done

