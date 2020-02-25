
# fix hosts

cat > /tmp/cleanup.sh <<EOF
test_and_insert_host(){
  local name=\$1
  sed -i '/'\$1'/d' /etc/hosts
  echo "\$2 \$1" >> /etc/hosts
}
test_and_insert_host hdfs-1 192.4.0.3 
test_and_insert_host hdfs-2 192.4.0.4 
test_and_insert_host hdfs-3 192.4.0.5 
test_and_insert_host hdfs-4 192.4.0.6 
test_and_insert_host hdfs-1.latte.org 192.4.0.3 
test_and_insert_host hdfs-2.latte.org 192.4.0.4 
test_and_insert_host hdfs-3.latte.org 192.4.0.5 
test_and_insert_host hdfs-4.latte.org 192.4.0.6 
rm -rf /hadoop/dfs/data/* /hadoop/dfs/name/*
mkdir -p /hadoop/dfs/data /hadoop/dfs/name
chown -R ubuntu:ubuntu /hadoop/dfs/
EOF

bash allcp.sh /tmp/cleanup.sh
bash allrun.sh "sudo bash cleanup.sh"
rm -f /tmp/cleanup.sh
master=hdfs-1.latte.org
eval $(docker-machine env $master)
docker run --name=hdfs-name -dt --restart=always -e HOST_RESOLVER=files_dns -e CLUSTER_NAME=std-hdfs --network host -v /hadoop/dfs/name:/hadoop/dfs/name uhopper/hadoop-namenode
for w in `seq 1 4`; do
  eval $(docker-machine env hdfs-$w.latte.org)
docker run --name=hdfs-data-$w -dt --restart=always -e HOST_RESOLVER=files_dns -e CORE_CONF_fs_defaultFS=hdfs://hdfs-1.latte.org:8020 --network host -v /hadoop/dfs/data:/hadoop/dfs/data uhopper/hadoop-datanode
done

