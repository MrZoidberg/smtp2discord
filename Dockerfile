# syntax=docker/dockerfile:1

FROM alpine:3.21
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/smtp2discord /usr/bin/smtp2discord
USER 65534:65534
ENTRYPOINT ["/usr/bin/smtp2discord"]
