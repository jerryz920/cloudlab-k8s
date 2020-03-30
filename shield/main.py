#!/usr/bin/env python3

from flask import request as frequest
from server import create, run
import sys
import utils
from hdfslib import do_hdfs_upload
import os

app=create(__name__)
hdfs_url=os.environ.get("HDFS_URL", "http://namenode:50070")
ca_cert = os.environ.get("CA_CERT", "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
cert = os.environ.get("TLS_CERT", "/opt/creds/server.crt")
key = os.environ.get("TLS_KEY", "/opt/creds/server.key")


@app.route('/upload', methods=['PUT'])
def upload_hdfs():
    print("filename = ", frequest.form['filename'])
    print("tagname = ", frequest.form['tagname'])
    return do_hdfs_upload(
            hdfs_url,
            utils.keyhash(frequest.environ["peercert"]),
            frequest.form["filename"],
            frequest.files["file"],
            frequest.form["tagname"])


if __name__ == "__main__":
    run(app, "0.0.0.0", 20000, ca_cert, cert, key)


