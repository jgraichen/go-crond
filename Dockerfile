FROM cgr.dev/chainguard/static:latest@sha256:41e17ed83c594a64a9396b6ab96dd26d5ddc290dacf4c177464712ff21ad534f

ARG TARGETPLATFORM

COPY $TARGETPLATFORM/go-crond /usr/bin/

CMD ["/usr/bin/go-crond"]
