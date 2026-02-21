param(
    [string]$MigrationsPath = "db/migrations"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path -Path $MigrationsPath -PathType Container)) {
    Write-Error "Migrations path not found: $MigrationsPath"
    exit 1
}

$files = Get-ChildItem -Path $MigrationsPath -File -Filter "*.sql" | Sort-Object Name
if ($files.Count -eq 0) {
    Write-Error "No migration files found in: $MigrationsPath"
    exit 1
}

$namePattern = '^(?<seq>\d{4})_.+\.sql$'
$seqs = @()
$hasError = $false

foreach ($file in $files) {
    if ($file.Name -notmatch $namePattern) {
        Write-Error "Invalid migration filename format: $($file.Name)"
        $hasError = $true
        continue
    }

    $seq = [int]$Matches['seq']
    $seqs += $seq

    $raw = Get-Content -Path $file.FullName -Raw
    if ($raw -notmatch '(?im)^\s*BEGIN\s*;') {
        Write-Error "Missing BEGIN; guard in $($file.Name)"
        $hasError = $true
    }
    if ($raw -notmatch '(?im)^\s*COMMIT\s*;') {
        Write-Error "Missing COMMIT; guard in $($file.Name)"
        $hasError = $true
    }
}

$uniqueSeqs = $seqs | Sort-Object -Unique
if ($uniqueSeqs.Count -ne $seqs.Count) {
    Write-Error "Duplicate migration sequence detected."
    $hasError = $true
}

$expected = 1
foreach ($s in ($uniqueSeqs | Sort-Object)) {
    if ($s -ne $expected) {
        Write-Error ("Sequence gap or mismatch. Expected {0:0000}, found {1:0000}" -f $expected, $s)
        $hasError = $true
    }
    $expected++
}

if ($hasError) {
    exit 1
}

Write-Output ("Migration validation passed. Files={0}, Last={1:0000}" -f $files.Count, ($uniqueSeqs | Measure-Object -Maximum).Maximum)
exit 0
