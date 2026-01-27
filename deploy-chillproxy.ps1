#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Deploy Chillproxy to production server
.DESCRIPTION
    This script commits and pushes local changes, then deploys to the production server by:
    1. SSHing to chl server
    2. Pulling latest changes
    3. Building Docker image
    4. Restarting the container
.EXAMPLE
    .\deploy-chillproxy.ps1
    .\deploy-chillproxy.ps1 -SkipCommit
#>

param(
    [switch]$SkipCommit = $false,
    [string]$CommitMessage = ""
)

$ErrorActionPreference = "Stop"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Chillproxy Production Deployment" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if we're in the chillproxy directory
$currentDir = Get-Location
if (-not (Test-Path ".\internal\device\tracker.go")) {
    Write-Host "Error: Must be run from chillproxy repository root" -ForegroundColor Red
    exit 1
}

# Step 1: Commit and push local changes (unless skipped)
if (-not $SkipCommit) {
    Write-Host "[1/5] Checking for local changes..." -ForegroundColor Yellow

    $status = git status --porcelain
    if ($status) {
        Write-Host "  Found uncommitted changes:" -ForegroundColor Cyan
        git status --short
        Write-Host ""

        # Get commit message if not provided
        if (-not $CommitMessage) {
            $CommitMessage = Read-Host "  Enter commit message (or press Enter to skip commit)"
            if (-not $CommitMessage) {
                Write-Host "  Skipping commit - deploying existing code only" -ForegroundColor Yellow
                Write-Host ""
            }
        }

        if ($CommitMessage) {
            Write-Host "  Committing changes..." -ForegroundColor Cyan
            git add .
            git commit -m "$CommitMessage`n`nCo-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"

            Write-Host "  Pushing to origin..." -ForegroundColor Cyan
            git push
            Write-Host "  ✓ Changes committed and pushed" -ForegroundColor Green
            Write-Host ""
        }
    } else {
        Write-Host "  ✓ No uncommitted changes" -ForegroundColor Green
        Write-Host ""
    }
} else {
    Write-Host "[1/5] Skipping commit (--SkipCommit flag)" -ForegroundColor Yellow
    Write-Host ""
}

# Step 2: SSH to production and deploy
Write-Host "[2/5] Connecting to production server (chl)..." -ForegroundColor Yellow

$deployScript = @'
set -e

echo ""
echo "[3/5] Pulling latest changes..."
cd ~/chillstreams-app/chillproxy

# Fetch and pull latest changes
git fetch
git pull

echo ""
echo "[4/5] Building Docker image..."
docker build -t chillproxy:latest .

# Tag with GHCR name so docker-compose uses local build
docker tag chillproxy:latest ghcr.io/lumberbeard/chillproxy:latest

echo ""
echo "[5/5] Recreating container with new image..."
cd ~/chillstreams-app

# Recreate container using docker-compose (will use locally tagged image)
docker-compose -f docker-compose.prod.yml up -d --force-recreate --no-deps chillproxy

echo ""
echo "✓ Deployment complete!"
echo ""
echo "Checking container status..."
docker ps | grep chillstreams-chillproxy

echo ""
echo "Recent logs:"
docker logs --tail 30 chillstreams-chillproxy
'@

ssh chl "bash -c '$deployScript'"

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "  ✓ Deployment Successful!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Cyan
Write-Host "  - Monitor logs: ssh chl 'docker logs -f chillstreams-chillproxy'" -ForegroundColor Gray
Write-Host "  - Check status: ssh chl 'docker ps | grep chillproxy'" -ForegroundColor Gray
Write-Host ""
