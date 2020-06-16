# Generate keys first
mkdir -p keys
if ! [ -f keys/id_rsa ]; then
  ssh-keygen -b 2048 -t rsa -f keys/id_rsa -q -N ""
fi
if ! [ -f keys/id_ed25519 ]; then
  ssh-keygen -b 256 -t ed25519 -f keys/id_25519 -q -N ""
fi
if ! [ -f keys/id_ecdsa ]; then
  ssh-keygen -b 256 -t ecdsa -f keys/id_ecdsa -q -N ""
fi

tar czf cloudlab-install.tgz *.sh *.py etc keys
#cp cloudlab-install.tgz ~/public/html/
