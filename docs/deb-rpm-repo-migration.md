# DEB/RPM Repo Migration

As part of our integration with Splunk, we are switching over our APT and RPM
packages of the Smart Agent to a new repository (`splunk.jfrog.io`) and signing
them with Splunk's key. The 5.0.0 release of the agent will be the first
version of the Smart Agent that will exist only in the new repositories.

All pre-5.0.0 versions of the Smart Agent will be mirrored over to the new
`splunk.jfrog.io` repository and will be available from our original
`dl.signalfx.com` repositories.

Our old APT/Debian and RPM repositories at `dl.signalfx.com` will not be
supported at all after April 21, 2020 and may cease to work at any point after
that.

## Migration Process

### APT

1. Delete the existing SignalFx apt key.

```sh
$ sudo apt-key del 5AE495F6
```

2. Import the new key:

```sh
$ curl https://splunk.jfrog.io/splunk/signalfx-agent-deb/splunk-B3CD4420.gpg | sudo apt-key add -
```

3. Replace the contents of /etc/apt/sources.list.d/signalfx-agent.list with:

```sh
deb https://splunk.jfrog.io/splunk/signalfx-agent-deb release main
```

4. Update the package metadata.

```sh
$ sudo apt-get update
```

5. Install the latest signalfx-agent.

```sh
$ sudo apt-get upgrade signalfx-agent
```

### RPM

1. Delete the existing SignalFx RPM key.

```sh
$ rpm -e gpg-pubkey-098acf3b-55a5351a
```

2. Modify /etc/yum.repos.d/signalfx-agent.repo. Specifically: 

 * Change baseurl to https://splunk.jfrog.io/splunk/signalfx-agent-rpm/release

 * Change gpgkey to https://splunk.jfrog.io/splunk/signalfx-agent-rpm/splunk-B3CD4420.pub

 The file should now look like:
```sh
[signalfx-agent]
name=SignalFx Agent Repository
baseurl=https://splunk.jfrog.io/splunk/signalfx-agent-rpm/release
gpgcheck=1
gpgkey=https://splunk.jfrog.io/splunk/signalfx-agent-rpm/splunk-B3CD4420.pub
enabled=1
```

3. Upgrade to the latest signalfx-agent package.

```sh
$ sudo yum update signalfx-agent
```

4. You will be prompted to import the key with the fingerprint 58C3 3310 B7A3 54C1 279D  B669 5EFA 01ED B3CD 4420. Press **y** to accept.

### Chef, Ansible, Puppet, and Salt

For Chef, Ansible, Puppet, and Salt users, you must update and run the latest
version to configure with the new Smart Agent repositories. This action will
ensure that the new key is added and that the old key is removed.


To verify that the old keys were removed:

- APT:
  1. Run the following command:
  ```sh
  $ apt-key list
  ```
  2. Verify that the key ending in 5AE495F6 is not present. You should see 58C3 3310 B7A3 54C1 279D  B669 5EFA 01ED B3CD 4420.
- RPM:
  1. Run the following command:
    ```sh
    $ rpm -q gpg-pubkey --qf '%{NAME}-%{VERSION}-%{RELEASE}\t%{SUMMARY}\n'
    ```
  2. Verify that the gpg-pubkey-098acf3b-55a5351a key is not present. You should only see gpg-pubkey-b3cd4420-5b5b79b1.

