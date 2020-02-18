source ./env.sh
if [ $# -ge 3 ]; then
	echo "accept 1 or 2 arguments"
	exit 1
fi

for n in $ALL_NODES; do
  scp -r $1 $n:$2
done
