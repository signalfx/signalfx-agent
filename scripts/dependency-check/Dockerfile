FROM owasp/dependency-check:5.1.1
USER root

RUN rm -f /etc/apt/sources.list.d/microsoft-prod.list && \
    apt-get update && \
    apt-get install -y --no-install-recommends apt-transport-https wget

ARG GO_VERSION=1.12.1
RUN wget -nv https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz && \
    rm go${GO_VERSION}.linux-amd64.tar.gz

ENV PATH=$PATH:/usr/share/dependency-check/bin:/usr/local/go/bin

VOLUME ["/usr/src/signalfx-agent" "/usr/share/dependency-check/data"]

WORKDIR /usr/src/signalfx-agent

ENTRYPOINT ["dependency-check.sh"]
CMD ["--help"]
