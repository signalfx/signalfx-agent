# Agent Packaging

We package the agent for both Debian and RHEL based Linux distributions.  Since
all of the agent dependencies are bundled with it, it means there is really no
traditional compilation phase when making the packages -- it is largely just a
matter of getting the bundle into the right folder and making sure init scripts
get installed properly.

# Package Revisions

Sometimes it is useful to be able to revision a package even when there are no
agent bundle changes (for example, to change an init script or the default
config file).  You can do this by making the desired changes to the package and
then tagging with a new annotated tag of the form `v<agent
version>-(deb|rpm)[2-9]`, where `<agent version>` is the latest agent version
released, `(deb|rpm)` is either the string `deb` or `rpm` depending on which
package you are updating, and `[2-9]` is the revision number, starting at 2
(revision 1 is already implicitly used for the agent version release).  For
example, if agent 2.1.0 is the latest version and I update the Debian package
and want to release it before a newer agent is released, I commit the changes and
then make an **annotated** tag `v2.1.0-deb2`, which will generate a new Debian
package with version `2.1.0-2` that will supercede package version `2.1.0-1`
that should have been released when the agent version was incremented.
