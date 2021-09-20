# Releasing the Agent

To release the agent make sure you have the following configured on your
workstation.  We will ideally make this released by CircleCI or Jenkins at some
point but for now the process is manual.

# Setup

1. Request prod access via slack and the `splunkcloud_account_power` role with
   `okta-aws-setup us0`.

1. Ensure you are authorized to push images to the Quay.io Docker repository
   `quay.io/signalfx/signalfx-agent`.

1. Ensure you are on the Splunk network and have access to the required
   credentials for artifactory and signing (check with an Integrations team
   member for details).

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

1. If the Helm assets have changed bump the chart version number in
   [Chart.yaml](deployments/k8s/helm/signalfx-agent/Chart.yaml) and commit/push
   the changes.

1. If there are relevant changes in the [python](./python) directory, bump the
   version in [setup.py](./python/setup.py) and commit/push the changes.

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
   with the new tag (may take up to 30 minutes; if you have permissions, you
   can trigger the sync immediately from the repo settings in gitlab).  The
   CI/CD pipeline will then trigger automatically for the new tag.

1. Ensure that the build and release jobs in gitlab for the tag are successful
   (may take over 30 minutes to complete).
   1. Ensure that the `quay.io/signalfx/signalfx-agent:<version>-windows`
      image was built and pushed.
   1. Ensure that the choco package was pushed to [chocolatey](
      https://community.chocolatey.org/packages/signalfx-agent).

1. Run the release script:

   ```sh
   $ scripts/release --artifactory-token <splunk.jfrog.io token> --chaperone-token <chaperone token> --staging-token <repo.splunk.com token>
   ```

   This will run for several minutes and will build/sign/push the docker image,
   deb, and rpm.  The linux bundle will be saved to
   `./signalfx-agent-<version>.tar.gz` and will need to be manually uploaded to
   the [Github Release](#github-release).  If there is an error, it will output
   on the command line.  Otherwise, the output should say "Successfully released
   <version>".

1. Create the `./build/signed` directory in your local repo root.

1. Download the artifacts from the `win-bundle-sign` and `win-msi-sign` gitlab
   jobs to `./build/signed`.

1. Run the release script to push the msi and bundle to S3:

   ```
   $ scripts/release --stage <STAGE> --push --component windows
   ```

   Where `<STAGE>` is `test`, `beta`, or `release`.

1. Build and release the certified RedHat container by running:

   ```sh
   $ scripts/release-redhat <X.Y.Z> <OSPID>
   ```

1. Wait for the RedHat build to complete and then publish it.

1. If the Helm assets have changed then update the repo from `dtools/helm_repo`
   by running (requires S3 access):

   ```sh
    AGENT_CHART_DIR=<agent dir>/deployments/k8s/helm/signalfx-agent ./update agent
    ```

1. Test out the new release by deploying it to a test environment and ensuring
   it works.

1. If the docs have changed since the last release, update the product docs
   repository by running the script `scripts/docs/to-product-docs`.  If the
   README has been updated, you will also need to run the script
   `scripts/docs/to-integrations-repo` to update the agent tile contents,
   which is based on the README.

   To release product docs, first ensure that you have pandoc installed (on
   Mac you can do `brew install pandoc`).  Next checkout the git repo
   github.com/signalfx/product-docs to your local workstation and run
   `PRODUCT_DOCS_REPO=<path to product docs> scripts/docs/to-product-docs`.

## Github Release

1. After completing the previous steps, create a [Github release](
   https://github.com/signalfx/signalfx-agent/releases) for the tag.

1. Get the `sha256` docker image digest by running:

   ```sh
   $ docker inspect --format='{{.RepoDigests}}' quay.io/signalfx/signalfx-agent:<version>
   ```

1. Add the release notes including the deprecation notice and the docker image
   digest from the previous step (see previous releases for reference).

1. Upload the `./signalfx-agent-<version>.tar.gz` bundle, and the
   zip bundle and MSI from the `./build/signed` directory to the release.

1. Publish the release.
