# Check if the signalFxAccessToken is present in configuration.

{% if not salt['pillar.get']('signalfx-agent:conf:signalFxAccessToken') %}
{{ "SignalFxAccessToken is absent in conf" }}

{% else %}

{% set conf_file_path = salt['pillar.get']('signalfx-agent:conf_file_path', '/etc/signalfx/agent.yaml') %}

# Changing the agent.yaml configuration.

{{ conf_file_path }}:
  file.managed:
    - user: signalfx-agent
    - group: signalfx-agent
    - makedirs: True
    - template: jinja
    - source: salt://signalfx-agent/agent.yaml

{% endif %}
