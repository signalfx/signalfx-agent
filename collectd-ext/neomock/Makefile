CFLAGS := -g -O0 -I/usr/include/collectd -I/usr/include -I/usr/local/include/collectd -I/usr/local/include -DSIGNALFX_EIM=1
LDFLAGS := -L /usr/local/lib/collectd -lcollectd -lpthread -Wl,-rpath /usr/local/lib/collectd

neomock: neomock.c
	gcc $(CFLAGS) -o neomock neomock.c $(LDFLAGS)
