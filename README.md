# SGEInventory_BE

Backend Sistem Manajemen Barang (Go + Fiber v3 + MariaDB).

## Menjalankan (development)

Hanya butuh Docker; tidak perlu Traefik atau network eksternal.

```bash
docker compose -f compose.dev.yaml up --build
```

| Layanan | URL / alamat |
|---|---|
| API | http://localhost:3000 |
| phpMyAdmin | http://localhost:8080 (login otomatis) |
| MariaDB | `localhost:3307` (user `db_admin`, password `db_admin`, db `sgeinventorydb_main`) |

Untuk mengubah port, salin `.env.example` menjadi `.env` lalu ubah nilainya.

Kode di-mount ke container dan di-reload otomatis oleh [air](https://github.com/air-verse/air).

## Reset database

Script di `docker/dev/mariadb/*.sql` (skema) hanya otomatis dijalankan saat volume database masih kosong:

```bash
docker compose -f compose.dev.yaml down -v
docker compose -f compose.dev.yaml up --build
```

Desain database: [docs/database-design.md](docs/database-design.md).

## Produksi

```bash
docker compose -f compose.prod.yaml up --build -d
```

Ganti password `CHANGE_ME` di `compose.prod.yaml` terlebih dahulu. API terbuka di port `3000` tanpa HTTPS,
jadi pasang reverse proxy/Cloudflare di depannya. phpMyAdmin hanya bisa diakses dari server itu sendiri
(`127.0.0.1:8080`, gunakan SSH tunnel) dan port database tidak dibuka ke host.
