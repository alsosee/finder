FROM golang:1.21 AS build-env

WORKDIR /go/src/app
COPY . /go/src/app

RUN go test -mod=vendor -cover ./codegen/...

ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-w -s" -mod=vendor -o /go/bin/codegen ./codegen


FROM scratch
COPY --from=build-env /go/bin/codegen /usr/local/bin/codegen
ENTRYPOINT ["/usr/local/bin/codegen"]
