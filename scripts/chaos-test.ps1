# Exercise 5 (stretch): chaos test. Proves that no paid order is lost when
# the broker or the workers crash in the middle of the flow.
#
#   Phase 1: RabbitMQ is killed, orders are paid while it is down.
#            -> webhooks still succeed, events wait in the outbox table.
#   Phase 2: both workers are killed while they are processing.
#            -> unacked messages are redelivered after the restart.
#   Check:   every order is paid, published, emailed and counted exactly once.
#
# Usage (from the project folder, with docker compose up and running):
#   powershell -ExecutionPolicy Bypass -File scripts\chaos-test.ps1
#   powershell -ExecutionPolicy Bypass -File scripts\chaos-test.ps1 -Orders 20

param(
    [int]$Orders = 10,
    [int]$BrokerDownSeconds = 10,
    [int]$TimeoutSeconds = 120
)

$ErrorActionPreference = "Continue"
if ($Orders -lt 2) { throw "-Orders must be at least 2 (half per phase)" }
Set-Location (Split-Path $PSScriptRoot -Parent)

function Step([string]$text) { Write-Host "`n=== $text ===" -ForegroundColor Cyan }

# Runs SQL in the db container and returns the result rows (no headers).
function Sql([string]$query) {
    $out = $query | docker compose exec -T db psql -U postgres -d checkout -t -A -q -v ON_ERROR_STOP=1
    if ($LASTEXITCODE -ne 0) { throw "SQL failed: $query" }
    # The leading comma stops PowerShell from unwrapping a one-row result
    # into a plain string (then [0] would be its first character).
    return , @($out | Where-Object { $_ -ne "" })
}

# Sends a signed webhook for every order id; returns how many got HTTP 204.
function Pay([int[]]$ids) {
    $ok = 0
    foreach ($id in $ids) {
        $reply = & $webhook -order $id 2>&1 | Out-String
        if ($reply -match "HTTP 204") { $ok++ } else { Write-Host "  order $id -> $($reply.Trim())" -ForegroundColor Red }
    }
    return $ok
}

function Wait-Healthy([string]$container) {
    for ($i = 0; $i -lt 60; $i++) {
        $status = docker inspect -f "{{.State.Health.Status}}" $container 2>$null
        if ($status -eq "healthy") { return }
        Start-Sleep -Seconds 1
    }
    throw "$container did not become healthy"
}

# ---------------------------------------------------------------- setup
Step "Setup"
$webhook = Join-Path $env:TEMP "fake-webhook.exe"
go build -o $webhook ./cmd/fake-webhook
if ($LASTEXITCODE -ne 0) { throw "could not build fake-webhook" }

$env:SIMULATE_EMAIL_FAILURE = "false"
docker compose up -d
Wait-Healthy "farida-checkout-rabbitmq"

$salesBefore = [int]((Sql "SELECT COALESCE((SELECT orders FROM daily_sales WHERE day = CURRENT_DATE), 0);")[0])

# N new pending orders with one item each. Prefer items without stock
# limits (virtual items / currency), so the test does not run out of stock.
$ids = Sql @"
WITH it AS (
    SELECT id, price FROM items
    ORDER BY (item_type = 'merch'), stock DESC, id
    LIMIT 1
),
u AS (SELECT id FROM users ORDER BY id LIMIT 1),
o AS (
    INSERT INTO orders (user_id, total, status)
    SELECT u.id, it.price, 'pending' FROM u, it, generate_series(1, $Orders)
    RETURNING id
),
li AS (
    INSERT INTO line_items (order_id, item_id, price, quantity)
    SELECT o.id, it.id, it.price, 1 FROM o, it
    RETURNING order_id
)
SELECT id FROM o ORDER BY id;
"@
$ids = @($ids | ForEach-Object { [int]$_ })
$idList = $ids -join ","
$half = [math]::Floor($ids.Count / 2)
$phase1 = $ids[0..($half - 1)]
$phase2 = $ids[$half..($ids.Count - 1)]
Write-Host "created $($ids.Count) pending orders: $idList"
Write-Host "daily_sales today before the test: $salesBefore orders"

# Counts for our orders: unsent outbox rows | emails done | analytics done
$progressSql = @"
WITH ids AS (SELECT 'order.paid:' || x AS mid FROM unnest(ARRAY[$idList]) x)
SELECT
  (SELECT count(*) FROM outbox WHERE sent_at IS NULL AND message_id IN (SELECT mid FROM ids)),
  (SELECT count(*) FROM processed_messages WHERE consumer = 'email' AND processed_at IS NOT NULL AND message_id IN (SELECT mid FROM ids)),
  (SELECT count(*) FROM processed_messages WHERE consumer = 'analytics' AND processed_at IS NOT NULL AND message_id IN (SELECT mid FROM ids));
