# Check if the signalFxAccessToken is present in configuration.

{% if not salt['pillar.get']('signalfx-agent:conf:signalFxAccessToken') %}
{{ "SignalFxAccessToken is absent in conf" }}

{% else %}

{% set conf_file_path = salt['pillar.get']('signalfx-agent:conf_file_path', '/etc/signalfx/agent.yaml') %}

{% set service_user = salt['pillar.get']('signalfx-agent:service_user', 'signalfx-agent') %}

{% set service_group = salt['pillar.get']('signalfx-agent:service_group', 'signalfx-agent') %}

# Changing the agent.yaml configuration.

{{ conf_file_path }}:
  file.managed:
    - user: {{ service_user }}
    - group: {{ service_group }}
    - mode: '0600'
    - makedirs: True
    - template: jinja
    - source: salt://signalfx-agent/agent.yaml
    - watch:
      - user: service_user
      - group: service_group

{% endif %}
