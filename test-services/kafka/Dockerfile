# NOTE: the image we are using is broken on version 1.1.0 due to bad parsing of
# JMX options.  Hopefully 1.1.1 will work correctly.
ARG KAFKA_VERSION=1.0.1
FROM wurstmeister/kafka:${KAFKA_VERSION}

ENV KAFKA_ADVERTISED_LISTENERS "INSIDE://:9092,OUTSIDE://_{HOSTNAME_COMMAND}:9094"
ENV KAFKA_LISTENERS "INSIDE://:9092,OUTSIDE://:9094"
ENV KAFKA_JMX_OPTS "-Dcom.sun.management.jmxremote -Dcom.sun.management.jmxremote.authenticate=false -Dcom.sun.management.jmxremote.ssl=false -Dcom.sun.management.jmxremote.port=1099 -Dcom.sun.management.jmxremote.rmi.port=1099"
ENV JMX_PORT "1099"
