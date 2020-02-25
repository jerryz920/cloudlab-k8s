
# fix hosts
master=hdfs-1.latte.org
eval $(docker-machine env $master)
docker logs hdfs-name
for w in `seq 1 4`; do
  eval $(docker-machine env hdfs-$w.latte.org)
docker logs hdfs-data-$w
done

