
confirm() {
	echo $1
	read input
	if [[ $input != "y" ]]; then
		exit 0
	fi
}
sudo chown -R $USER:`id -g -n` ~/.m2

build_k8s() {
(
echo "Building K8s"
cd kubernetes
# TODO: put k8s build script things here.
#KUBE_FASTBUILD=true build/run.sh make kube-apiserver kube-proxy kube-controller-manager kube-scheduler kubectl kubelet kubeadm
KUBE_FASTBUILD=true make kube-apiserver kube-proxy kube-controller-manager kube-scheduler kubectl kubelet kubeadm
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
#bash wcp.sh _output/release-images/amd64/kube-apiserver.tar
#bash wrun.sh "docker image load -i kube-apiserver.tar;"

# edit kubelet config to reload
sudo sed -i 's/image: .\+/image: k8s.gcr.io\/kube-apiserver-amd64:'$tag'/' /etc/kubernetes/manifests/kube-apiserver.yaml
sudo sed -i -e '$a# Build hack force restart' /etc/kubernetes/manifests/kube-apiserver.yaml

# wait for api server to reload.
sleep 5
cd ..

bash allrun.sh "sudo systemctl stop kubelet;"
for n in kubelet kubeadm kubectl; do
  make -j $n -C kubernetes
  bash allcp.sh kubernetes/_output/bin/$n
  bash allrun.sh "sudo cp $n /usr/bin/"
done
bash allrun.sh "sudo systemctl start kubelet;"

)
}

build_hdfs() {

kubectl delete -f configs/hdfs.yml
(
echo "Building Hdfs"
cd hadoop
bash docker-build.sh
)

(
echo "Building Hdfs-Docker-Images"
cd hadoop-image
bash buildhdfs.sh

# setup the necessary temporary directory for Hadoop data/name node
cd ../
bash allrun.sh "sudo mkdir -p /openstack/hdfs-name /openstack/hdfs-data;"

echo "Relaunching HDFS cluster"
# remove old first if exists
)
}

build_spark() {

(
echo "Building Spark and Spark Docker Images"
cd spark
dev/make-distribution.sh --tgz --name safe-spark -Phadoop-2.7 -Dscala-2.11
bin/docker-image-tool.sh -t v2.3 build

echo "Provision Spark Cluster Credentials"
kubectl create -f configs/spark.yml
)
}

build_shield() {
(
echo "Building Shield Pod"

kubectl delete -f configs/shield.yml
# install hostname to use for shield
grep shield.latte.org /etc/hosts > /dev/null
if [ $? -ne 0 ]; then
bash allrun.sh "echo '127.0.0.1 shield.latte.org' | sudo tee -a /etc/hosts"
fi


cd shield
gen_cert() {
openssl req -new -nodes -newkey rsa:2048 -keyout $1.key -out $1.csr -subj "/O=users/CN=$1" -reqexts v3_req -config ../configs/shield.cnf

sudo openssl x509 -req -days 1000 -in $1.csr -CA $2.crt -CAkey $2.key -set_serial 0101 -out $1.crt -sha256 -extensions 'v3_req' -extfile ../configs/shield.cnf
}
docker build -t shield .
gen_cert shield /etc/kubernetes/pki/ca
bash create-creds.sh
sudo rm shield.csr shield.key shield.crt

)
}

wload() {
  docker image save $1 > $2.tar
  bash wcp.sh $2.tar
  bash wrun.sh "docker image rm $1; docker image load -i $2.tar; rm -r $2.tar"
  rm $2.tar
}

load_image() {
(
# There is no simple way of setting up docker repository, so we load them on
# every node.
echo "Loading Docker Images"

wload uhopper/hadoop-datanode dn
wload uhopper/hadoop-namenode nn
wload spark:v2.3 spark
)
}

build_misc() {
(
echo "Building Latte Proxy"
cd proxy
make
grep mds.latte.org /etc/hosts > /dev/null
if [ $? -ne 0 ]; then
  bash allrun.sh "echo '10.10.1.1 mds mds.latte.org' | sudo tee -a /etc/hosts"
fi
)

(
echo "Building Riak Image"
cd riak-image
docker build -t riak .
docker run -t --rm -d -p 8098:8098 -p 8087:8087 --name riak riak
)
}

TARGETS=${*:-"k8s hdfs spark misc shield image"}

for n in $TARGETS; do
  case $n in
    k8s) build_k8s;;
    hdfs) build_hdfs;
      wload uhopper/hadoop-datanode dn;
      wload uhopper/hadoop-namenode nn;
      kubectl create -f configs/hdfs.yml;;
    spark) build_spark;
      wload spark:v2.3 spark
      ;;
    misc) build_misc;;
    shield) build_shield;
      wload shield shield;
      kubectl create -f configs/shield.yml;
      ;;
    image) load_image;;
    *) echo "unknown target $n"
  esac
done


