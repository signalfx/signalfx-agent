:orphan:

.. _realm-smart-agent:

************
Realms
************

Your **realm** is self-contained deployment of SignalFx in which your organization is hosted.
Different realms have different API endpoints. Consider the endpoint for sending data to SignalFx:

* In the `us1` realm, the endpoint is `https://ingest.us1.signalfx.com/v2/datapoint`.
* In the `eu0` realm, the endpoint is `https://ingest.eu0.signalfx.com/v2/datapoint`.

In general, the format for the ingest API endpoint is `https://ingest.<REALM>.signalfx.com`. When you see a
reference to `<REALM>` in the documentation, replace it with the actual value for your realm.

To find the realm for your organization, navigate to your
[profile page](https://docs.signalfx.com/en/latest/getting-started/get-around-ui.html#your-profile)
in the SignalFx UI.

If you don't include your realm name when specifying an endpoint, SignalFx interprets it as pointing to the `us0` realm.

==========================
Realms and the Smart Agent
==========================

The Smart Agent uses an API endpoint to send data to SignalFx. To ensure that your monitoring data goes to
your organization, make sure you specify the correct realm when you configure the Smart Agent.