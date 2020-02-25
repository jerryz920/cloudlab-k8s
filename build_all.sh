(
echo "Building K8s"
cd kubernetes
# TODO: put k8s build script things here.
)

(
echo "Building Hdfs"
cd hadoop
bash docker-build.sh
)

(
echo "Building Hdfs-Docker-Images"
cd hadoop-image
bash buildhdfs.sh
)

(
echo "Building Spark and Spark Docker Images"
cd spark
dev/make-distribution.sh --tgz --name safe-spark -Phadoop-2.7 -Dscala-2.11
bin/docker-image-tool.sh -t v2.3 build
)

(
# There is no simple way of setting up docker repository, so we load them on
# every node.
echo "Loading Docker Images"
wload() {
  docker image save $1 > $2.tar
  bash wcp.sh $2.tar
  bash wrun.sh "docker image rm $2; docker image load -i $2.tar; rm -r $2.tar"
  rm $2.tar
}

wload uhopper/hadoop-datanode dn
wload uhopper/hadoop-namenode nn
wload spark:v2.3 spark
)

(
echo "Building Latte Proxy"
cd proxy
)

(
echo "Building Safe"
cd safe
)

