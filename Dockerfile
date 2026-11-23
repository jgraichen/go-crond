FROM cgr.dev/chainguard/static:latest@sha256:8658e9eaeb2294fa0e49b0667f7d66b4e21fa7690b8d2a4c30cd1d013ceb9eb7

ARG TARGETPLATFORM

COPY $TARGETPLATFORM/go-crond /usr/bin/

CMD ["/usr/bin/go-crond"]
