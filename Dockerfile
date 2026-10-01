# Build stage
FROM golang:1.26-alpine AS builder
ARG VERSION=dev
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-X main.version=${VERSION}" -o /out/myturn .

# Runtime stage
FROM alpine:3.20
ARG VERSION=dev
LABEL org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.title="myturn"
RUN apk add --no-cache tzdata \
	&& adduser -D -u 10001 myturn \
	&& mkdir -p /data \
	&& chown myturn:myturn /data
WORKDIR /app
COPY --from=builder /out/myturn /app/myturn

ENV PORT=8031
ENV MYTURN_CONFIG=/data/config.json
VOLUME ["/data"]
EXPOSE 8031

USER myturn
ENTRYPOINT ["/app/myturn"]
