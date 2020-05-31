
#!/bin/bash
# generate test purpose certificates for pipefitter communication

USER_TO_CREATE=${1:-example}

KUBECTL=kubectl

# find CA key. For lazy reason, we directly use docker-machine. In fact, keys should be distributed in
# a more robust way...

gen_key() {
  openssl genrsa -out $1.key 2048
#openssl ecparam -name prime256v1 -out $1_param.pem 
#openssl ecparam -name prime256v1 -in $1_param.pem -genkey -noout -out $1_ec.key
## convert the EC key to PKCS key
#openssl pkcs8 -topk8 -nocrypt -in $1_ec.key -outform PEM -out $1.key
#rm -f $1_param.pem $1_ec.key tmp-$1.key
}

# TODO: configs may be merged into a common CA config file, so the shell cmd could look elegant.
gen_cert() {
	#/O defines group, /CN defines username, K8s convention
openssl req -new -key $1.key -out $1.csr -subj "/O=users/CN=$1" #-config $SCRIPT_DIR/openssl.cnf 
echo openssl x509 -req -days 1000 -in $1.csr -CA $2.crt -CAkey $2.key -set_serial 0101 -out $1.crt -sha256 -extensions 'v3_req' #-extfile $SCRIPT_DIR/openssl.cnf 
sudo openssl x509 -req -days 1000 -in $1.csr -CA $2.crt -CAkey $2.key -set_serial 0101 -out $1.crt -sha256 -extensions 'v3_req' #-extfile $SCRIPT_DIR/openssl.cnf 
sudo chown $USER:$GROUP $1.crt
}

wrap_config() {
  local master_ip=10.10.2.1
  sed 's/USERNAME/'$1'/' templates/user_config.yaml > config-$1
  $KUBECTL config --kubeconfig=config-$1 set-cluster kubernetes --server=https://$master_ip:6443 --certificate-authority=/etc/kubernetes/pki/ca.crt --embed-certs=true
  $KUBECTL config --kubeconfig=config-$1 set-credentials $1 --client-certificate=$1.crt --client-key=$1.key --embed-certs=true
  $KUBECTL config --kubeconfig=config-$1 set-context latte --cluster=kubernetes --namespace=latte-$1 --user=$1
  $KUBECTL config --kubeconfig=config-$1 use-context latte
  $KUBECTL config --kubeconfig=config-$1 view
}

gen_role_and_privilege() {
  sed 's/USERNAME/'$1'/' templates/user_role.yaml > role-$1.yml
  $KUBECTL create namespace latte-$1
  $KUBECTL delete -f role-$1.yml
  $KUBECTL create -f role-$1.yml
}

gen_key $USER_TO_CREATE
gen_cert $USER_TO_CREATE /etc/kubernetes/pki/ca
wrap_config $USER_TO_CREATE
gen_role_and_privilege $USER_TO_CREATE

sudo chown $USER:$GROUP config-$USER_TO_CREATE

echo "##############################################################"
echo "   User $USER_TO_CREATE created"
echo " run following command to use it: "
echo "   mkdir -p ~/.kube/"
echo "   cp -i config-$USER_TO_CREATE ~/.kube/config"
echo " Then you can use kubectl to operate on behave of this user."
echo "##############################################################"

mkdir -p /openstack/home/
USERHOME=/openstack/home/$USER_TO_CREATE
sudo useradd -m -d $USERHOME -s /bin/bash $USER_TO_CREATE
sudo mkdir -p $USERHOME/.kube
sudo cp -f config-$USER_TO_CREATE $USERHOME/.kube/config
sudo mv $USER_TO_CREATE.* $USERHOME/
sudo cp configs/hello.yml $USERHOME/
rm role-$USER_TO_CREATE.yml config-$USER_TO_CREATE
sudo chown -R $USER_TO_CREATE $USERHOME/


