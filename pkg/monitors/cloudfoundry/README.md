# cloudfoundry-firehose-nozzle

## Developer Resources

- https://github.com/cf-platform-eng/firehose-nozzle-v2/tree/master/gateway
- https://github.com/cloudfoundry/loggregator/blob/master/docs/rlp_gateway.md
- https://github.com/cloudfoundry/go-loggregator/blob/master/rlp_gateway_client.go

## Tanzu Application Service Setup

### Create a new TAS environment

1. Get access to [Pivotal Partners Slack](https://pivotalpartners.slack.com/archives/C42PWTRR9)
1. Create a new TAS environment via: https://self-service.isv.ci/


### Configure TAS for monitoring

Required tools:

- https://github.com/pivotal/hammer
- https://github.com/cloudfoundry/bosh-cli
- https://github.com/cloudfoundry/cli (most probably v6)
- https://github.com/cloudfoundry-community/firehose-plugin
- https://github.com/cloudfoundry/cf-uaac

> :warning: The example assumes that the environement is named `wildblueyonder`.

1. Download the hammer config from https://self-service.isv.ci

2. Login in CF CLI

    ```sh
    hammer -t wildblueyonder.json cf-login
    ```

3. Create a new space and configure it as a default target space

    ```sh
    cf create-space test-space && cf target -s test-space
    ```

4. Deploy a sample application:

    ```sh
    git clone https://github.com/cloudfoundry-samples/test-app && cd test-app && cf push && cd .. && rm -rf test-app
    ```

5. Test if the sample application is working:

    ```sh
    cf apps
    ```

6. Get the UAA credentials:
    
    1. Login to the `Tanzu Ops Manager` UI (URL and credentials in from https://self-service.isv.ci)
    2. Navigate: `Small Footprint Pivotal Application Service` > `Credentials` > `UAA` > `Identity Client Credentials`
    3. Export the credentials:
        
        ```sh
        export UAA_CREDS=<creds>
        ```

6. Login in UAA CLI:

    ```sh
    uaac target https://uaa.sys.wildblueyonder.cf-app.com
    uaac token client get identity -s $UAA_CREDS
    ```

7. Create a UAA user with the proper permissions to access the RLP Gateway (replace `<signalfx-nozzle client secret>` with something else):

    ```sh
    export NOZZLE_SECRET=<signalfx-nozzle client secret>
    uaac client add my-v2-nozzle --name signalfx-nozzle --secret $NOZZLE_SECRET --authorized_grant_types client_credentials,refresh_token --authorities logs.admin
    ```

### Configure SignalFx Smart Agent monitor

Example config:

```yaml
---
signalFxAccessToken: <signalfx token>
intervalSeconds: 10
logging:
  level: debug
monitors:
  - type: cloudfoundry-firehose-nozzle
    rlpGatewayUrl: https://log-stream.sys.wildblueyonder.cf-app.com
    rlpGatewaySkipVerify: true
    uaaUser: my-v2-nozzle
    uaaPassword: <signalfx-nozzle client secret>
    uaaUrl: https://uaa.sys.wildblueyonder.cf-app.com
    uaaSkipVerify: true
# Required: What format to send data in
writer:
  traceExportFormat: sapm
```

More: [docs/monitors/cloudfoundry-firehose-nozzle.md](../../../docs/monitors/cloudfoundry-firehose-nozzle.md)
