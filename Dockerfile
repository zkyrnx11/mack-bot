FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG VERSION=0.0.1
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

RUN go build \
      -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildDate=${BUILD_DATE}" \
      -trimpath \
      -o /mack \
      ./

FROM python:3.12-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
      ffmpeg \
      ca-certificates \
      tzdata \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /mack /usr/local/bin/mack

COPY scraper/pyproject.toml /opt/mack-scraper/
COPY scraper/ /opt/mack-scraper/scraper/
RUN pip install --no-cache-dir /opt/mack-scraper && \
    playwright install chromium --with-deps

ENV PYTHONPATH=/opt/mack-scraper

RUN mkdir -p /root/Documents/Mack-Bot
WORKDIR /root/Documents/Mack-Bot

ENTRYPOINT ["mack"]
