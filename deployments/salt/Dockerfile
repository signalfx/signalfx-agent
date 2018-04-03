#
# Salt Stack Salt Dev Container
#

FROM ubuntu:16.04

# Update System
RUN apt-get update && apt-get upgrade -y -o DPkg::Options::=--force-confold


# Dependencies

RUN apt-get install -y software-properties-common

RUN apt-get install -y vim

RUN apt-get install -y apt-transport-https ca-certificates


# Install Salt-Master and Salt-Minion

RUN apt-get install -y salt-master salt-minion

RUN sed -i "s|#master: salt|master: localhost|g" /etc/salt/minion

RUN sed -i "s|#auto_accept: False|auto_accept: True|g" /etc/salt/master

RUN sed -i "s|#open_mode: False|open_mode: True|g" /etc/salt/master

COPY ./signalfx-agent/ /srv/salt/signalfx-agent/

COPY ./pillar.example /srv/pillar/signalfx-agent.sls


# Add Entrypoint File

ADD entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

ENTRYPOINT [ "/usr/local/bin/entrypoint.sh"]