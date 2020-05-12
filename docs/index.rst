.. _smart-agent:

*********************************
Use the Smart Agent
*********************************


The SignalFx Smart Agent gathers host performance, application, and service-level metrics from both containerized and non-container environments: See `sample monitor metrics <https://docs.signalfx.com/en/latest/integrations/agent/monitors/collectd-couchbase.html#Metrics>`__. The Smart Agent installs with more than 100 bundled monitors for gathering data, including Python-based plug-ins such as Mongo, Redis, and Docker. See `included monitors <https://docs.signalfx.com/en/latest/integrations/agent/monitors/_monitor-config.html#monitor-list>`__.

You can easily create integrations to Amazon Web Services, Azure, Google Cloud Platform, and Kubernetes environments using Smart Agent, and then login to SignalFx to view the incoming metrics in an easy-to-understand graphical format.  Metrics are sent to SignalFx to display on `built-in dashboards and charts <https://docs.signalfx.com/en/latest/getting-started/built-in-content/built-in-dashboards.html>`__. Check out the health of your network and nodes using `Infrastructure Navigator <https://docs.signalfx.com/en/latest/getting-started/built-in-content/infra-nav.html>`__.

The Smart Agent also supports receiving and sending trace data: See :new-page-ref:`apm2-smart-agent`.

To quickly install the SignalFx Smart Agent see the `Quick Install <https://docs.signalfx.com/en/latest/integrations/agent/quick-install.html>`__. Then, for a guided tour around SignalFx capabilities, try the `15-Minute Quick Start <https://docs.signalfx.com/en/latest/getting-started/quick-start.html>`__.


.. toctree::
   :maxdepth: 1
   :hidden:

   /integrations/agent/quick-install
   /integrations/agent/agent-install-pguidance
   /integrations/agent/install-packages
   /integrations/agent/install-standalone-linux
   /integrations/agent/install-standalone-windows
   /integrations/agent/agent-k8s-install-helm
   /integrations/agent/agent-k8s-install-kubectl
   /integrations/agent/agent-install-awsecs
   /integrations/agent/agent-install-config-mgmt
   /integrations/agent/config-schema
   /integrations/agent/observers/index
   /integrations/agent/monitors/index
   /integrations/agent/auto-discovery
   /integrations/agent/filtering
   /integrations/agent/remote-config
   /integrations/agent/windows
   /integrations/agent/faq
   /integrations/agent/legacy-filtering
   /integrations/agent/deb-rpm-repo-migration
