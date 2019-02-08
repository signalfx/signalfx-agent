ARG KONG_VERSION=1.0.0-centos
FROM kong:${KONG_VERSION}
RUN yum install -y epel-release
RUN yum install -y git sudo unzip
WORKDIR /usr/local/share/lua/5.1/kong
RUN sed -i '38ilua_shared_dict kong_signalfx_aggregation 10m;' templates/nginx_kong.lua
RUN sed -i '38ilua_shared_dict kong_signalfx_locks 100k;' templates/nginx_kong.lua
RUN sed -i '29i\ \ "signalfx",' constants.lua
RUN luarocks install kong-plugin-signalfx
RUN echo 'custom_plugins = signalfx' > /etc/kong/signalfx.conf
WORKDIR /
RUN mkdir -p /usr/local/kong/logs
RUN ln -s /dev/stderr /usr/local/kong/logs/error.log
RUN ln -s /dev/stdout /usr/local/kong/logs/access.log

# workaround for https://github.com/Kong/docker-kong/issues/216
RUN sed -i 's|su-exec|sudo -u|' /docker-entrypoint.sh
