FROM golang:1.8.3
MAINTAINER Gruntwork <info@gruntwork.io>

# Install sudo, cron, and gox
RUN apt-get update && apt-get install -y sudo cron rsyslog
RUN go get github.com/mitchellh/gox

# We need to put Go source code in the GOPATH, which is by default /go
ENV SRCPATH /go/src/github.com/gruntwork-io/package-openvpn/modules/openvpn-admin
WORKDIR $SRCPATH

# Install Glide
RUN curl https://glide.sh/get | sh

# Install dependencies. Note: we put the glide files one folder above the source code so that the vendor folder isn't
# overridden when you mount the source code from your host OS
COPY glide.yaml $SRCPATH/glide.yaml
COPY glide.lock $SRCPATH/glide.lock
RUN glide install --force

# Copy the rest of the source code
COPY src/* $SRCPATH/src/
COPY scripts/* $SRCPATH/scripts/

# Create a user that will be used in the automated tests
RUN useradd test-user

# As the default command, run the app
ENTRYPOINT ["./scripts/run.sh"]