FROM cgr.dev/chainguard/static:latest@sha256:c12f0107f8afb0bbc1b0eff96ff8675e85a10c6083d039b777c51fef5c27420f

ARG TARGETPLATFORM

COPY $TARGETPLATFORM/go-crond /usr/bin/

CMD ["/usr/bin/go-crond"]
