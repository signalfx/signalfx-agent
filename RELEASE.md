# Releasing the Agent

To release the agent make sure you have the following configured on your
workstation.  We will ideally make this released by CircleCI or Jenkins at some
point but for now the process is manual.

# Setup

1. Install the [AWS CLI](
   https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html).

1. Request prod access via slack and the `splunkcloud_account_power` role with
   `okta-aws-setup us0`.

1. Install Python tools to update the Python package in the `python/`
   directory if it has changed since the last release:

   ```sh
   $ pip install --user keyring twine setuptools wheel
   ```

   Then set your password for Pypi by running the following command:

   ```sh
   $ keyring set https://upload.pypi.org/legacy/ your-username
   ```

1. Ensure you have access to the `o11y-gdi/signalfx-agent-releaser` gitlab
   repo and CI/CD pipeline.

1. Ensure you have access to the Red Hat Container Certification Project, the
   project ID, and a personal API key.  See [here](
   https://redhat-connect.gitbook.io/partner-guide-for-red-hat-openshift-and-container/appendix/connect-portal-api/project-creation#api-key)
   for details on how to create a personal API key.

1. Clone the [integrations repository](https://github.com/signalfx/integrations)
   for documentation updates.

1. Clone the product docs repository from GitLab (`observability/docs/product-docs`).

## Release Process

1. Make sure everything that go out in the release is in the `main` branch.
   If so, checkout the main branch locally and ensure you are up to date
   with Github.

1. Examine the differences since the last release.  The simplest way to do
   this is to go to the [releases
   page](https://github.com/signalfx/signalfx-agent/releases) and click on the
   link for "<N> commits to main since this release" for the last release.
   This will give you a commit list and diff in the browser.

   You can also do `git cherry -v <last release tag>` to see the commit
   summaries.

1. Determine the next release version.  If this is a very simple, non-breaking
   change or a simple addition to existing functionality, a patch release may
   be appropriate (i.e. only the last number of the version is incremented).
   If there are any breaking changes, it should be at least a minor release
   (i.e. the second number of the version increments and the last number
   resets to 0), if not a major release (where the first # of a release
   increments and the second and third component reset to 0).  Major releases
   should be reserved for only very major breaking changes that are of high
   value.

   We roughly follow [semver](https://semver.org/), but not terribly
   strictly and with the additional consideration that we are not only
   considering an "API" but also MTSs.  For example, if you are going to make
   a change that would add a new dimension to existing metrics, given the same
   configuration, this is considered a breaking change since it would result
   in new MTSs in the backend.

1. Update the deployment versions with the new version determined from the
   previous step (without the `v`) and commit/push the changes:

   ```sh
   $ ./scripts/update-deployments-version <version>
   ```

1. If the Helm assets have changed (apart from the agent version updates from
   the previous step), bump the chart version number in
   [Chart.yaml](deployments/k8s/helm/signalfx-agent/Chart.yaml) and commit/push
   the changes. This can be determined by checking to see if any changes have
   been made in the [Helm directory](deployments/k8s/helm).

1. Once you know the next release version, create an annotation tag of the
   form `v<version>` where `<version>` is that version.  E.g. a release of
   2.5.2 would need a tag `v2.5.2`.  Annotated tags are created by passing the
   `-a` flag to `git tag`:

   ```sh
   $ git tag -a v2.5.2
   ```

   This will open your configured text editor and let you write the
   annotation.  This should be of the form (assuming you are releasing version
   2.5.2):

   ```
   2.5.2

   - Did something to the agent
   - Fixed this bug

   Breaking Changes:

   - This thing won't work anymore
   ```

   If there are no breaking changes, you can omit that section.

   Then push that tag with `git push --tags`.

1. Wait for the `o11y-gdi/signalfx-agent-releaser` gitlab repo to be synced
   with the new tag (may take up to 30 minutes). The CI/CD pipeline will then
   trigger automatically for the new tag.
   - If you have `Maintainer` permissions or above, you can trigger the sync
     immediately. Go to `Settings` -> `Repository` -> `Mirroring repositories` ->
     `Click button to update existing mirrored repository`

1. Ensure that the build and release jobs in gitlab for the tag are successful
   (may take over 30 minutes to complete).
   1. Ensure that the [Github Release](
      https://github.com/signalfx/signalfx-agent/releases) for the tag was
      created and the linux tar.gz bundle, windows zip bundle, and windows MSI
      were uploaded.
   1. Ensure that the `quay.io/signalfx/signalfx-agent:<version>` image was
      built and pushed.
   1. Ensure that the `quay.io/signalfx/signalfx-agent:<version>-windows`
      image was built and pushed.
   1. Ensure that the choco package was pushed to [chocolatey](
      https://community.chocolatey.org/packages/signalfx-agent). Some jobs may take
      a day or two to be started.
   1. Ensure that `https://dl.signalfx.com/windows/release/msi/SignalFxAgent-<version>-win64.msi`
      and `https://dl.signalfx.com/windows/release/zip/SignalFxAgent-<version>-win64.zip`
      were uploaded.
   1. Ensure that the `https://dl.signalfx.com/signalfx-agent.ps1` and
      `https://dl.signalfx.com/signalfx-agent.sh` installer scripts were released
      by comparing the remote files with the local files:
      ```sh
      $ curl -sSL https://dl.signalfx.com/signalfx-agent.ps1 | diff - deployments/installer/install.ps1
      $ curl -sSL https://dl.signalfx.com/signalfx-agent.sh | diff - deployments/installer/install.sh
      ```

1. If the Helm [Chart.yaml](./deployments/k8s/helm/signalfx-agent/Chart.yaml)
   version was updated from step #5, then update the repo from
   `dtools/helm_repo` by running (requires S3 access):
   1. Get `us0` realm credentials through Okta and select the
      `signalfx/splunkcloud_account_power` role.
      ```sh
      $ okta-aws-setup us0
      ```
   1. Clone the `dtools` repo if you haven't already.
   1. Run:
      ```sh
      $ cd <dtools repo dir>/helm_repo
      $ AGENT_CHART_DIR=<agent repo dir>/deployments/k8s/helm/signalfx-agent ./update agent
      ```

1. Test out the new release by deploying it to a test environment and ensuring
   it works.
