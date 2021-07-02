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
- https://github.com/pivotal-cf/om

1. Download the hammer config from https://self-service.isv.ci and name it like your environement and export a variable

    ```sh
    export TAS_ENV=<TAS environement name>
    ```

2. Create a new space and configure it as a default target space

    ```sh
    hammer -t $TAS_ENV.json cf-login
    cf create-space test-space && cf target -s test-space
    ```

3. Deploy a sample application:

    ```sh
    hammer -t $TAS_ENV.json cf-login
    git clone https://github.com/cloudfoundry-samples/test-app && cd test-app && cf push && cd .. && rm -rf test-app && cf apps
    ```

4. Create a UAA user with the proper permissions to access the RLP Gateway:

    ```sh
    eval "$(hammer -t $TAS_ENV.json om)"
    export UAA_CREDS=$(om credentials -p cf -c .uaa.identity_client_credentials -t json | jq '.password' -r)
    uaac target https://uaa.sys.$TAS_ENV.cf-app.com
    uaac token client get identity -s $UAA_CREDS
    export NOZZLE_SECRET=$(openssl rand -base64 12)
    uaac client add my-v2-nozzle --name signalfx-nozzle --secret $NOZZLE_SECRET --authorized_grant_types client_credentials,refresh_token --authorities logs.admin
    echo "signalfx-nozzle client secret: $NOZZLE_SECRET"
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
    rlpGatewayUrl: https://log-stream.sys.<TAS environement name>.cf-app.com
    rlpGatewaySkipVerify: true
    uaaUser: my-v2-nozzle
    uaaPassword: <signalfx-nozzle client secret>
    uaaUrl: https://uaa.sys.<TAS environement name>.cf-app.com
    uaaSkipVerify: true
# Required: What format to send data in
writer:
  traceExportFormat: sapm
```

More: [docs/monitors/cloudfoundry-firehose-nozzle.md](../../../docs/monitors/cloudfoundry-firehose-nozzle.md)
