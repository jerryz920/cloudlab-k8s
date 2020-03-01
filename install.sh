source ./env.sh
sudo apt-get -y update
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add
sudo apt-add-repository "deb http://apt.kubernetes.io/ kubernetes-xenial main"
bash allrun.sh "sudo apt-get update; curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add; sudo apt-add-repository 'deb http://apt.kubernetes.io/ kubernetes-xenial main'; sudo apt-get update; sudo apt-get install -y kubelet kubeadm"

bash allrun.sh "sudo apt-get install -y lldpd jq"


net_info=`sudo bash helpers/check_link.sh  | head -n 1`
net_id=`echo $net_info | awk '{print $1}'`

bash allcp.sh helpers/check_link.sh /tmp/
bash allrun.sh "sudo bash /tmp/check_link.sh | grep $net_id | awk '{print \$2}' > vlan_dev"

start=1
for n in $ALL_NODES; do
	ssh $n 'sudo ip addr add dev `cat vlan_dev` 192.168.0.'$start'/16; sudo swapoff /dev/sda3'
	start=$((start+1))
done

bash allcp.sh configs/daemon.json
bash allrun.sh "sudo mkdir -p /etc/docker/; sudo cp daemon.json /etc/docker/; sudo systemctl restart docker; sudo gpasswd -a $USER docker;"

# Cluster admin credential
sudo kubeadm init --apiserver-advertise-address 10.10.1.1 --pod-network-cidr=192.168.0.0/16 
mkdir -p $HOME/.kube
sudo cp -f /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config

# For safety
sudo iptables -N security
sudo iptables -I INPUT -j security
sudo iptables -A security -d 10.10.1.1 -p tcp --dport 6443 -j ACCEPT
sudo iptables -A security -d 10.10.2.1 -p tcp --dport 6443 -j ACCEPT
sudo iptables -A security -p tcp --dport 6443 -j DROP

generate_token=`sudo kubeadm token create`
ca_hash=`openssl x509 -pubkey -in /etc/kubernetes/pki/ca.crt | openssl rsa -pubin -outform der 2>/dev/null | openssl dgst -sha256 -hex | sed 's/^.* //'`

for n in $W_NODES; do
	ssh $n "sudo kubeadm join 10.10.1.1:6443 --token $generate_token --discovery-token-ca-cert-hash sha256:$ca_hash"
done

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
sudo chown $USER:`id -g -n` go
sudo ln -s /openstack/goroot/go/ ~/goroot
sudo ln -s /openstack/go/ ~/go

echo "export GOROOT=~/goroot" >> ~/.bashrc
echo "export GOPATH=~/go" >> ~/.bashrc
echo "export PATH=\$PATH:/.local/bin:\$PATH:\$GOROOT/bin/:\$GOPATH/bin/" >> ~/.bashrc

export GOROOT=~/goroot
export GOPATH=~/go
export PATH=$PATH:$GOROOT/bin/:$GOPATH/bin/




