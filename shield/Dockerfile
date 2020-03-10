from ubuntu:xenial

run apt-get update
run apt-get install -y python-dev libpython-dev python3-pip python3-dev libpython3-dev python-pip 
run apt-get install -y libssl-dev
run pip3 install Flask hdfs pyopenssl requests Werkzeug
copy . /opt/hdfs-shield
workdir /opt/hdfs-shield
entrypoint ["/usr/bin/python3"]
cmd ["main.py"]

run echo "make sure to use create-creds.sh to create K8s secret, and spec.yml to load the Pod"
