###############################
# STEP 1 build the application
###############################
ARG GO_VERSION=0
ARG ALPINE_VERSION=0
ARG BASE_IMAGE_VERSION=0

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

RUN apk add --no-cache bash git upx
RUN mkdir -p /acceldata/src/gurl /acceldata/bin

COPY . /acceldata/src/gurl
ENV GOPATH=/acceldata
WORKDIR /acceldata/src/gurl
RUN go mod vendor && \
    \
    env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=vendor -a -installsuffix cgo -gcflags=all='-l -B' -ldflags '-s -w' -o /acceldata/bin/gurl && \
    \
    upx -9 -k /acceldata/bin/gurl

#############################################
# STEP 2 copy gurl binary to the base image
#############################################

# Use OpenJRE KRB5 Base Image
FROM alpine:${BASE_IMAGE_VERSION}
# Copy gurl static executable binary
COPY --from=builder /acceldata/bin/gurl /usr/bin/gurl
