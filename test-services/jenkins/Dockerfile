ARG JENKINS_VERSION=alpine
FROM jenkins:${JENKINS_VERSION}
COPY metrics.groovy /usr/share/jenkins/ref/init.groovy.d/metrics.groovy
USER root
RUN apk update && apk add coreutils
RUN curl -sSl https://raw.githubusercontent.com/jenkinsci/docker/master/install-plugins.sh -o /tmp/install-plugins.sh && \
    chmod a+x /tmp/install-plugins.sh
RUN curl -sSl https://raw.githubusercontent.com/jenkinsci/docker/master/jenkins-support -o /usr/local/bin/jenkins-support
RUN /tmp/install-plugins.sh docker-slaves metrics

ARG JENKINS_PORT

ENV JENKINS_OPTS --httpPort=${JENKINS_PORT}
ENV JAVA_OPTS="-Djenkins.install.runSetupWizard=false"
EXPOSE ${JENKINS_PORT}
USER jenkins
