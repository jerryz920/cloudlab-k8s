
import hdfs
from flask import request
import os


sealed_prefix="/sealed"



def do_hdfs_upload(url, keyhash, fname, fdata, ftag):
    # enforce teh file name to be the public hash prefixed or sealed prefix
    if not fname.startswith("/" + keyhash) or not fname.startswith(sealed_prefix):
        fname = "/%s/%s" % (keyhash, fname)
    dirname = os.path.dirname(fname)
    prepare_user_dir(url, dirname, keyhash)

    # create hdfs client on demand now. No need to optimize for PoC
    hdfs_client = hdfs.InsecureClient(url, user=ftag)
    return hdfs_client.write(fname, fdata, overwrite=True, permission=666)

def prepare_dir(url, dirname, keyhash):
    hdfs_client = hdfs.InsecureClient(url, user='root')
    # We don't know if this exist or not.
    hdfs_client.makedirs(dirname, permission=0600)
    hdfs_client.set_owner(dirname, owner=keyhash)


