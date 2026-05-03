FROM golang:1.25.8-alpine AS build
WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath \
    -ldflags="-s -w -X main.Version=${VERSION}" \
    -o /out/supremacy ./cmd/supremacy

FROM alpine:3.20
RUN apk add --no-cache ca-certificates curl tini && adduser -D -u 10001 supremacy
USER supremacy
COPY --from=build /out/supremacy /usr/local/bin/supremacy
EXPOSE 8080 9100
ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/supremacy"]
CMD ["core-api"]