"@

# ---------------------------------------------------------------- phase 1
Step "Phase 1: kill RabbitMQ, pay $($phase1.Count) orders while it is down"
docker compose kill rabbitmq
$ok1 = Pay $phase1
Write-Host "webhooks answered 204: $ok1 / $($phase1.Count)  (the API does not need the broker)"

$p = (Sql $progressSql)[0].Split("|")
Write-Host "events waiting in the outbox (sent_at IS NULL): $($p[0])  <- not lost, just not sent yet"

Write-Host "broker stays down for $BrokerDownSeconds s..."
Start-Sleep -Seconds $BrokerDownSeconds

docker compose start rabbitmq
Wait-Healthy "farida-checkout-rabbitmq"
Write-Host "RabbitMQ is back. The relay reconnects and publishes the outbox rows."
# The workers crashed when the broker went away; restart: unless-stopped
# brings them back, this makes sure they are running.
docker compose up -d notification-worker analytics-worker

# ---------------------------------------------------------------- phase 2
Step "Phase 2: pay $($phase2.Count) orders, kill both workers mid-processing"
$ok2 = Pay $phase2
Write-Host "webhooks answered 204: $ok2 / $($phase2.Count)"

# Relay polls every 1 s and an email takes 2 s: after ~2 s a worker is in the
# middle of a send, holding an unacked message.
Start-Sleep -Seconds 2
docker compose kill notification-worker analytics-worker
Write-Host "workers killed (SIGKILL: no clean shutdown, unacked messages go back to the queue)"
Start-Sleep -Seconds 3
docker compose start notification-worker analytics-worker

# ---------------------------------------------------------------- wait
Step "Waiting until every order is published, emailed and counted"
$deadline = (Get-Date).AddSeconds($TimeoutSeconds)
do {
    $p = (Sql $progressSql)[0].Split("|")
    Write-Host ("  outbox unsent: {0}   emailed: {1}/{3}   counted: {2}/{3}" -f $p[0], $p[1], $p[2], $ids.Count)
    $done = ([int]$p[0] -eq 0) -and ([int]$p[1] -eq $ids.Count) -and ([int]$p[2] -eq $ids.Count)
    if (-not $done) { Start-Sleep -Seconds 3 }
} while (-not $done -and (Get-Date) -lt $deadline)

# ---------------------------------------------------------------- verify
Step "Results"
$paid = [int]((Sql "SELECT count(*) FROM orders WHERE id = ANY(ARRAY[$idList]) AND status = 'paid';")[0])
$events = [int]((Sql "SELECT count(*) FROM outbox WHERE sent_at IS NOT NULL AND message_id IN (SELECT 'order.paid:' || x FROM unnest(ARRAY[$idList]) x);")[0])
$salesAfter = [int]((Sql "SELECT COALESCE((SELECT orders FROM daily_sales WHERE day = CURRENT_DATE), 0);")[0])
$counted = $salesAfter - $salesBefore

# Email log lines per order: more than one = a duplicate email. Possible in
# theory (at-least-once), when a worker dies after sending but before
# recording it; processed_messages keeps this window very small.
$logs = docker compose logs --no-log-prefix notification-worker 2>$null | Out-String
$dupes = @()
foreach ($id in $ids) {
    $n = ([regex]::Matches($logs, "\[EMAIL\] order #$id confirmed")).Count
    if ($n -gt 1) { $dupes += "#$id x$n" }
}

$n = $ids.Count
$checks = @(
    @{ Name = "orders paid";                    Got = $paid;    Want = $n },
    @{ Name = "events published (outbox sent)"; Got = $events;  Want = $n },
    @{ Name = "emails processed";               Got = [int]$p[1]; Want = $n },
    @{ Name = "analytics: counted exactly once"; Got = $counted; Want = $n }
)
$failed = 0
foreach ($c in $checks) {
    $pass = $c.Got -eq $c.Want
    if (-not $pass) { $failed++ }
    $color = if ($pass) { "Green" } else { "Red" }
    $mark = if ($pass) { "PASS" } else { "FAIL" }
    Write-Host ("  [{0}] {1,-34} {2}/{3}" -f $mark, $c.Name, $c.Got, $c.Want) -ForegroundColor $color
}
if ($dupes.Count -gt 0) {
    Write-Host "  [INFO] duplicate emails (at-least-once): $($dupes -join ', ')" -ForegroundColor Yellow
} else {
    Write-Host "  [INFO] no duplicate emails" -ForegroundColor Green
}

if ($failed -eq 0) {
    Write-Host "`nCHAOS TEST PASSED: no order lost, none counted twice." -ForegroundColor Green
    exit 0
}
Write-Host "`nCHAOS TEST FAILED: $failed check(s). See docker compose logs." -ForegroundColor Red
exit 1
