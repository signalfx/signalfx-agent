# SignalFx agent Splunk Docker-compose example

This example showcases how the agent works with Splunk Enterprise.

The example runs as a Docker Compose deployment. The agent can be configured to send various metrics to Splunk Enterprise.

Splunk is configured to receive data from the SignalFx Agent using the HTTP Event collector. To learn more about HEC, visit [our guide](https://dev.splunk.com/enterprise/docs/dataapps/httpeventcollector/).

To deploy the example, open a terminal and in this directory type:
```bash
$> docker-compose up
```
:
Splunk will become available on port 18000. You can login on [http://localhost:18000] with `admin` and `changeme`.

Once logged in, visit the [analytics workspace](http://localhost:18000/en-US/app/search/analytics_workspace) to see which metrics are sent by the SignalFx Agent.

