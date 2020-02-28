# Releasing the Agent

To release the agent make sure you have the following configured on your
workstation.  We will ideally make this released by CircleCI or Jenkins at some
point but for now the process is manual.

*Note:* The Windows release process is currently separate from everything else
(see the "Windows Release Process" section below).

# Setup

1. Add a profile called `prod` to your AWS CLI tool config that contains your
   IAM credentials to our production AWS account.  The default region does not
   matter because we only deal with S3 and CloudFront, which are regionless.
   This is genrally done by adding a section with the header `[prod]` in the
   file `~/.aws/credentials`.

1. Ensure you are authorized to push images to the Quay.io Docker repository
   `quay.io/signalfx/signalfx-agent`.

1. Ensure you are on the Splunk network and have access to the required
   credentials for artifactory and signing (check with an Integrations team
   member for details).

1. Create a Github access token by going to [Personal Access tokens](
   https://github.com/settings/tokens) on Github.  Create a new token that can
   write to the SignalFx Agent repo.  Save the token somewhere where you can
   access it later.

   We need a Github token to create the Github release and upload the
   standalone bundle to it as an asset.  The release script will do both of
   those things automatically.

1. Install Python tools to update the Python package in the `python/`
   directory if it has changed since the last release:

   ```sh
   $ pip install --user keyring twine setuptools wheel
   ```

   Then set your password for Pypi by running the following command:

   ```sh
   $ keyring set https://upload.pypi.org/legacy/ your-username
   ```

## Release Process

1. Make sure everything that go out in the release is in the `master` branch.
   If so, checkout the master branch locally and ensure you are up to date
   with Github.

1. Examine the differences since the last release.  The simplest way to do
   this is to go to the [releases
   page](https://github.com/signalfx/signalfx-agent/releases) and click on the
   link for "<N> commits to master since this release" for the last release.
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

1. Run the release script:

   ```sh
   $ scripts/release --github-user <github username> --github-token <github token> --artifactory-token <splunk.jfrog.io token> --chaperone-token <chaperone token> --staging-token <repo.splunk.com token>
   ```

   Using the service account tokens and your personal Github token created
   earlier in the Setup section.

   This will run for several minutes.  If there is an error, it will output on
   the command line.  Otherwise, the output should say "Successfully released
   <version>", at which point you are done.

1. Build and release the certified RedHat container by running:

   ```sh
   $ scripts/release-redhat <X.Y.Z> <OSPID>
   ```

1. Wait for the RedHat build to complete and then publish it.

1. If the Helm assets have changed bump the chart version number in [Chart.yaml](deployments/k8s/helm/signalfx-agent/Chart.yaml)
   then update the repo from `dtools/helm_repo` by running:

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

# Windows Release Process

## Setup

1. You must be on the Splunk network and have access to the required credentials
   for signing (check with an Integrations team member for details).

1. You must have a Github access token to publish the agent bundle to Github Releases.

1. You must have your AWS CLI set up on your local workstation and have access to our
   S3 bucket.

1. You must have access to a Windows machine that is provisioned with the required
   build tools.  Alternatively, you can build, provision, and start the Windows
   Server 2016 vagrant box. See the "Windows" section in
   [development.md](docs/development.md) for details.

## Release Process

1. Open a Powershell terminal in the Windows virtual machine and execute:

   ```
   $ cd c:\users\vagrant\signalfx-agent
   $ scripts/windows/make.ps1 bundle -AGENT_VERSION "<X.Y.Z>"
   ```

   Where `<X.Y.Z>` is the release version.

1. If the build is successful, verify that
   `c:\users\vagrant\signalfx-agent\build\SignalFxAgent-X.Y.Z-win64.zip` exists.

1. If everything looks good, run the release script from your local workstation (must be
   on the Splunk network):

   ```
   $ scripts/release --stage <STAGE> --push --new-version <X.Y.Z> --component windows --staging-token <repo.splunk.com token> --chaperone-token <chaperone token>
   ```

   Where `<STAGE>` is `test`, `beta`, or `release`,`<X.Y.Z>` is the same version from
   step 1, and `<repo.splunk.com token>` and `<chaperone token>` are the API tokens for
   the `srv-signalfx-agent` service account.

1. Install/deploy the new release by running the installer script in a Powershell terminal
   (replace `YOUR_SIGNALFX_API_TOKEN` and `STAGE` with the appropriate values):

   ```
   $ & {Set-ExecutionPolicy Bypass -Scope Process -Force; $script = ((New-Object System.Net.WebClient).DownloadString('https://dl.signalfx.com/signalfx-agent.ps1')); $params = @{access_token = "YOUR_SIGNALFX_API_TOKEN"; stage = "STAGE"}; Invoke-Command -ScriptBlock ([scriptblock]::Create(". {$script} $(&{$args} @params)"))}
   ```
