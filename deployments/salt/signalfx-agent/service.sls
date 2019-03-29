{% set conf_file_path = salt['pillar.get']('signalfx-agent:conf_file_path', '/etc/signalfx/agent.yaml') %}

signalfx-agent-service:
  service.running:
    - name: signalfx-agent
    - enable: True
    - require:
      - pkg: signalfx-agent
    - watch:
      - pkg: signalfx-agent
      - file: {{ conf_file_path }}
