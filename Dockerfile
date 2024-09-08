ARG GOLANG_VERSION=1.22

FROM golang:${GOLANG_VERSION} AS builder
RUN apt-get update && apt-get install -qq -y ca-certificates musl-tools
ENV SERVICE_HOME=/opt/livery-stable
WORKDIR ${SERVICE_HOME}
COPY . .
RUN CGO_ENABLED=1 \
    CC=musl-gcc \
    go build \
    -tags musl \
    -ldflags '-extldflags "-static"' \
    -mod vendor \
    -o ./out \
    cmd

FROM scratch
ENV SERVICE_HOME=/opt/livery-stable
ARG VERSION
ENV VERSION=${VERSION}
WORKDIR ${SERVICE_HOME}
COPY --from=builder ${SERVICE_HOME}/out .
COPY --from=builder /etc/ssl/certs /etc/ssl/certs
EXPOSE 8080
ENTRYPOINT ["out/livery-stable"]