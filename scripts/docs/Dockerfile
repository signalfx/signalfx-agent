FROM ubuntu:18.04

RUN apt update -q &&\
    apt install -yq \
	  python3-pip \
	  git

WORKDIR /opt/agent
# Expected context path is the root of the agent repo
COPY docs/ ./docs/
COPY scripts/docs/ ./scripts/docs/
COPY pkg/ ./pkg/
COPY selfdescribe.json selfdescribe.json

RUN pip3 install -r ./scripts/docs/requirements.txt
