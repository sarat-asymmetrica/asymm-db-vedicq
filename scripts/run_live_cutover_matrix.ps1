param(
    [string]$RepoRoot = ".",
    [string]$EvidenceDir = "evidence",
    [string]$MigrationsDir = "db/migrations"
)

$ErrorActionPreference = "Stop"

function Require-Env([string]$name) {
    $v = [Environment]::GetEnvironmentVariable($name)
    if ([string]::IsNullOrWhiteSpace($v)) {
        throw "Missing required environment variable: $name"
    }
}

Require-Env "DATABASE_URL"

$started = Get-Date
$timestamp = $started.ToString("yyyyMMdd_HHmmss")
$outDir = Join-Path $RepoRoot $EvidenceDir
New-Item -ItemType Directory -Path $outDir -Force | Out-Null
$logPath = Join-Path $outDir "cutover_matrix_$timestamp.log"

"[$($started.ToString('yyyy-MM-dd HH:mm:ss zzz'))] cutover matrix start" | Tee-Object -FilePath $logPath

Push-Location $RepoRoot
try {
    "[$((Get-Date).ToString('yyyy-MM-dd HH:mm:ss zzz'))] validate migrations" | Tee-Object -FilePath $logPath -Append
    go run ./cmd/dbctl migrate validate --dir $MigrationsDir 2>&1 | Tee-Object -FilePath $logPath -Append

    "[$((Get-Date).ToString('yyyy-MM-dd HH:mm:ss zzz'))] apply migrations" | Tee-Object -FilePath $logPath -Append
    go run ./cmd/dbctl migrate up --dir $MigrationsDir 2>&1 | Tee-Object -FilePath $logPath -Append

    "[$((Get-Date).ToString('yyyy-MM-dd HH:mm:ss zzz'))] runtime selfcheck" | Tee-Object -FilePath $logPath -Append
    go run ./cmd/platform_runtime selfcheck 2>&1 | Tee-Object -FilePath $logPath -Append

    "[$((Get-Date).ToString('yyyy-MM-dd HH:mm:ss zzz'))] integration matrix" | Tee-Object -FilePath $logPath -Append
    $env:INTEGRATION_DATABASE_URL = $env:DATABASE_URL
    go test ./integration -v 2>&1 | Tee-Object -FilePath $logPath -Append
}
finally {
    Pop-Location
}

$ended = Get-Date
$elapsed = New-TimeSpan -Start $started -End $ended
"[$($ended.ToString('yyyy-MM-dd HH:mm:ss zzz'))] cutover matrix end elapsed=$('{0:hh\:mm\:ss}' -f $elapsed)" | Tee-Object -FilePath $logPath -Append
"evidence_log=$logPath"
