FROM alpine:3.22 AS certs

FROM tianon/toybox:0.8.13

ARG TARGETPLATFORM

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY ${TARGETPLATFORM}/homer-go /usr/local/bin/homer-go

RUN mkdir -p /data

WORKDIR /data

ENV HOMER_GO_ADDR=:8732

EXPOSE 8732
VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/homer-go"]
