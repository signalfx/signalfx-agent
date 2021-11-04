FROM ubuntu:20.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update  && \
    apt-get install -y apt-transport-https ca-certificates python3 python3-pip sshpass openssh-client

RUN pip3 install --upgrade ansible==3.0.0 ansible-lint==5.0.0

RUN mkdir -p /etc/ansible && \
    echo 'localhost' > /etc/ansible/hosts

# default command: display Ansible version
# CMD [ "ansible-playbook", "--version" ]

WORKDIR /opt/ansible
COPY . /opt/ansible
RUN echo "[signalfx-host-group]" > inventory && \
    echo localhost >> inventory
