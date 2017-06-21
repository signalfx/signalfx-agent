# NeoMock

This is a very stripped down mock of the Neoagent collectd plugin written
entirely in C, useful for doing memory leak and performance testing of our
collectd library.  It basically reimplements the [`Collectd.run`
method](../../plugins/monitors/collectd/collectd.go), except that reloads are
triggered by sending the HUP signal instead of being triggered by other
plugins.

This is built into the neoagent final image for ease of use.  Use the
`image-debug` make target to also get debugging symbols and gdb in the final
image to greatly help with debugging.

## Usage

Just run `neomock` in the neoagent container instead of the default command.
To simulate reloads of collectd, send the neomock process the `HUP` signal.

To generate a usable the collectd.conf file, it is helpful to run the neoagent
(`signalfx-agent`) first and let it generate the file.  Then run `neomock`.
