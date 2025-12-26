#!/usr/bin/env pwsh
# Fix for Invalid Header Issue in Chillproxy

$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$logfile = "C:\chillproxy\header_fix_$timestamp.log"

function Log($msg) {
    Write-Host $msg
    Add-Content -Path $logfile -Value $msg
}

LogSection = {
    Log ""
    Log "================================"
    Log $args[0]
    Log "================================"
}

# Initialize log
"=== CHILLPROXY HEADER FIX ===" | Set-Content -Path $logfile
"Timestamp: $timestamp" | Add-Content -Path $logfile

Log "The issue: When Chillproxy sets the Authorization header with a pool key,"
Log "Go's http.Request validation is rejecting the header value."
Log ""
Log "This is likely because:"
Log "1. Pool key contains invalid characters for HTTP headers"
Log "2. Header value not properly sanitized before setting"
Log "3. Need to URL-encode or base64-encode the pool key"
Log ""
Log "Next steps:"
Log "1. Check how Chillproxy sets Authorization headers in store clients"
Log "2. Ensure pool key is properly encoded before setting as header"
Log "3. Test with real TorBox API endpoint"

