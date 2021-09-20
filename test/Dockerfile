FROM golang:1.15
WORKDIR /awat/
COPY . /awat/
RUN make build
ENTRYPOINT ["/usr/bin/make", "e2e"]
