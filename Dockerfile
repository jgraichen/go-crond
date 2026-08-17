FROM cgr.dev/chainguard/static:latest@sha256:f68e3a8244c7d0f4cd56635aaff8e6a533cf6cc3850d8fb339567a5782d6a0b0

ARG TARGETPLATFORM

COPY $TARGETPLATFORM/go-crond /usr/bin/

CMD ["/usr/bin/go-crond"]
