# Menjalankan gateway dengan penyimpanan ke database lokal.
# Password diambil dari file .env, jadi tidak pernah tertulis di Git.
# Pemakaian (dari folder fleet-platform):
#   powershell -ExecutionPolicy Bypass -File scripts\run-gateway.ps1
$pw = (Get-Content "$PSScriptRoot\..\.env" | Where-Object { $_ -like "POSTGRES_PASSWORD=*" }) -replace "^POSTGRES_PASSWORD=", ""
$env:DATABASE_URL = "host=localhost port=5432 user=fleet password=$pw dbname=fleet sslmode=disable"
Set-Location "$PSScriptRoot\.."
go run ./cmd/gateway