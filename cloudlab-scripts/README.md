
# Provision the cloudlab cluster for latte usage:

This script is used on your own workstation, to remotely provision a cloudlab cluster after you create it through cloudlab dashboard.

1. copy the list of machines to file "config". You can directly copy the dashboard "list" tab into this file

Example:
compute1        c220g2-011303   c220g2  ready   n/a     emulab-ops/UBUNTU18-64-STD      ssh -p 22 yanzhai@c220g2-011303.wisc.cloudlab.us
controller      c220g2-011306   c220g2  ready   n/a     emulab-ops/UBUNTU18-64-STD      ssh -p 22 yanzhai@c220g2-011306.wisc.cloudlab.us
network c220g1-030801   c220g1  ready   n/a     emulab-ops/UBUNTU18-64-STD     ssh -p 22 yanzhai@c220g1-030801.wisc.cloudlab.us
compute2        c220g1-030803   c220g1  ready   n/a     emulab-ops/UBUNTU18-64-STD      ssh -p 22 yanzhai@c220g1-030803.wisc.cloudlab.us

2. Assuming you have keyless ssh already configured (no need to pass -i). Run all\_dev.sh, will take ~30 minutes to setup basic storage and necessary ssh access.


3. Can now log into the cluster, and work with the actual cloudlab-k8s things (install.sh, build-all.sh, etc)
