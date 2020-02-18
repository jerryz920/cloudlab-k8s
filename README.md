
Assume keyless ssh has been setup between nodes.


Simple note:


env.sh: defines the nodes involved for this k8s cluster

allcp.sh allrun.sh wcp.sh wrun.sh: cluster helper scripts

calico.yaml
daemon.json
check\_link.sh: internal usage only

build_apiserver.sh update_kubelet.sh: dev helper script

install.sh: run this and a cluster is setup

create_user.sh templates: used to provision new user credential (current user as admin)

kubernetes: # submodule, run git submodule init and git submodule update to fetch dev source
