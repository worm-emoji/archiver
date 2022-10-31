#!/usr/bin/env sh

set -eux

flyctl auth docker

go run github.com/tailscale/mkctr@latest \
  --base="ghcr.io/tailscale/alpine-base:3.16" \
  --gopaths="github.com/worm-emoji/archiver/cmd/api:/usr/local/bin/api" \
  --ldflags="-X main.GitSha=`git rev-parse --short HEAD`" \
  --tags="latest" \
  --repos="registry.fly.io/archiver" \
  --target=flyio \
  --push \
	/usr/local/bin/api

flyctl deploy --detach \
	-i registry.fly.io/archiver:latest \
	-a archiver \
    -c fly.toml
