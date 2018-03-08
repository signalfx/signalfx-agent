FROM ubuntu:16.04

ENV PATH=$PATH:/opt/puppetlabs/bin:/opt/puppetlabs/pdk/bin

WORKDIR /tmp
RUN apt update &&\
    apt install -y make wget vim gcc ruby ruby-dev libxml2 libxml2-dev libxslt1-dev &&\
	wget https://apt.puppetlabs.com/puppet5-release-xenial.deb &&\
	dpkg -i puppet5-release-xenial.deb &&\
	rm *.deb &&\
	apt update &&\
	apt install -y puppet-agent

RUN wget -O pdk.deb 'https://pm.puppetlabs.com/cgi-bin/pdk_download.cgi?dist=ubuntu&rel=16.04&arch=amd64&ver=latest' &&\
    dpkg -i pdk.deb &&\
	rm *.deb

WORKDIR /etc/puppetlabs/code/modules/signalfx_agent
COPY ./Rakefile ./Gemfile ./Gemfile.lock ./
RUN gem install bundle && bundle install

COPY ./ ./
