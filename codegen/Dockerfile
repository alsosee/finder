FROM --platform=${TARGETPLATFORM:-linux/amd64} golang:1.21 as build-env

WORKDIR /go/src/codegen
ADD . /go/src/codegen

RUN go test -mod=vendor -cover ./...

ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-w -s" -mod=vendor -o /go/bin/codegen
