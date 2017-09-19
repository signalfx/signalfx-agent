libcollectd.so: stub.c collectd.h configfile.h plugin.h
	gcc -shared -o $@ -fPIC $<

install: libcollectd.so
	sudo mkdir -p /usr/local/lib
	sudo mkdir -p /usr/src/collectd/src/daemon
	sudo cp $< /usr/local/lib
	sudo cp *.h /usr/src/collectd/src/daemon
