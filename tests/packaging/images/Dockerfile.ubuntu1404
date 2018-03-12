FROM ubuntu:14.04

# See https://github.com/tianon/dockerfiles

RUN apt update &&\
    apt install -y ca-certificates procps wget apt-transport-https

RUN rm /usr/sbin/policy-rc.d; \
	rm /sbin/initctl; dpkg-divert --rename --remove /sbin/initctl

RUN /usr/sbin/update-rc.d -f ondemand remove; \
	for f in \
		/etc/init/u*.conf \
		/etc/init/mounted-dev.conf \
		/etc/init/mounted-proc.conf \
		/etc/init/mounted-run.conf \
		/etc/init/mounted-tmp.conf \
		/etc/init/mounted-var.conf \
		/etc/init/hostname.conf \
		/etc/init/networking.conf \
		/etc/init/tty*.conf \
		/etc/init/plymouth*.conf \
		/etc/init/hwclock*.conf \
		/etc/init/module*.conf\
	; do \
		dpkg-divert --local --rename --add "$f"; \
	done; \
	echo '# /lib/init/fstab: cleared out for bare-bones Docker' > /lib/init/fstab

COPY socat /bin/socat

# Insert our fake certs to the system bundle so they are trusted
COPY certs/*.signalfx.com.* /
RUN cat /*.cert >> /etc/ssl/certs/ca-certificates.crt

COPY init-fake.conf /etc/init/fake-container-events.conf

CMD ["/sbin/init", "-v"]
