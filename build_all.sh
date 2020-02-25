(
echo "Building K8s"
cd kubernetes
# TODO: put k8s build script things here.
KUBE_FASTBUILD=true build/run.sh make kube-apiserver kube-proxy kube-controller-manager kube-scheduler kubectl kubelet kubeadm
if [ $? -ne 0 ]; then
	echo "build fail!"
	exit 1
fi
make quick-release
# We can do make package later for faster rebuild. But now we need quick-release to get everything done
#KUBE_FASTBUILD=true make package
docker image rm kube-apiserver
tag=`hack/print-workspace-status.sh | grep DOCKER_TAG | awk '{print $2}'`
# Delete first then reload
docker image rm k8s.gcr.io/kube-apiserver-amd64:$tag
docker image rm k8s.gcr.io/kube-apiserver:$tag
docker image load -i _output/release-images/amd64/kube-apiserver.tar
bash wcp.sh _output/release-images/amd64/kube-apiserver.tar
bash wrun.sh "docker image load -i kube-apiserver.tar;"

# edit kubelet config to reload
sed -i 's/image: .\+/image: k8s.gcr.io\/kube-apiserver-amd64:'$tag'/' /etc/kubernetes/manifests/kube-apiserver.yaml

# wait for api server to reload.
sleep 5

bash wrun.sh "sudo systemctl stop kubelet;"
for n in kubelet kubeadm kubectl; do
  make -j $n
  bash wcp.sh _output/bin/$n
  bash wrun.sh "sudo cp $n /usr/bin/"
done
bash wrun.sh "sudo systemctl start kubelet;"

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
make
)

(
echo "Building Riak Image"
cd riak-image
docker build -t riak .
)
