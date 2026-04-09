FROM --platform=$BUILDPLATFORM golang:1.23.0-alpine3.20 AS build

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/hermes ./cmd/hermes

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -h /home/hermes hermes

COPY --from=build /out/hermes /usr/local/bin/hermes

USER hermes
WORKDIR /home/hermes

ENTRYPOINT ["/usr/local/bin/hermes"]
CMD ["version"]
