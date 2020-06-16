source ./env.sh
sudo apt-get -y update
sudo apt-get install -y dos2unix
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add
sudo apt-add-repository "deb http://apt.kubernetes.io/ kubernetes-xenial main"
bash allrun.sh "sudo apt-get update; curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add; sudo apt-add-repository 'deb http://apt.kubernetes.io/ kubernetes-xenial main'; sudo apt-get update; sudo apt-get install -y kubelet kubeadm"

echo "deb https://dl.bintray.com/sbt/debian /" | sudo tee -a /etc/apt/sources.list.d/sbt.list
curl -sL "https://keyserver.ubuntu.com/pks/lookup?op=get&search=0x2EE0EA64E40A89B84B2DF73499E82A75642AC823" | sudo apt-key add
sudo apt-get update
sudo apt-get install sbt

bash allrun.sh "sudo apt-get install -y lldpd jq"


net_info=`sudo bash helpers/check_link.sh  | head -n 1`
net_id=`echo $net_info | awk '{print $1}'`

bash allcp.sh helpers/check_link.sh /tmp/
bash allrun.sh "sudo bash /tmp/check_link.sh | grep $net_id | awk '{print \$2}' > vlan_dev"

start=1
for n in $ALL_NODES; do
        swp=`sudo fdisk -l | grep swap | awk '{print $1}'`
	ssh $n 'sudo ip addr add dev `cat vlan_dev` 192.168.0.'$start'/16; sudo swapoff '$swp
	start=$((start+1))
done

bash allcp.sh configs/daemon.json
bash allrun.sh "sudo mkdir -p /etc/docker/; sudo cp daemon.json /etc/docker/; sudo systemctl restart docker; sudo gpasswd -a $USER docker;"

# Cluster admin credential
sudo kubeadm init --apiserver-advertise-address 10.10.1.1 --pod-network-cidr=192.168.128.0/16 
mkdir -p $HOME/.kube
sudo cp -f /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config

# For safety
sudo iptables -N security
sudo iptables -I INPUT -j security
sudo iptables -A security -d 10.10.1.1 -p tcp --dport 6443 -j ACCEPT
sudo iptables -A security -p tcp --dport 6443 -j DROP
#
generate_token=`sudo kubeadm token create`
ca_hash=`openssl x509 -pubkey -in /etc/kubernetes/pki/ca.crt | openssl rsa -pubin -outform der 2>/dev/null | openssl dgst -sha256 -hex | sed 's/^.* //'`
bash wrun.sh "sudo kubeadm join 10.10.1.1:6443 --token $generate_token --discovery-token-ca-cert-hash sha256:$ca_hash"

sleep 2
echo "applying calico network"
kubectl apply -f configs/calico.yaml

pushd .

# Installing Golang for building things later
cd /openstack/
sudo wget https://dl.google.com/go/go1.13.8.linux-amd64.tar.gz -O gobinary.tar.gz
sudo apt-get install -y cmake

sudo rm -rf goroot
sudo mkdir -p goroot
sudo tar xf gobinary.tar.gz -Cgoroot
sudo mkdir -p go
sudo chown -R $USER:`id -g -n` go
sudo ln -s /openstack/goroot/go/ ~/goroot
sudo ln -s /openstack/go/ ~/go

echo "export GOROOT=~/goroot" >> ~/.bashrc
echo "export GOPATH=~/go" >> ~/.bashrc
echo "export PATH=\$PATH:/.local/bin:\$PATH:\$GOROOT/bin/:\$GOPATH/bin/" >> ~/.bashrc

export GOROOT=~/goroot
export GOPATH=~/go
export PATH=$PATH:$GOROOT/bin/:$GOPATH/bin/


popd
# label nodes to prepare hdfs running
index=0
for node in `bash wrun.sh hostname | dos2unix`; do
  echo "Labeling node $node"
  kubectl label node $node nodetype=worker-$index --overwrite
  index=$((index+1))
done
kubectl label node `hostname | dos2unix`  nodetype=master --overwrite
kubectl create namespace kube-exts

# installing a systemd service for http file upload
# intentionally to be read only by root. Drop jar files to /openstack/files
sudo mkdir /openstack/files/
sudo cp configs/http-file.service /etc/systemd/system
sudo systemctl daemon-reload
sudo systemctl start http-file

