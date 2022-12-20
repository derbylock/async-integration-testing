# Multi-stage build setup (https://docs.docker.com/develop/develop-images/multistage-build/)

# Stage 1 (to create a "build" image)
FROM golang:1.19.4 AS builder
RUN go version

COPY *.go /go/src/
COPY *.mod /go/src/
COPY *.sum /go/src/
COPY cmd /go/src/cmd
COPY internal /go/src/internal
COPY pkg /go/src/pkg
COPY vendor /go/src/vendor
COPY resources /go/src/resources
WORKDIR /go/src/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod vendor -a -o app .

# Stage 2 (to create a downsized "container executable")

# If you need SSL certificates for HTTPS, replace `FROM SCRATCH` with:
#
#   FROM alpine:3.7
#   RUN apk --no-cache add ca-certificates
#
FROM scratch
WORKDIR /root/
COPY --from=builder /go/src/app .
COPY --from=builder /go/src/resources ./resources

EXPOSE 9580
ENTRYPOINT ["./app"]