CFLAGS := -g -O0 -I/usr/src/collectd/src/daemon -I /usr/src/collectd/src/liboconfig -I /usr/src/collectd/src/ -I/usr/include -I/usr/local/include -DSIGNALFX_EIM=1
# libcollectd will ultimately live at /usr/local/lib/collectd in the final image
LDFLAGS := -L /usr/src/collectd -lcollectd -lpthread -Wl,-rpath /usr/local/lib/collectd

neomock: neomock.c
	gcc $(CFLAGS) -o neomock neomock.c $(LDFLAGS)
