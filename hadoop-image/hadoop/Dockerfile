FROM ubuntu:xenial


#RUN apt-get update && apt-get install -y software-properties-common && add-apt-repository -y ppa:openjdk-r/ppa

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends  openjdk-8-jre-headless ca-certificates-java \
    && rm -rf /var/lib/apt/lists/*
    
ENV JAVA_HOME=/usr/lib/jvm/java-8-openjdk-amd64/

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends net-tools curl

ENV HADOOP_VERSION 2.8.0
COPY hadoop-2.8.0.tar.gz /tmp/hadoop.tar.gz
#ENV HADOOP_URL https://www.apache.org/dist/hadoop/common/hadoop-$HADOOP_VERSION/hadoop-$HADOOP_VERSION.tar.gz
RUN tar -xvf /tmp/hadoop.tar.gz -C /opt/ \
    && rm /tmp/hadoop.tar.gz* \
    && ln -s /opt/hadoop-$HADOOP_VERSION/etc/hadoop /etc/hadoop \
    && cp /etc/hadoop/mapred-site.xml.template /etc/hadoop/mapred-site.xml \
    && mkdir /opt/hadoop-$HADOOP_VERSION/logs \
    && mkdir /hadoop-data \
    && rm -Rf /opt/hadoop-$HADOOP_VERSION/share/doc/hadoop

ENV HADOOP_PREFIX=/opt/hadoop-$HADOOP_VERSION
ENV HADOOP_CONF_DIR=/etc/hadoop
ENV MULTIHOMED_NETWORK=1

ENV USER=root
ENV PATH $HADOOP_PREFIX/bin/:$PATH

ADD entrypoint.sh /entrypoint.sh
RUN chmod a+x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
