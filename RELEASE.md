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

1. If using Mac, create the directories `/opt/signalfx-agent-deb-cache` and
   `/opt/signalfx-agent-rpm-cache` on your Mac filesystem.  Then add those
   directories to the allowed list in Docker Preferences -> File Sharing, and
   then click "Apply & Restart" to enable this config in Docker for Mac.

1. Ensure you are authorized to push images to the Quay.io Docker repository
   `quay.io/signalfx/signalfx-agent`.

1. Import the GPG keys for Debian and RPM package signing.  These are two
   separate keys that you will have to obtain securely from somebody else on
   the project who has them.  Once you have them, you can import them with
   `gpg2 --import <keyfile>`.

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
   $ scripts/release --github-user <github username> --github-token <github token>
   ```

   Using the Github token created earlier in the Setup section.

   This will run for several minutes.  If there is an error, it will output on
   the command line.  Otherwise, the output should say "Successfully released
   <version>", at which point you are done.

1. Build and release the certified RedHat container by running:

   ```sh
   $ scripts/release-redhat <X.Y.Z> <OSPID>
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

1. You must have access to the `SFX_Windows.pfx` code signing certificate and password.

1. You must have your AWS CLI set up on your local workstation and have access to our
   S3 bucket.

1. Build, provision, and start the Windows Server 2008 vagrant box. See the "Windows"
   section in [development.md](docs/development.md) for details.

1. Copy `SFX_Windows.pfx` to some folder in the virtual machine.

## Release Process

1. Open a Powershell terminal in the Windows virtual machine and execute:

   ```
   $ cd c:\users\vagrant\signalfx-agent
   $ scripts/windows/make.ps1 bundle -AGENT_VERSION "<X.Y.Z>" -PFX_PATH "<PFX_PATH>" -PFX_PASSWORD "<PFX_PASSWORD>"
   ```

   Where `<X.Y.Z>` is the release version, `<PFX_PATH>` is the path to `SFX_Windows.pfx`
   in the virtual machine, and `<PFX_PASSWORD>` is the password for `SFX_Windows.pfx`.

1. If the build is successful, verify that
   `c:\users\vagrant\signalfx-agent\build\SignalFxAgent\bin\signalfx-agent.exe` is signed and that
   `c:\users\vagrant\signalfx-agent\build\SignalFxAgent-X.Y.Z-win64.zip` exists.

1. If everything looks good, run the release script from your local workstation:

   ```
   $ scripts/release --stage <STAGE> --push --new-version <X.Y.Z> --component windows
   ```

   Where `<STAGE>` is `test`, `beta`, or `final` and `<X.Y.Z>` is the same version from step 1.

1. Install/deploy the new release by running the installer script in a Powershell terminal
   (replace `YOUR_SIGNALFX_API_TOKEN` and `STAGE` with the appropriate values):

   ```
   $ & {Set-ExecutionPolicy Bypass -Scope Process -Force; $script = ((New-Object System.Net.WebClient).DownloadString('https://dl.signalfx.com/signalfx-agent.ps1')); $params = @{access_token = "YOUR_SIGNALFX_API_TOKEN"; stage = "STAGE"}; Invoke-Command -ScriptBlock ([scriptblock]::Create(". {$script} $(&{$args} @params)"))}
   ```
