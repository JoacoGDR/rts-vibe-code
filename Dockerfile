FROM golang:1.25.8-alpine AS build
WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG LDFLAGS="-s -w -X main.Version=${VERSION}"
RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="${LDFLAGS}" -o /out/supremacy ./cmd/supremacy && \
    go build -trimpath -ldflags="${LDFLAGS}" -o /out/core-api ./cmd/core-api && \
    go build -trimpath -ldflags="${LDFLAGS}" -o /out/gateway ./cmd/gateway && \
    go build -trimpath -ldflags="${LDFLAGS}" -o /out/engine ./cmd/engine && \
    go build -trimpath -ldflags="${LDFLAGS}" -o /out/worker ./cmd/worker && \
    go build -trimpath -ldflags="${LDFLAGS}" -o /out/ai-bot ./cmd/ai-bot

FROM alpine:3.20
RUN apk add --no-cache ca-certificates curl tini && adduser -D -u 10001 supremacy
USER supremacy
COPY --from=build /out/supremacy /usr/local/bin/supremacy
COPY --from=build /out/core-api /usr/local/bin/core-api
COPY --from=build /out/gateway /usr/local/bin/gateway
COPY --from=build /out/engine /usr/local/bin/engine
COPY --from=build /out/worker /usr/local/bin/worker
COPY --from=build /out/ai-bot /usr/local/bin/ai-bot
EXPOSE 8080 9100
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/usr/local/bin/core-api"]
