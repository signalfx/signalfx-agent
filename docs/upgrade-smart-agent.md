# Overview

On XX/XX/XXX, SignalFx will launch the Smart Agent, version 5.0.

At a high-level, to better integrate with Splunk, this new version of the Smart Agent will be signed with Splunk certificates.

SignalFx recommends that you upgrade your Smart Agent as soon as possible.

Current users can continue to run Smart Agent 4.X; however, after XX/XX/XXX, any new installs or upgrades will require you to change to the new Splunk-signed repositories.

## Upgrade the Smart Agent

Review the following options to learn how to upgrade the Smart Agent:

### Option 1: APT

Note: For commands prefixed by #, you must run as root.

1. Delete the existing SignalFx apt key:
 - apt-key del 5AE495F6
 - You must run this command as root. 

2. Import the new key:
 - curl https://dl.signalfx.com/splunk-B3CD4420.gpg \
 | apt-key add -
 - You must run this command as root. 

3. Replace /etc/apt/sources.list.d/signalfx-agent.list with the contents:
  * deb  https://dl.signalfx.com/debs/signalfx-agent/release /

4. Update package metadata:
 - apt-get update
 - You must run this command as root. 

5. Install latest signalfx-agent:
 - apt-get upgrade signalfx-agent
 - You must run this command as root. 

### Option 2: YUM:

1. Delete the existing SignalFx RPM key:
  - rpm -e gpg-pubkey-098acf3b-55a5351a
  - You must run this command as root. 

2. Modify /etc/yum.repos.d/signalfx-agent.repo
  - Change baseurl to https://dl.signalfx.com/rpms/signalfx-agent/release
  - Change gpgkey to https://dl.signalfx.com/splunk-B3CD4420.pub

3. Upgrade to the latest signalfx-agent package:
  - yum update signalfx-agent
  - You must run this command as root. 

4. You will be prompted to import the key with the fingerprint 58C3 3310 B7A3 54C1 279D  B669 5EFA 01ED B3CD 4420. 
 - Press **y** to accept.

### Option 3: Chef, Ansible, Puppet, and Salt

For Chef, Ansible, Puppet, and Salt users, you must update and run the latest version to configure with the new Smart Agent repositories. This action will ensure that the new key is added and that the old key is removed.

1. Upgrade to the latest version:
  * Chef:
    * 1.0
  * Ansible:
    * https://github.com/signalfx/signalfx-agent/tree/v5.0.0/deployments/ansible
  * Puppet:
    * 1.0
  * Salt:
    * https://github.com/signalfx/signalfx-agent/tree/v5.0.0/deployments/salt

2. Verify that the old keys were removed:
  * Option 1: APT
    1. Run the following command: $ apt-key list
    2. Verify that the key ending in 5AE495F6 is not present. You should see 58C3 3310 B7A3 54C1 279D  B669 5EFA 01ED B3CD 4420.

  * Option 2: RPM
    1. Run the following command: 
      * $ rpm -q gpg-pubkey --qf '%{NAME}-%{VERSION}-%{RELEASE}\t%{SUMMARY}\n'
    2. Verify that the gpg-pubkey-098acf3b-55a5351a key is not present. You should only see gpg-pubkey-b3cd4420-5b5b79b1.

## Additional information

### Continue with Smart Agent 4.X

If you want to continue with Smart Agent 4.X, you must pin the agent version. Afterwards, you must run a check to verify that the old key was removed.

Note: Current users can continue to run Smart Agent 4.X; however, after XX/XX/XXX, any new installs or upgrades will require you to change to the new Splunk-signed repositories.

#### Option 1: APT

Note: For commands prefixed by #, you must run as root.

1. Delete the existing SignalFx apt key:
  * apt-key del 5AE495F6
  * You must run this command as root. 

2. Import the new key:
  * curl https://dl.signalfx.com/splunk-B3CD4420.gpg \
  | apt-key add -
  * You must run this command as root. 

3. Replace /etc/apt/sources.list.d/signalfx-agent.list with the contents:
  * deb  https://dl.signalfx.com/debs/signalfx-agent/release /

4. Update package metadata:
  * apt-get update
  * You must run this command as root. 

5. Pin the agent version.
  * To learn more, see https://github.com/signalfx/signalfx-agent/tree/master/deployments.

6. Run the following command:
  * $ apt-key list

7. Verify that the key ending in 5AE495F6 is not present.
  * You should see 58C3 3310 B7A3 54C1 279D  B669 5EFA 01ED B3CD 4420.

#### Option 2: RPM

1. Delete the existing SignalFx RPM key:
  * rpm -e gpg-pubkey-098acf3b-55a5351a
  * You must run this command as root. 

2. Modify /etc/yum.repos.d/signalfx-agent.repo
  * Change baseurl to https://dl.signalfx.com/rpms/signalfx-agent/release
  * Change gpgkey to https://dl.signalfx.com/splunk-B3CD4420.pub

3. Pin the agent version.
  * To learn more, see https://github.com/signalfx/signalfx-agent/tree/master/deployments.

4. Run the following command:
  * $ rpm -q gpg-pubkey --qf '%{NAME}-%{VERSION}-%{RELEASE}\t%{SUMMARY}\n'

5. Verify that the gpg-pubkey-098acf3b-55a5351a key is not present.
  * You should only see gpg-pubkey-b3cd4420-5b5b79b1.

### Locate your Smart Agent version

For APT, run:
  * dpkg -l signalfx-agent
For RPM, run:
  * rpm -q signalfx-agent

## Related documentation

To learn how to install the Smart Agent, see [Quick Install](./quick-install.md).
