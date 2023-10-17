FROM golang:1.21-alpine3.18 as build

ENV CGO_ENABLED=0

RUN apk add gcc protoc protobuf-dev

WORKDIR /workspace

COPY api api
COPY internal internal
COPY Makefile go.* main.go ./
RUN go mod download
RUN go install .

COPY test test
COPY main_test.go ./

CMD ["go", "test", "./..."]
