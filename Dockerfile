# Support setting various labels on the final image
ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""

# Build Getx in a stock Go builder container
FROM golang:1.19-alpine as builder

RUN apk add --no-cache gcc musl-dev linux-headers git

# Get dependencies - will also be cached if we won't change go.mod/go.sum
COPY go.mod /go-ETX/
COPY go.sum /go-ETX/
RUN cd /go-ETX && go mod download

ADD . /go-ETX
RUN cd /go-ETX && go run build/ci.go install -static ./cmd/getx

# Pull Getx into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go-ETX/build/bin/getx /usr/local/bin/

EXPOSE 8545 8546 30303 30303/udp
ENTRYPOINT ["getx"]

# Add some metadata labels to help programatic image consumption
ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""

LABEL commit="$COMMIT" version="$VERSION" buildnum="$BUILDNUM"
