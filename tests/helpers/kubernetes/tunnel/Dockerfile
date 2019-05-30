FROM alpine:3.9

RUN apk add --no-cache curl openssh &&\
    ssh-keygen -A

RUN echo 'root:root' | chpasswd &&\
    sed -i -e 's/GatewayPorts no/GatewayPorts yes/' /etc/ssh/sshd_config &&\
	sed -i -e 's/AllowTcpForwarding no/AllowTcpForwarding yes/' /etc/ssh/sshd_config

COPY id_rsa.pub /root/.ssh/authorized_keys

CMD ["/usr/sbin/sshd", "-D", "-e"]
