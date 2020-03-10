
from OpenSSL import crypto
from hashlib import sha256

def keyhash(cert):
    raw = crypto.dump_publickey(crypto.FILETYPE_ASN1, cert.get_pubkey())
    hasher = sha256(raw)
    return hasher.hexdigest()

def read_keyhash(cert_file):
    with open(cert_file, "r") as fcert:
        data = fcert.read()
        return keyhash(crypto.load_certificate(crypto.FILETYPE_PEM, data))

