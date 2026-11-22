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
cmd/migrate, cmd/seed           (kosong) cadangan jika nanti migrasi/seed mau dijalankan sebagai program terpisah
domain/                         inti aplikasi, tidak tahu soal HTTP
  entity/                       struct tabel database + aturan bisnisnya (User, Lembaga, Item)
  dto/                          bentuk data request/response JSON (+ isi token JWT)
  contracts/                    interface repository & service ("janji" tiap lapisan)
  enums/                        konstanta role & status barang
  errx/                         error bisnis + Kind-nya (400/401/403/404/409)
internal/
  app/<fitur>/                  satu folder per fitur: auth, user, lembaga, item
    routes.go                   daftar URL → middleware → method handler
    handler.go                  baca request (params, body, query) → panggil service → balas JSON
    service.go                  aturan bisnis (hash password, cek stok, buat token, ...)
  config/config.go              membaca env
  infra/database/postgres.go    membuka koneksi GORM ke PostgreSQL
  infra/database/migrate.go     AutoMigrate: membuat/menyesuaikan tabel dari struct entity setiap app start
  infra/database/seed.go        data awal: user Max (ADMIN) & Raka (USER) + lembaga BEM, HMDTIF, DPM
                                (hanya saat APP_ENV=development)
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
| POST | `/api/users` | Semua | Registrasi user (role otomatis `USER`). Wajib: `fullName`, `nim` (8–20 digit angka, unik), `email`, `password` |
| GET | `/api/users` | Admin | Daftar user. Query: `page`, `limit` (1–100, default 10), `search`, `role` |
| GET | `/api/users/:id` | Admin / pemilik | Detail user |
| PUT | `/api/users/:id` | Admin / pemilik | Update parsial: `fullName`, `email`, `password`, `contact` |
| DELETE | `/api/users/:id` | Admin / pemilik | Hapus user (soft delete) |

```bash
curl -X POST localhost:3000/api/users -H "Content-Type: application/json" \
  -d '{"fullName":"Budi Santoso","nim":"2210511001","email":"budi@mail.com","password":"rahasia123","contact":"08123456789"}'

curl -X POST localhost:3000/api/auth/login -H "Content-Type: application/json" \
  -d '{"email":"budi@mail.com","password":"rahasia123"}'

curl localhost:3000/api/auth/me -H "Authorization: Bearer <token>"
```

User tidak terikat lembaga; lembaga dipilih per pengajuan peminjaman (`borrowings.lembaga_id`).
Data awal development (dibuat otomatis oleh `internal/infra/database/seed.go` saat `APP_ENV=development`):
Max (ADMIN, NIM `00000001`) dan Raka (USER, NIM `00000002`), serta lembaga BEM, HMDTIF dan DPM. Data lain
tambahkan sendiri lewat API atau pgAdmin.

Catatan:
- Admin tidak bisa dibuat lewat API. Daftar seperti biasa, lalu ubah lewat pgAdmin (Query Tool):
  `UPDATE users SET role='ADMIN' WHERE email='...'`, kemudian login ulang agar token memuat role baru.
- User yang dihapus tetap menempati email-nya (soft delete), jadi email itu tidak bisa didaftarkan ulang.

## API Lembaga

Semua endpoint butuh token. Melihat lembaga boleh semua user yang login (dipakai FE untuk pilihan lembaga);
menambah, mengubah dan menghapus hanya admin.

| Method | Path | Akses | Fungsi |
|---|---|---|---|
| POST | `/api/lembaga` | Admin | Tambah lembaga: `name` (2–150 karakter, unik) |
| GET | `/api/lembaga` | Login | Daftar lembaga, urut nama A–Z. Query: `page`, `limit` (1–100, default 10), `search` |
| GET | `/api/lembaga/:id` | Login | Detail lembaga |
| PUT | `/api/lembaga/:id` | Admin | Ganti nama: `name` (wajib) |
| DELETE | `/api/lembaga/:id` | Admin | Hapus lembaga (soft delete) |

