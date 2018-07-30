FROM ubuntu:16.04

ENV KAFKA_VERSION=1.0.1
RUN apt-get update
RUN apt-get install -y wget
RUN apt-get install -y default-jre
RUN cd /opt/ && wget https://archive.apache.org/dist/kafka/"$KAFKA_VERSION"/kafka_2.11-"$KAFKA_VERSION".tgz && tar -zxf kafka_2.11-"$KAFKA_VERSION".tgz && cd kafka_2.11-"$KAFKA_VERSION"/
ADD scripts/* scripts/
CMD ["bash", "scripts/10-setup.sh"]
