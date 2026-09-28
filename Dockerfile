FROM golang:1.24-bookworm

RUN apt-get update && apt-get install -y \
    protobuf-compiler \
    git \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /go/src

RUN wget https://github.com/tidwall/gjson/archive/refs/tags/v1.18.0.tar.gz
RUN tar xf v1.18.0.tar.gz && rm v1.18.0.tar.gz


WORKDIR /go/src/gjson-1.18.0

COPY artifacts artifacts

RUN mv artifacts/fuzz_parse_test.go .
RUN sed -i '/github.com\/yandex-cloud\/go-protobuf-mutator/d' go.mod

RUN go get \
    github.com/yandex-cloud/go-protobuf-mutator@latest \
    google.golang.org/protobuf@latest

RUN go mod tidy

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest


RUN mkdir -p artifacts/testdata/fuzz/FuzzParseJSON


CMD ["/usr/bin/bash"]