- Nama kembar → 409. Seperti email user, nama lembaga yang sudah dihapus tetap terpakai.
- Lembaga yang masih menjadi asal barang tidak bisa dihapus (409). Hapus atau pindahkan barangnya dulu.

## API Barang

Semua endpoint butuh token. Melihat barang boleh semua user yang login; menambah, mengubah dan menghapus hanya admin.

| Method | Path | Akses | Fungsi |
|---|---|---|---|
| POST | `/api/items` | Admin | Tambah barang: `name`, `lembagaId` (wajib), `totalQuantity`, `description` (opsional, maks 1000 karakter) |
| GET | `/api/items` | Login | Daftar barang. Query: `page`, `limit` (1–100, default 10), `search` (nama atau keterangan), `available=true` |
| GET | `/api/items/:id` | Login | Detail barang |
| PUT | `/api/items/:id` | Admin | Update parsial: `name`, `description` (`""` = dikosongkan), `lembagaId`, `totalQuantity` |
| DELETE | `/api/items/:id` | Admin | Hapus barang (soft delete) |

```bash
curl -X POST localhost:3000/api/items -H "Authorization: Bearer <token admin>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Proyektor Epson","description":"Simpan di lemari B","lembagaId":1,"totalQuantity":5}'
```

Contoh data barang:

```json
{"id":1,"name":"Proyektor Epson","description":"Simpan di lemari B","lembaga":{"id":1,"name":"BEM"},
 "totalQuantity":5,"availableQuantity":3,"status":"TERSEDIA", ...}
```

- `description`: keterangan barang, boleh kosong (`""` jika tidak diisi).
- `lembagaId` (dikirim saat create/update) / `lembaga` (di response): lembaga asal barang. `lembagaId` yang tidak ada atau sudah dihapus → 404.
- `totalQuantity`: jumlah barang yang dimiliki (diatur admin).
- `availableQuantity`: sisa yang bisa dipinjam. Saat barang baru dibuat, nilainya sama dengan total; nanti
  berkurang saat pengajuan di-ACC dan bertambah saat barang dikembalikan.
- `status` tidak disimpan di database: `TERSEDIA` jika `availableQuantity > 0`, `HABIS` jika 0.
- Mengubah `totalQuantity` menggeser `availableQuantity` sebesar selisihnya. Total tidak boleh lebih kecil dari
  jumlah yang sedang dipinjam (409).
- Barang yang masih dipinjam tidak bisa dihapus (409).

## Struktur tabel (AutoMigrate)

Tabel tidak dibuat dari file SQL. Setiap kali API start, `database.Migrate` (GORM AutoMigrate) menyamakan
database dengan struct di `domain/entity` (tag `gorm:"..."`: ukuran kolom, `not null`, index unik, `check`).
Lalu, khusus `APP_ENV=development` (diset di `compose.dev.yaml`), `database.Seed` membuat user awal jika belum ada.

- Tambah kolom/tabel: ubah atau buat struct entity (tabel baru: daftarkan di `migrate.go`), lalu simpan — Air
  me-restart API dan AutoMigrate menambahkannya.
- AutoMigrate **tidak** menghapus atau mengganti nama kolom. Kolom yang dihapus dari struct tetap ada di database
  sampai dihapus manual (`ALTER TABLE ... DROP COLUMN ...`) atau database direset.

## Reset database

Menghapus semua data; tabel dan user awal dibuat ulang otomatis saat API start:

```bash
docker compose -f compose.dev.yaml down
docker volume rm sgeinventory_be_postgres_data
docker compose -f compose.dev.yaml up --build -d
```

(`down -v` juga bisa, tetapi ikut menghapus volume pgAdmin sehingga password server di pgAdmin perlu diisi lagi.)

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
Di produksi (`APP_ENV=production`) tabel tetap dibuat otomatis, tetapi user awal tidak; buat admin pertama
lewat register lalu `UPDATE users SET role='ADMIN' WHERE email='...'`.
