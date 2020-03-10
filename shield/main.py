#!/usr/bin/env python3

from flask import request as frequest
from server import create, run
import sys
import utils
from hdfslib import do_hdfs_upload
from api import define_tag
import os

app=create(__name__)
myid=None
hdfs_url=os.environ.get("HDFS_URL", "http://hdfs-1.latte.org:50070")
safe_url=os.environ.get("MDS_URL", "http://mds:19851")
ca_cert = os.environ.get("CA_CERT", "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
cert = os.environ.get("TLS_CERT", "/opt/creds/server.crt")
key = os.environ.get("TLS_KEY", "/opt/creds/server.key")


@app.route('/upload', methods=['PUT'])
def upload_hdfs():
    return do_hdfs_upload(
            myid,
            hdfs_url,
            safe_url,
            utils.keyhash(frequest.environ["peercert"]),
            frequest.form["filename"],
            frequest.files["file"],
            frequest.form["tagname"])


if __name__ == "__main__":
    run(app, "0.0.0.0", 20000, ca_cert, cert, key)


