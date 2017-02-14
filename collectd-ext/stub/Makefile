libcollectd.so: stub.c collectd.h configfile.h plugin.h
	gcc -shared -o $@ -fPIC $<

install: libcollectd.so
	sudo mkdir -p /usr/local/lib/collectd
	sudo mkdir -p /usr/local/include/collectd
	sudo cp $< /usr/local/lib/collectd
	sudo cp *.h /usr/local/include/collectd