{% set service_user = salt['pillar.get']('signalfx-agent:service_user', 'signalfx-agent') %}

{% set service_group = salt['pillar.get']('signalfx-agent:service_group', 'signalfx-agent') %}

service_group:
  group.present:
    - name: {{ service_group }}
    - system: True
    - unless: getent group {{ service_group }}

service_user:
  user.present:
    - name: {{ service_user }}
    - system: True
    - shell: /sbin/nologin
    - createhome: False
    - groups:
      - {{ service_group }}
    - unless: getent passwd {{ service_user }}
    - watch:
      - group: service_group

{% if salt['grains.get']('init') == 'systemd' %}

/etc/tmpfiles.d/signalfx-agent.conf:
  file.managed:
    - contents: |
        D /run/signalfx-agent 0755 {{ service_user }} {{ service_group }} - -
    - makedirs: True
    - watch:
      - user: service_user
      - group: service_group

/etc/systemd/system/signalfx-agent.service.d/service-owner.conf:
  file.managed:
    - contents: |
        [Service]
        User={{ service_user }}
        Group={{ service_group }}
    - makedirs: True
    - watch:
      - user: service_user
      - group: service_group

stop-service:
  service.dead:
    - name: signalfx-agent
    - onchanges:
      - file: /etc/tmpfiles.d/signalfx-agent.conf
      - file: /etc/systemd/system/signalfx-agent.service.d/service-owner.conf

init-tmpfile:
  cmd.run:
    - name: systemd-tmpfiles --create --remove /etc/tmpfiles.d/signalfx-agent.conf
    - onchanges:
      - file: /etc/tmpfiles.d/signalfx-agent.conf

reload-service:
  cmd.run:
    - name: systemctl daemon-reload
    - onchanges:
      - file: /etc/systemd/system/signalfx-agent.service.d/service-owner.conf

{% else %}

/etc/default/signalfx-agent:
  file.managed:
    - replace: False
    - makedirs: True

set-initd-user:
  file.replace:
    - name: /etc/default/signalfx-agent
    - pattern: ^user=.*
    - repl: user={{ service_user }}
    - append_if_not_found: True
    - watch:
      - user: service_user

set-initd-group:
  file.replace:
    - name: /etc/default/signalfx-agent
    - pattern: ^group=.*
    - repl: group={{ service_group }}
    - append_if_not_found: True
    - watch:
      - group: service_group

stop-service:
  service.dead:
    - name: signalfx-agent
    - onchanges:
      - file: /etc/default/signalfx-agent

{% endif %}
