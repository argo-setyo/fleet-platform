# Menjalankan goose ke database lokal, dengan password dari file .env
# Pemakaian (dari folder fleet-platform):
#   powershell -ExecutionPolicy Bypass -File scripts\migrate.ps1 status
#   powershell -ExecutionPolicy Bypass -File scripts\migrate.ps1 up
$pw = (Get-Content "$PSScriptRoot\..\.env" | Where-Object { $_ -like "POSTGRES_PASSWORD=*" }) -replace "^POSTGRES_PASSWORD=", ""
$env:GOOSE_DRIVER = "postgres"
$env:GOOSE_DBSTRING = "host=localhost port=5432 user=fleet password=$pw dbname=fleet sslmode=disable"
$env:GOOSE_MIGRATION_DIR = "$PSScriptRoot\..\migrations"
goose @args