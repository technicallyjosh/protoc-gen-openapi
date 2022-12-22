FROM golang:1.19-alpine3.16 as golang

RUN go install github.com/golang/protobuf/protoc-gen-go@v1.5.2

WORKDIR /workspace

COPY api ./api

FROM bufbuild/buf:1.7.0 as buf

COPY --from=golang /go/bin/protoc-gen-go /usr/local/bin/

ENTRYPOINT ["buf"]
