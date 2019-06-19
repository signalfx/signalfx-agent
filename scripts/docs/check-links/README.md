# Markdown Link Checker

Docker image with [`markdown-link-check`](https://github.com/tcort/markdown-link-check).

Run `make check-links` from the root of the repository to scan all markdown documents
in this repository (excluding ones within the `vendor` directory).

Update [config.json](./config.json) to exclude specific links from being checked (e.g.
examples or links requiring authorization).  Check
[this](https://github.com/tcort/markdown-link-check/blob/master/README.md#config-file-format)
for details and other options.
