#!/bin/bash

set -e

echo "🚀 Deploying GoLab..."

cd /home/ec2-user/golab

echo "📥 Pulling latest code..."
git pull --ff-only origin main

cd backend

echo "🔨 Building..."
go build -o app .

echo "💾 Ensuring Go build cache volume exists..."
docker volume inspect golab-go-cache >/dev/null 2>&1 || \
    docker volume create golab-go-cache

echo "🔐 Fixing cache permissions..."
docker run --rm \
    -u 0 \
    -v golab-go-cache:/gocache \
    alpine:3.22 \
    sh -c "chown -R 10001:10001 /gocache"

echo "🔄 Restarting service..."
sudo systemctl restart golab

echo "✅ Deployment complete!"
sudo systemctl is-active --quiet golab && echo "🟢 GoLab is running."
