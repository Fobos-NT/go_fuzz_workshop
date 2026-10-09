FROM golang:1.25-bookworm

RUN apt-get update && \
    apt-get install -y clang protobuf-compiler git && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /go/src/go_fuzz_workshop

COPY . .

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

RUN git clone https://github.com/Fobos-NT/go-118-fuzz-build.git /tmp/go-118-fuzz-build && \
    cd /tmp/go-118-fuzz-build && \
    git checkout 25f75035e16d5ff06a56b9ee86ca3171f2332e8e && \
    go build -o /usr/local/bin/go-118-fuzz-build .

RUN protoc \
    --go_out=. \
    --go_opt=paths=source_relative \
    artifacts/json.proto

RUN go get github.com/tidwall/gjson@v1.18.0
RUN go get github.com/yandex-cloud/go-protobuf-mutator@v1.1.0

# RUN go-118-fuzz-build \
#     -proto \
#     -proto_format binary \
#     -func FuzzParseJSON \
#     -o gjson_fuzz.a \
#     go_fuzz_workshop

# RUN clang++ \
#     -fsanitize=fuzzer,address \
#     -o gjson_fuzz \
#     gjson_fuzz.a

CMD ["/usr/bin/bash"]