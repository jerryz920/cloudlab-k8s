cd kubernetes
KUBE_FASTBUILD=true build/run.sh make kube-apiserver
if [ $? -ne 0 ]; then
	echo "build fail!"
	exit 1
fi
KUBE_FASTBUILD=true make package


hack/print-workspace-status.sh | grep DOCKER_TAG | awk '{print $2}'
docker load -i _output/release-images/amd64/kube-apiserver.tar

# replace kubelet configure

