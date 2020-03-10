
# Some necessary TLS setup
from flask import Flask
from flask import request as frequest
import werkzeug.serving
import ssl
import OpenSSL

class PeerCertWSGIRequestHandler( werkzeug.serving.WSGIRequestHandler ):
    """
    We subclass this class so that we can gain access to the connection
    property. self.connection is the underlying client socket. When a TLS
    connection is established, the underlying socket is an instance of
    SSLSocket, which in turn exposes the getpeercert() method.

    The output from that method is what we want to make available elsewhere
    in the application.
    """
    def make_environ(self):
        """
        The superclass method develops the environ hash that eventually
        forms part of the Flask request object.

        We allow the superclass method to run first, then we insert the
        peer certificate into the hash. That exposes it to us later in
        the request variable that Flask provides
        """
        environ = super(PeerCertWSGIRequestHandler, self).make_environ()
        x509_binary = self.connection.getpeercert(True)
        x509 = OpenSSL.crypto.load_certificate( OpenSSL.crypto.FILETYPE_ASN1, x509_binary )
        environ['peercert'] = x509
        return environ


def create(name):
    return Flask(name)

def run(app, host, port, ca_cert, cert, key):
    # create_default_context establishes a new SSLContext object that
    # aligns with the purpose we provide as an argument. Here we provide
    # Purpose.CLIENT_AUTH, so the SSLContext is set up to handle validation
    # of client certificates.
    ssl_context = ssl.create_default_context( purpose=ssl.Purpose.CLIENT_AUTH,
                                              cafile=ca_cert )
    # load in the certificate and private key for our server to provide to clients.
    # force the client to provide a certificate.
    ssl_context.load_cert_chain(certfile=cert, keyfile=key, password=None)
    ssl_context.verify_mode = ssl.CERT_REQUIRED
    app.run(host, port, ssl_context=ssl_context, request_handler=PeerCertWSGIRequestHandler)
