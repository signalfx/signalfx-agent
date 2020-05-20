.. _smart-agent:

*********************************
Use the Smart Agent
*********************************

.. toctree::
   :hidden:

   /integrations/agent/agent-install-methods
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

The SignalFx Smart Agent gathers host performance, application, and service-level
:strong:metrics from both containerized and non-container environments. To learn more about
these metrics, see
`sample monitor metrics <https://docs.signalfx.com/en/latest/integrations/agent/monitors/collectd-couchbase.html#Metrics>`__.

The Smart Agent gathers metrics using :strong:monitors, and comes with more than 100 bundled monitors
including Python-based plug-ins such as Mongo, Redis, and Docker. To learn more about
these monitors, see
`included monitors <https://docs.signalfx.com/en/latest/integrations/agent/monitors/_monitor-config.html#monitor-list>`__.

After you install the Smart Agent, use it to integrate with cloud services, including Amazon Web Services,
Microsoft Azure, Google Cloud Platform, and Kubernetes environments. Then,
login to SignalFx to view the incoming metrics in
`built-in dashboards and charts <https://docs.signalfx.com/en/latest/getting-started/built-in-content/built-in-dashboards.html>`__.

Check out the health of your network and nodes using
`Infrastructure Navigator <https://docs.signalfx.com/en/latest/getting-started/built-in-content/infra-nav.html>`__.

The Smart Agent also supports receiving and sending trace data: See :new-page-ref:`apm2-smart-agent`.

To quickly install the SignalFx Smart Agent, see the topic
`Quick Install <https://docs.signalfx.com/en/latest/integrations/agent/quick-install.html>`__.

To learn about all the available installation methods, see topic
`Choose a Smart Agent Install Method <agent-choose-install-type.rst>`__.

For a guided tour around SignalFx capabilities, try the
`15-Minute Quick Start <https://docs.signalfx.com/en/latest/getting-started/quick-start.html>`__.
