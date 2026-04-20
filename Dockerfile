FROM cgr.dev/chainguard/static:latest@sha256:6d508f497fe786ba47d57f4a3cffce12ca05c04e94712ab0356b94a93c4b457f

ARG TARGETPLATFORM

COPY $TARGETPLATFORM/go-crond /usr/bin/

CMD ["/usr/bin/go-crond"]
