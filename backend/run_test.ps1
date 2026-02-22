$ErrorActionPreference = "Stop"
try {
    go test ./internal/generator -v 2>&1 | Tee-Object -FilePath "compile_out.log"
} catch {
    Write-Host "Caught error"
}
