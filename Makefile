SHELL      := powershell.exe
.SHELLFLAGS := -NoProfile -Command

BINARY     := mack
VERSION    ?= 0.0.1
COMMIT     := $(shell git rev-parse --short HEAD 2>$$null)
LDFLAGS    := -s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT)
ENTRYPOINT := .

.PHONY: build run setup setup-ffmpeg setup-ytdlp setup-scraper upgrade icon release docker tag help

build:
	go build -ldflags "$(LDFLAGS)" -trimpath -o $(BINARY).exe $(ENTRYPOINT)

run:
	./$(BINARY).exe

setup: setup-ffmpeg setup-ytdlp setup-scraper
	@Write-Host "Setup complete. Run 'make build' to compile Mack-Bot."

setup-ffmpeg:
	@if (Get-Command ffmpeg -EA 0) { Write-Host "ffmpeg already installed: $$(ffmpeg -version 2>&1 | Select-Object -First 1)" } elseif (Get-Command winget -EA 0) { winget install --id Gyan.FFmpeg -e --accept-source-agreements --accept-package-agreements } elseif (Get-Command choco -EA 0) { choco install ffmpeg -y } elseif ($$IsMacOS) { brew install ffmpeg } else { sudo apt-get install -y ffmpeg }

setup-ytdlp:
	@if (Get-Command yt-dlp -EA 0) { Write-Host "yt-dlp already installed: $$(yt-dlp --version)" } elseif (Get-Command pipx -EA 0) { pipx install yt-dlp } elseif (Get-Command pip -EA 0) { pip install --user yt-dlp } elseif (Get-Command pip3 -EA 0) { pip3 install --user yt-dlp } else { Write-Host "Please install yt-dlp manually: https://github.com/yt-dlp/yt-dlp#installation" }

setup-scraper:
	@if (Get-Command pipx -EA 0) { pipx install --editable ./scraper } elseif (Get-Command pip -EA 0) { pip install --editable ./scraper } elseif (Get-Command pip3 -EA 0) { pip3 install --editable ./scraper } else { Write-Host "Please install Python 3.12+ and pip, then re-run 'make setup-scraper'" }

upgrade:
	go get -u ./...
	go mod tidy
	go build ./...

icon:
	go-winres make --product-version $(VERSION).0 --file-version $(VERSION).0

release:
	pwsh -NoProfile -File scripts/release.ps1 -Version $(VERSION)

docker:
	docker compose build

tag:
	git tag -a "v$(VERSION)" -m "Release v$(VERSION)"
	git push origin "v$(VERSION)"
	@Write-Host "Tagged and pushed v$(VERSION)"

help:
	@Select-String "^## " Makefile | ForEach-Object { $$_.Line -replace "^## ", "" }
