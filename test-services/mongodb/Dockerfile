FROM mongo:3.6
ENV MONGO_INITDB_ROOT_USERNAME=root
ENV MONGO_INITDB_ROOT_PASSWORD=passwd
COPY userAuth.sh /docker-entrypoint-initdb.d/userAuth.sh
