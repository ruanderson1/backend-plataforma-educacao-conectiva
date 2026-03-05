$ErrorActionPreference = "Stop"

# Ativa validações estritas para evitar erros silenciosos no script.
Set-StrictMode -Version Latest

# Garante execução a partir da raiz do repositório.
$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

# Lista explícita dos arquivos de teste da suíte principal.
$testFiles = @(
    "./tests/auth/service_integration_test.go",
    "./tests/database/mongo_test.go",
    "./tests/handlers/auth_handler_test.go"
)

Write-Host "Running 3 test files..." -ForegroundColor Cyan

# Executa os testes um a um para destacar rapidamente falhas por arquivo.
foreach ($testFile in $testFiles) {
    Write-Host "`n-> $testFile" -ForegroundColor Yellow
    go test -count=1 $testFile
    if ($LASTEXITCODE -ne 0) {
        Write-Host "`nFAILED: $testFile" -ForegroundColor Red
        exit $LASTEXITCODE
    }
}

Write-Host "`nAll test files passed." -ForegroundColor Green
exit 0
