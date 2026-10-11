# Chaos experiment: runs loadtest/chaos.js and breaks the database on a
# schedule that matches the phases in chaos.js.
#
# Run from checkout-api, with `docker compose up -d` already running:
#   .\loadtest\chaos.ps1

$dir = (Get-Location).Path

$injector = Start-Job -ArgumentList $dir -ScriptBlock {
    param($dir)
    Set-Location $dir
    $t0 = Get-Date
    function At($sec, $label, [scriptblock]$action) {
        $wait = $sec - ((Get-Date) - $t0).TotalSeconds
        if ($wait -gt 0) { Start-Sleep -Milliseconds ([int]($wait * 1000)) }
        & $action 2>&1 | Out-Null
        "{0,6:N1}s  {1}" -f ((Get-Date) - $t0).TotalSeconds, $label
    }
    # Line up with k6's clock: wait for the k6 process, start from its start time.
    $k6 = $null
    while (-not $k6) { Start-Sleep -Milliseconds 200; $k6 = Get-Process k6 -ErrorAction SilentlyContinue | Select-Object -First 1 }
    $t0 = $k6.StartTime.AddSeconds(1)   # ~1 s for k6 to load the script and run setup()
    At 60  'docker compose pause db    (DB hangs)'      { docker compose pause db }
    At 105 'docker compose unpause db  (recovery 1)'    { docker compose unpause db }
    At 165 'docker compose stop db     (DB down)'       { docker compose stop db }
    At 210 'docker compose start db    (recovery 2)'    { docker compose start db }
}

try {
    k6 run loadtest/chaos.js
}
finally {
    Write-Host "`nFault injection log (seconds since test start):"
    Receive-Job $injector -Wait -AutoRemoveJob
    # Never leave the database broken, even if the test was cancelled.
    docker compose unpause db 2>$null | Out-Null
    docker compose start db 2>$null | Out-Null
    Write-Host "Database restored."
}
