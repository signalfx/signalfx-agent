ARG JENKINS_REPO=jenkins/jenkins
ARG JENKINS_VERSION=alpine
FROM ${JENKINS_REPO}:${JENKINS_VERSION}
COPY metrics.groovy /usr/share/jenkins/ref/init.groovy.d/metrics.groovy
RUN jenkins-plugin-cli -p docker-slaves metrics

ARG JENKINS_PORT

ENV JENKINS_OPTS --httpPort=${JENKINS_PORT}
ENV JAVA_OPTS="-Djenkins.install.runSetupWizard=false"
EXPOSE ${JENKINS_PORT}
