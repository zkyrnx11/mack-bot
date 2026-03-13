FROM python:3.12-slim

ARG VERSION=latest

RUN apt-get update && apt-get install -y --no-install-recommends \
      ffmpeg \
      ca-certificates \
      tzdata \
      wget \
      tar \
      jq \
    && rm -rf /var/lib/apt/lists/*

RUN if [ "$VERSION" = "latest" ]; then \
        VERSION=$(wget -qO- https://api.github.com/repos/zkyrnx11/mack-bot/releases/latest | jq -r .tag_name | sed 's/^v//'); \
    fi && \
    wget -qO mack.tar.gz "https://github.com/zkyrnx11/mack-bot/releases/download/v${VERSION}/mack_${VERSION}_linux_amd64.tar.gz" && \
    tar -xzf mack.tar.gz -C /usr/local/bin && \
    rm mack.tar.gz

COPY scraper /tmp/scraper
RUN pip install --no-cache-dir /tmp/scraper && \
    playwright install chromium --with-deps && \
    rm -rf /tmp/scraper

RUN mkdir -p /root/Documents/Mack-Bot
WORKDIR /root/Documents/Mack-Bot

ENTRYPOINT ["mack"]
