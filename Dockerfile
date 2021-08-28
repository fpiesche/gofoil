FROM golang:alpine3.14 AS builder

WORKDIR /build
ADD . /build
RUN go build

FROM alpine:3.14.2

COPY --from=builder /build/gofoil /bin/gofoil
CMD [ "/bin/gofoil" ]