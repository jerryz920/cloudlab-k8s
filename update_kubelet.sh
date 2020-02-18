cd kubernetes
make -j kubelet kubeadm kubectl
bash allcp.sh _output/bin/kubelet
bash allrun.sh "sudo systemctl stop kubelet; sudo cp kubelet /usr/bin/kubelet; sudo systemctl start kubelet"

