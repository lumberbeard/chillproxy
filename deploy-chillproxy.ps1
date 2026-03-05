#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Deploy Chillproxy to production server
.DESCRIPTION
    Run locally from your machine. This script:
    1. Optionally commits and pushes local chillproxy changes
    2. SSHes to the production server (chl)
    3. Pulls latest code from GitHub
    4. Builds Docker image on the server
    5. Recreates the chillproxy container
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

# Navigate to the chillproxy directory (script may be invoked from chillstreams repo)
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $scriptDir
Write-Host "  Working directory: $scriptDir" -ForegroundColor Gray

if (-not (Test-Path ".\go.mod")) {
    Write-Host "Error: Not in chillproxy repository root (go.mod not found)" -ForegroundColor Red
    exit 1
}

# Step 1: Commit and push local changes (unless skipped)
if (-not $SkipCommit) {
    Write-Host "[1/6] Checking for local changes..." -ForegroundColor Yellow

    $status = git status --porcelain
    if ($status) {
        Write-Host "  Found uncommitted changes:" -ForegroundColor Cyan
        git status --short
        Write-Host ""

        if (-not $CommitMessage) {
            $CommitMessage = Read-Host "  Enter commit message (or press Enter to skip commit)"
            if (-not $CommitMessage) {
                Write-Host "  Skipping commit - deploying existing code only" -ForegroundColor Yellow
                Write-Host ""
            }
        }

        if ($CommitMessage) {
            Write-Host "  Committing changes..." -ForegroundColor Cyan
            git add -A
            git commit -m "$CommitMessage`n`nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"

            Write-Host "  Pushing to origin..." -ForegroundColor Cyan
            git push
            Write-Host "  Changes committed and pushed" -ForegroundColor Green
            Write-Host ""
        }
    } else {
        Write-Host "  No uncommitted changes" -ForegroundColor Green
    }

    # Always ensure local commits are pushed
    $unpushed = git log --oneline '@{u}..HEAD' 2>$null
    if ($unpushed) {
        Write-Host "  Pushing unpushed commits..." -ForegroundColor Cyan
        git push
        Write-Host "  Pushed" -ForegroundColor Green
    }
    Write-Host ""
} else {
    Write-Host "[1/6] Skipping commit (--SkipCommit flag)" -ForegroundColor Yellow
    Write-Host ""
}

# Step 2-5: SSH to production and deploy
Write-Host "[2/6] Connecting to production server..." -ForegroundColor Yellow

# Build the remote script as a string, strip Windows \r line endings before piping to bash
$remoteScript = @'
set -e

echo ""
echo "[3/6] Pulling latest chillproxy code..."
cd ~/chillstreams-app/chillproxy
git fetch origin
git reset --hard origin/main
echo "  Commit: $(git log --oneline -1)"

echo ""
echo "[4/6] Building Docker image (no cache)..."
docker build --no-cache -t ghcr.io/lumberbeard/chillproxy:latest . 2>&1

echo ""
echo "[5/6] Pushing image to GHCR (so PROD-DEPLOY won't overwrite with stale image)..."
docker push ghcr.io/lumberbeard/chillproxy:latest 2>&1

echo ""
echo "[6/6] Recreating chillproxy container..."
cd ~/chillstreams-app
docker compose -f docker-compose.prod.yml up -d --force-recreate --no-deps chillproxy

echo ""
echo "=== Deployment complete ==="
echo ""
sleep 3
echo "Container status:"
docker ps --filter name=chillproxy --format "table {{.Names}}\t{{.Status}}\t{{.CreatedAt}}"
echo ""
echo "Image build time:"
docker image inspect ghcr.io/lumberbeard/chillproxy:latest --format "{{.Created}}"
echo ""
echo "Recent logs:"
docker logs --tail 15 chillstreams-chillproxy
'@

# Strip \r (Windows CRLF) so bash on Linux doesn't choke
$remoteScript = $remoteScript -replace "`r", ""
$remoteScript | ssh chl "bash -s"

if ($LASTEXITCODE -ne 0) {
    Write-Host ""
    Write-Host "  Deployment FAILED (exit code $LASTEXITCODE)" -ForegroundColor Red
    exit $LASTEXITCODE
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "  Deployment Successful!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "Monitor: ssh chl 'docker logs -f chillstreams-chillproxy'" -ForegroundColor Gray
Write-Host ""
