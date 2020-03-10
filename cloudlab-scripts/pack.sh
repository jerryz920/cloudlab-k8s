# Generate keys first
ssh-keygen -b 2048 -t rsa -f keys/id_rsa -q -N ""
tar czf cloudlab-install.tgz *.sh *.py etc keys
#cp cloudlab-install.tgz ~/public/html/
