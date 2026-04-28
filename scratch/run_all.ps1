$env:SEED_DB="true"
Start-Process -FilePath "go" -ArgumentList "run", "cmd/server/main.go" -RedirectStandardOutput "server_log.txt" -RedirectStandardError "server_err.txt" -NoNewWindow -PassThru

Write-Host "Waiting for database seeding to complete..."
$completed = $false
while (-not $completed) {
    Start-Sleep -Seconds 5
    if (Test-Path "server_log.txt") {
        $logContent = Get-Content "server_log.txt" -Raw
        if ($logContent -match "Full database seeding completed successfully.") {
            $completed = $true
            Write-Host "Seeding completed."
        }
    }
}

Write-Host "Running E2E tests..."
go run scratch/run_e2e_tests.go
