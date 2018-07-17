FROM kong:latest

WORKDIR /usr/local/share/lua/5.1/kong
RUN sed -i '38ilua_shared_dict kong_signalfx_aggregation 10m;' templates/nginx_kong.lua
RUN sed -i '38ilua_shared_dict kong_signalfx_locks 100k;' templates/nginx_kong.lua
RUN sed -i '29i\ \ "signalfx",' constants.lua
RUN luarocks install --server=http://luarocks.org/manifests/signalfx kong-plugin-signalfx
RUN echo 'custom_plugins = signalfx' > /etc/kong/signalfx.conf
