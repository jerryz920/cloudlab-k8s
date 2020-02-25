
# fix hosts
master=hdfs-1.latte.org
eval $(docker-machine env $master)
docker run --name=toolbox -it --rm -e HOST_RESOLVER=files_dns -e CLUSTER_NAME=std-hdfs --network host -v /home/ubuntu/spdata:/data -v /hadoop/dfs/name:/hadoop/dfs/name uhopper/hadoop-namenode bash
