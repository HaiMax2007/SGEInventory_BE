# SGEInventory_BE

Backend Sistem Manajemen Barang (Go + Fiber v3 + PostgreSQL).

## Menjalankan (development)

Hanya butuh Docker; tidak perlu Traefik atau network eksternal.

```bash
docker compose -f compose.dev.yaml up --build
```

| Layanan | URL / alamat |
|---|---|
| API | http://localhost:3000 |
| pgAdmin | http://localhost:8081 (tanpa login; saat membuka server pertama kali, isi password `db_admin` lalu centang "Save password") |
| PostgreSQL | `localhost:5433` (user `db_admin`, password `db_admin`, db `sgeinventorydb_main`) |

Untuk mengubah port, salin `.env.example` menjadi `.env` lalu ubah nilainya.

Kode di-mount ke container dan di-reload otomatis oleh [air](https://github.com/air-verse/air).

Konfigurasi dibaca dari environment (`internal/config/config.go`): `PORT`, `DB_HOST`, `DB_PORT`, `DB_USER`,
`DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` (default `disable`), `JWT_SECRET`, `CORS_ORIGINS`. Compose sudah mengisinya; jika menjalankan
`go run ./cmd/app` di luar Docker, isi sendiri (mis. `DB_HOST=localhost DB_PORT=5433`).

```bash
go build ./...     # cek semua kode bisa di-compile
go vet ./...       # cek kesalahan umum
go test ./...      # jalankan test (folder tests/unit, masih kosong)
```

## Struktur kode

Mengikuti pola Clean Architecture (handler → service → repository).

```
cmd/app/main.go                 titik awal: baca config → sambung DB → rakit repository/service/handler → routes
cmd/migrate, cmd/seed           (kosong) untuk program migrasi & seeder nanti
domain/                         inti aplikasi, tidak tahu soal HTTP
  entity/                       struct tabel database + aturan bisnisnya (User, Item)
  dto/                          bentuk data request/response JSON (+ isi token JWT)
  contracts/                    interface repository & service ("janji" tiap lapisan)
  enums/                        konstanta role & status barang
  errx/                         error bisnis + Kind-nya (400/401/403/404/409)
internal/
  app/<fitur>/                  satu folder per fitur: auth, user, item
    routes.go                   daftar URL → middleware → method handler
    handler.go                  baca request (params, body, query) → panggil service → balas JSON
    service.go                  aturan bisnis (hash password, cek stok, buat token, ...)
  config/config.go              membaca env
  infra/database/postgres.go    membuka koneksi GORM ke PostgreSQL
  infra/repository/             query GORM (satu file per tabel)
  middlewares/auth.go           Protected (cek token) & AdminOnly (cek role)
  middlewares/error_handler.go  mengubah semua error menjadi JSON + status HTTP
helpers/pagination/             hitung page/limit/offset/totalPages
pkg/response/                   membuat respons sukses {"message","data","meta"}
pkg/validator/                  mengecek tag `validate:"..."` pada DTO
tests/unit/                     (kosong) tempat unit test
```

Alur request, contoh `PUT /api/items/:id`:

1. `internal/app/item/routes.go` → middleware `Protected` (cek token) → `AdminOnly` (cek role)
2. `ItemHandler.Update` membaca `:id` & body, lalu memvalidasi DTO
3. `ItemService.UpdateItem` menjalankan aturan bisnis (`entity.Item.ChangeTotal`)
4. `ItemRepository.Update` menyimpan ke database lewat GORM
5. Handler membalas JSON lewat `pkg/response`

Jika ada error di langkah mana pun, kode cukup `return err`. `middlewares.ErrorHandler` yang mengubahnya
menjadi JSON: error dari `domain/errx` memakai status & pesannya sendiri, error lain dicatat di log dan dibalas
500 tanpa membocorkan detail.

Semua bagian dirakit di `cmd/app/main.go` (dependency injection): repository menerima `*gorm.DB`, service
menerima interface repository, handler menerima interface service. Karena service hanya mengenal interface,
repository bisa diganti versi palsu (mock) saat menulis unit test.

## Auth (JWT)

1. FE memanggil `POST /api/auth/login` dengan `{"email","password"}`.
2. Respons berisi `data.token` (berlaku 24 jam) dan `data.user`. FE menyimpan token di cookie bernama `token`
   (path `/`).
3. Setiap request yang butuh login mengirim header `Authorization: Bearer <token>`.
4. Logout: FE memanggil `POST /api/auth/logout` dengan header `Authorization`, lalu menghapus cookie `token`
   miliknya sendiri. BE tidak menyentuh cookie sama sekali.

```js
// contoh di FE
const res = await fetch("http://localhost:3000/api/auth/login", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ email, password }),
});
const { data } = await res.json();
document.cookie = `token=${data.token}; path=/; max-age=${data.expiresIn}`;

const token = document.cookie.split("; ").find((c) => c.startsWith("token="))?.split("=")[1];
const me = await fetch("http://localhost:3000/api/auth/me", {
  headers: { Authorization: `Bearer ${token}` },
});

// logout
await fetch("http://localhost:3000/api/auth/logout", {
  method: "POST",
  headers: { Authorization: `Bearer ${token}` },
});
document.cookie = "token=; path=/; max-age=0"; // FE menghapus cookie-nya sendiri
```

Logout tidak mematikan JWT di server. Token yang sempat tersalin masih berlaku sampai kedaluwarsa (24 jam);
begitu juga perubahan role atau penghapusan user baru berlaku setelah user login ulang.

| Status | Arti |
|---|---|
| 401 | Token tidak ada / salah / kedaluwarsa, atau email/password salah saat login → arahkan ke halaman login |
| 403 | Token valid tapi tidak punya izin (bukan admin / bukan data miliknya) |
| 429 | Login lebih dari 5x per menit dari IP yang sama |

Alamat FE yang boleh memanggil API diatur lewat env `CORS_ORIGINS` (default dev:
`http://localhost:5173,http://localhost:3001`). Jika FE jalan di port lain, isi `CORS_ORIGINS` di `.env` lalu
`docker compose -f compose.dev.yaml up -d`.

## API User

Format respons: `{"message": "...", "data": ...}`; list menambah `meta` (`page`, `limit`, `total`, `totalPages`).
Error: `{"message": "..."}`; untuk data tidak valid (400) ditambah `"error"` berisi detail validasi.

| Method | Path | Akses | Fungsi |
|---|---|---|---|
| POST | `/api/auth/login` | Semua | Login, membalas token |
| POST | `/api/auth/logout` | Login | Logout (FE lalu menghapus token-nya sendiri) |
| GET | `/api/auth/me` | Login | Data user yang sedang login |
| POST | `/api/users` | Semua | Registrasi user (role otomatis `USER`) |
| GET | `/api/users` | Admin | Daftar user. Query: `page`, `limit` (1–100, default 10), `search`, `role` |
| GET | `/api/users/:id` | Admin / pemilik | Detail user |
| PUT | `/api/users/:id` | Admin / pemilik | Update parsial: `fullName`, `email`, `password`, `contact` |
| DELETE | `/api/users/:id` | Admin / pemilik | Hapus user (soft delete) |

```bash
curl -X POST localhost:3000/api/users -H "Content-Type: application/json" \
  -d '{"fullName":"Budi Santoso","email":"budi@mail.com","password":"rahasia123","contact":"08123456789"}'

curl -X POST localhost:3000/api/auth/login -H "Content-Type: application/json" \
  -d '{"email":"budi@mail.com","password":"rahasia123"}'

curl localhost:3000/api/auth/me -H "Authorization: Bearer <token>"
```

User tidak terikat lembaga; lembaga dipilih per pengajuan peminjaman (`borrowings.lembaga_id`).
Data awal development hanya akun admin di akhir `docker/dev/postgres/001_schema.sql`; data lain tambahkan
sendiri lewat API atau pgAdmin.

Catatan:
- Admin tidak bisa dibuat lewat API. Daftar seperti biasa, lalu ubah lewat pgAdmin (Query Tool):
  `UPDATE users SET role='ADMIN' WHERE email='...'`, kemudian login ulang agar token memuat role baru.
- User yang dihapus tetap menempati email-nya (soft delete), jadi email itu tidak bisa didaftarkan ulang.

## API Barang

Semua endpoint butuh token. Melihat barang boleh semua user yang login; menambah, mengubah dan menghapus hanya admin.

| Method | Path | Akses | Fungsi |
|---|---|---|---|
| POST | `/api/items` | Admin | Tambah barang: `name`, `totalQuantity` |
| GET | `/api/items` | Login | Daftar barang. Query: `page`, `limit` (1–100, default 10), `search`, `available=true` |
| GET | `/api/items/:id` | Login | Detail barang |
| PUT | `/api/items/:id` | Admin | Update parsial: `name`, `totalQuantity` |
| DELETE | `/api/items/:id` | Admin | Hapus barang (soft delete) |

```bash
curl -X POST localhost:3000/api/items -H "Authorization: Bearer <token admin>" \
  -H "Content-Type: application/json" -d '{"name":"Proyektor Epson","totalQuantity":5}'
```

Contoh data barang:

```json
{"id":1,"name":"Proyektor Epson","totalQuantity":5,"availableQuantity":3,"status":"TERSEDIA", ...}
```

- `totalQuantity`: jumlah barang yang dimiliki (diatur admin).
- `availableQuantity`: sisa yang bisa dipinjam. Saat barang baru dibuat, nilainya sama dengan total; nanti
  berkurang saat pengajuan di-ACC dan bertambah saat barang dikembalikan.
- `status` tidak disimpan di database: `TERSEDIA` jika `availableQuantity > 0`, `HABIS` jika 0.
- Mengubah `totalQuantity` menggeser `availableQuantity` sebesar selisihnya. Total tidak boleh lebih kecil dari
  jumlah yang sedang dipinjam (409).
- Barang yang masih dipinjam tidak bisa dihapus (409).

## Reset database

Script di `docker/dev/postgres/*.sql` (skema) hanya otomatis dijalankan saat volume database masih kosong:

```bash
docker compose -f compose.dev.yaml down -v
docker compose -f compose.dev.yaml up --build
```

Desain database: [docs/database-design.md](docs/database-design.md).

## Produksi

```bash
docker compose -f compose.prod.yaml up --build -d
```

Ganti semua `CHANGE_ME` di `compose.prod.yaml` terlebih dahulu: password database (`POSTGRES_PASSWORD` dan
`DB_PASSWORD` harus sama), password pgAdmin, `JWT_SECRET` (string acak minimal 32 karakter, mis.
`openssl rand -hex 32`; API menolak jalan jika kurang) dan `CORS_ORIGINS` (domain FE). API terbuka di port
`3000` tanpa HTTPS, jadi pasang reverse proxy/Cloudflare di depannya. pgAdmin (login dengan
`PGADMIN_DEFAULT_EMAIL` + password) hanya bisa diakses dari server itu sendiri (`127.0.0.1:8081`, gunakan SSH
tunnel); tambahkan server di pgAdmin dengan host `db`. Port database tidak dibuka ke host.
