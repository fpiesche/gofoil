# Build stage: Copy repo to image and build.
FROM golang:alpine3.14 AS builder

WORKDIR /build
ADD . /build
RUN go build

# Copy gofoil to a new image from builder to keep image size small
# The base alpine image is about 10MB, rather than >100MB for golang:alpine
FROM alpine:3.14.2
COPY --from=builder /build/gofoil /bin/gofoil

EXPOSE 8000
VOLUME [ "/games" ]

ENV GOFOIL_LISTENADDRESS="0.0.0.0:8000" \
    GOFOIL_EXTERNALADDRESS="localhost:8000" \
    GOFOIL_ROOT="/games"

CMD [ "/bin/gofoil" ]
