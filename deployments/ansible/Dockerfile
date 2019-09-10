FROM ubuntu:16.04

RUN sed -i -e 's|^deb http://.*\.ubuntu.com/ubuntu/\?|deb mirror://mirrors.ubuntu.com/mirrors.txt|' /etc/apt/sources.list
RUN echo "===> Adding Ansible's PPA..."  && \
    echo "deb http://ppa.launchpad.net/ansible/ansible/ubuntu xenial main" | tee /etc/apt/sources.list.d/ansible.list           && \
    echo "deb-src http://ppa.launchpad.net/ansible/ansible/ubuntu xenial main" | tee -a /etc/apt/sources.list.d/ansible.list    && \
    apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys 7BB9C367    && \
    DEBIAN_FRONTEND=noninteractive  apt-get update  && \
    \
    \
    echo "===> Installing Ansible..."  && \
    apt-get install -y apt-transport-https ca-certificates && \
    apt-get install -y ansible  && \
    \
    \
    echo "===> Installing handy tools (not absolutely required)..."  && \
    apt-get install -y python-pip              && \
    pip install --upgrade pywinrm              && \
    pip install ansible-lint                   && \
    apt-get install -y sshpass openssh-client  && \
    \
    \
    echo "===> Removing Ansible PPA..."  && \
    rm -rf /var/lib/apt/lists/*  /etc/apt/sources.list.d/ansible.list  && \
    \
    \
    echo "===> Adding hosts for local development and test..."  && \
    echo 'localhost' > /etc/ansible/hosts

# default command: display Ansible version
# CMD [ "ansible-playbook", "--version" ]

WORKDIR /opt/ansible
COPY . /opt/ansible
RUN echo "[signalfx-host-group]" > inventory && \
    echo localhost >> inventory

