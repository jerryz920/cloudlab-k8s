#!/bin/sh

#nodes=master
#for n in `seq 1 5`; do
#  nodes="$nodes worker$n"
#done
cp ../hadoop/hadoop-dist/target/hadoop-2.8.0.tar.gz hadoop/
if [ $? -ne 0 ]; then
  echo "build hadoop first."
  exit 1
fi


for i in hadoop namenode datanode; do
  echo Building $i
  ( cd $i && ./build.sh)
done
