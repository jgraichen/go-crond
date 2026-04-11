FROM gcr.io/distroless/static

ARG TARGETPLATFORM

COPY $TARGETPLATFORM/go-crond /usr/bin/

CMD ["/usr/bin/go-crond"]
