from golang:1.14
workdir /awat/
copy . /awat/
run make build
entrypoint ["/usr/bin/make", "e2e"]
