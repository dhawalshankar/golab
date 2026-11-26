#!/usr/bin/env bash
# ==============================================================================
# GoLab — Complete Automated EC2 Server Bootstrap & Deployment Script
# Targets: Amazon Linux 2023 (AL2023) on AWS EC2 (t3.micro / t4g.micro)
# ==============================================================================

set -euo pipefail

echo "=================================================================="
echo "🚀 GOLAB BACKEND RUNNER — AUTOMATED SERVER BOOTSTRAP"
echo "=================================================================="

# 1. Update OS and Install Core Toolchains
echo "📦 [1/8] Updating package repository and installing Golang & Git..."
sudo dnf update -y
sudo dnf install -y golang git

echo "✓ Go Version: $(go version)"
echo "✓ Git Version: $(git --version)"

# 2. Install and Configure Docker
echo "🐳 [2/8] Installing Docker Container Engine..."
sudo yum install -y docker
sudo systemctl enable --now docker
sudo usermod -aG docker ec2-user

echo "✓ Docker enabled and ec2-user added to docker group."

# 3. Install Caddy Web Server (with automated Let's Encrypt HTTPS)
echo "🔒 [3/8] Installing Caddy Web Server for Automatic TLS..."
cd /tmp
curl -sL -o caddy 'https://caddyserver.com/api/download?os=linux&arch=amd64'
chmod +x caddy
sudo mv caddy /usr/bin/caddy
sudo mkdir -p /etc/caddy

if ! id "caddy" &>/dev/null; then
    sudo groupadd --system caddy
    sudo useradd --system \
        --gid caddy \
        --create-home \
        --home-dir /var/lib/caddy \
        --shell /usr/sbin/nologin \
        --comment "Caddy web server" \
        caddy
fi
sudo chown -R caddy:caddy /etc/caddy
echo "✓ Caddy $(caddy version) installed."

# 4. Clone or Pull GoLab Repository
echo "📥 [4/8] Setting up GoLab codebase..."
cd /home/ec2-user
if [ ! -d "/home/ec2-user/golab" ]; then
    echo "Cloning repository..."
    git clone https://github.com/DhawalShankar/golab.git
fi
cd /home/ec2-user/golab
git checkout main
git pull origin main

# 5. Build Go Backend Application
echo "🔨 [5/8] Compiling backend Go binary..."
cd /home/ec2-user/golab/backend
go mod download
go build -o app .
echo "✓ Binary compiled: /home/ec2-user/golab/backend/app"

# 6. Build Docker Compiler & Runtime Sandboxes
echo "📦 [6/8] Building Docker Sandbox Images..."
docker build -t golab-compiler:latest -f sandbox/compiler.Dockerfile .
docker build -t golab-runtime:latest -f sandbox/runtime.Dockerfile .

docker volume inspect golab-go-cache >/dev/null 2>&1 || \
    docker volume create golab-go-cache

docker run --rm \
    -u 0 \
    -v golab-go-cache:/gocache \
    alpine:3.22 \
    sh -c "chown -R 10001:10001 /gocache"

echo "✓ Docker sandbox images and volume cache initialized."

# 7. Configure and Enable Systemd Services
echo "⚙️ [7/8] Configuring systemd services..."

# Install golab.service
sudo cp /home/ec2-user/golab/ec2-deployment/golab.service /etc/systemd/system/golab.service
# Install Caddyfile
sudo cp /home/ec2-user/golab/ec2-deployment/Caddyfile /etc/caddy/Caddyfile
sudo chown -R caddy:caddy /etc/caddy
# Install caddy.service
sudo cp /home/ec2-user/golab/ec2-deployment/caddy.service /etc/systemd/system/caddy.service

sudo systemctl daemon-reload
sudo systemctl enable --now golab
sudo systemctl enable --now caddy

# 8. Verification and Health Check
echo "🔍 [8/8] Verifying live services..."
sleep 2

sudo systemctl is-active --quiet golab && echo "✓ golab.service: ACTIVE (Running on :8080)" || echo "⚠ golab.service failed to start"
sudo systemctl is-active --quiet caddy && echo "✓ caddy.service: ACTIVE (Listening on :80 and :443)" || echo "⚠ caddy.service failed to start"

echo "=================================================================="
echo "🎉 GOLAB SERVER BOOTSTRAP COMPLETE!"
echo "=================================================================="
echo "Health Check Test:"
echo "  curl http://localhost:8080/health"
echo "Public Domain:"
echo "  Ensure api.golangforall.in A-Record points to this EC2 Elastic IP!"
echo "=================================================================="
