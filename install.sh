source ./env.sh
sudo apt-get -y update
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add
sudo apt-add-repository "deb http://apt.kubernetes.io/ kubernetes-xenial main"
bash allrun.sh "sudo apt-get update; curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add; sudo apt-add-repository 'deb http://apt.kubernetes.io/ kubernetes-xenial main'; sudo apt-get update; sudo apt-get install -y kubelet"
sudo apt-get install -y kubeadm

bash allrun.sh "sudo apt-get install -y lldpd"

sudo apt-get install -y jq


vlan_info=`sudo bash check_link.sh  | head -n 1`
vlan_id=`echo $vlan_info | awk '{print $1}'`

bash allcp.sh check_link.sh /tmp/
bash allrun.sh "sudo bash check_link.sh | grep $vlan_id | awk '{print \$2}' > vlan_dev"

start=1
for n in $ALL_NODES; do
	ssh $n 'sudo ip addr add dev `cat vlan_dev` 192.168.0.'$start'/16; sudo swapoff /dev/sda3'
	start=$((start+1))
done

bash allcp.sh daemon.json
bash allrun.sh "sudo mkdir -p /etc/docker/; sudo cp daemon.json /etc/docker/; sudo systemctl restart docker"

# Cluster admin credential
sudo kubeadm init --apiserver-advertise-address 10.10.1.1 --pod-network-cidr=192.168.0.0/16 
mkdir -p $HOME/.kube
sudo cp -f -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config

# For safety
sudo iptables -A INPUT -d 10.10.1.1 --dport 6443 -j ACCEPT
sudo iptables -A INPUT -d 10.10.2.1 --dport 6443 -j ACCEPT
sudo iptables -A INPUT --dport 6443 -j DROP

generate_token=`sudo kubeadm token create`
ca_hash=`openssl x509 -pubkey -in /etc/kubernetes/pki/ca.crt | openssl rsa -pubin -outform der 2>/dev/null | openssl dgst -sha256 -hex | sed 's/^.* //'`

for n in $W_NODES; do
	ssh $n "sudo kubeadm join 10.10.2.1:6443 --token $generate_token --discovery-token-ca-cert-hash sha256:$ca_hash"
done
