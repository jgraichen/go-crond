FROM cgr.dev/chainguard/static:latest@sha256:24dd7ff8788fdfadda39eeeaefefb6d1cec6002a545935a5f7e017484053734f

ARG TARGETPLATFORM

COPY $TARGETPLATFORM/go-crond /usr/bin/

CMD ["/usr/bin/go-crond"]
