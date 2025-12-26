#!/usr/bin/env pwsh
# Check the actual pool key being used

Write-Host "Checking pool keys in database..." -ForegroundColor Cyan

# Get the pool key values
$poolKeys = docker exec chillstreams-postgres psql -U postgres -d chillstreams -t -c "SELECT id, api_key, LENGTH(api_key) as key_length FROM torbox_pool LIMIT 5;" 2>&1
Write-Host "Pool Keys:"
Write-Host $poolKeys

Write-Host ""
Write-Host "Checking for whitespace/special characters..." -ForegroundColor Cyan

# Get pool key with hex encoding to see any hidden characters
$poolKeysHex = docker exec chillstreams-postgres psql -U postgres -d chillstreams -t -c "SELECT id, api_key, ENCODE(api_key::bytea, 'hex') as hex_encoded FROM torbox_pool LIMIT 5;" 2>&1
Write-Host "Pool Keys (Hex):"
Write-Host $poolKeysHex

Write-Host ""
Write-Host "Getting assignment details..." -ForegroundColor Cyan

# Check the actual assignment
$assign = docker exec chillstreams-postgres psql -U postgres -d chillstreams -c "SELECT ta.user_id, ta.assigned_pool_key_id, tp.api_key, LENGTH(tp.api_key) as key_len FROM torbox_assignments ta LEFT JOIN torbox_pool tp ON ta.assigned_pool_key_id = tp.id LIMIT 1;" 2>&1
Write-Host $assign

Write-Host ""
Write-Host "Checking if api_key field type is correct..." -ForegroundColor Cyan

$schema = docker exec chillstreams-postgres psql -U postgres -d chillstreams -c "\d torbox_pool" 2>&1 | Select-String -Pattern "api_key"
Write-Host $schema

