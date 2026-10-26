# Cara menjalankan

## Kebutuhan
- Go (cek dengan `go version`)
- Git

## Menjalankan gateway
    go run ./cmd/gateway

Server mendengarkan di port 38000.

## Menjalankan tes
    go test -v ./cmd/gateway

## Menghubungkan perangkat
Atur server perangkat ke IP komputer (lihat dengan `ipconfig`) dan port 38000.
Perangkat dan komputer harus berada di jaringan yang sama.