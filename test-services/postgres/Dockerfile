ARG POSTGRES_VERSION=11-alpine
FROM postgres:${POSTGRES_VERSION}

CMD ["postgres", "-c", "shared_preload_libraries=pg_stat_statements"]

RUN apk add --no-cache unzip wget bash
WORKDIR /opt
# dvdrental.zip downloaded from https://www.postgresqltutorial.com/wp-content/uploads/2019/05/dvdrental.zip
COPY dvdrental.zip ./dvdrental.zip
RUN unzip dvdrental.zip
RUN tar -xf dvdrental.tar
RUN sed -i -e 's/\$\$PATH\$\$/\/opt/' ./restore.sql
RUN chmod 777 /opt/*

COPY init.sh /docker-entrypoint-initdb.d/
