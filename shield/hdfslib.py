
import hdfs
from flask import request


sealed_prefix="/sealed"



def do_hdfs_upload(myid, url, safe_url, keyhash, fname, fdata, ftag):
    # enforce teh file name to be the public hash prefixed or sealed prefix

    if not fname.startsWith("/" + keyhash) or not fname.startsWith(sealed_prefix):
        fname = "/%s/%s" % (keyhash, fname)

    # create hdfs client on demand now. No need to optimize for PoC
    hdfs_client = hdfs.InsecureClient(url, user=ftag)
    return hdfs_client.write(fname, fdata, overwrite=True, permission=666)

