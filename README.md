# SGEInventory_BE

Backend Sistem Manajemen Barang (Go + Fiber v3 + PostgreSQL).

## Menjalankan (development)

Hanya butuh Docker; tidak perlu Traefik atau network eksternal.

```bash
docker compose -f compose.dev.yaml up --build
```

| Layanan | URL / alamat |
|---|---|
| API | `http://localhost:${PORT}` (mis. http://localhost:4004) |
| pgAdmin | `http://localhost:${PGADMIN_PORT}` (tanpa login; saat membuka server pertama kali, isi password database lalu centang "Save password") |
| PostgreSQL | `localhost:${DB_PORT}` (mis. `localhost:5400`; user/password/db sesuai `.env`) |

Port di atas diatur lewat `.env` (salin dari `.env.example`). `PORT` dan `DB_PORT` di `.env` **hanya** menentukan
port yang dibuka di laptop. Di dalam Docker alamatnya selalu tetap (API listen di 3000, database `db:5432`),
diatur langsung di `compose.dev.yaml`, jadi mengganti port di `.env` tidak membuat API gagal tersambung.

Kode di-mount ke container dan di-reload otomatis oleh [air](https://github.com/air-verse/air).

Konfigurasi dibaca dari environment (`internal/config/config.go`): `PORT`, `DB_HOST`, `DB_PORT`, `DB_USER`,
`DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` (default `disable`), `JWT_SECRET`, `CORS_ORIGINS`, `CLOUDINARY_URL` (wajib di production, lihat "Penyimpanan foto"),
`CLOUDINARY_FOLDER` (default `sgeinventory`), `UPLOAD_DIR` (default `uploads`). Compose sudah mengisinya; jika menjalankan
`go run ./cmd/app` di luar Docker, isi sendiri (mis. `DB_HOST=localhost DB_PORT=<DB_PORT di .env>`).

```bash
go build ./...     # cek semua kode bisa di-compile
go vet ./...       # cek kesalahan umum
go test ./...      # jalankan unit test (folder tests/unit; tidak butuh database/Docker)
go test ./tests/... -v -run TestApprove   # contoh: hanya test yang namanya diawali TestApprove
```

## Struktur kode

Mengikuti pola Clean Architecture (handler → service → repository).

```
cmd/app/main.go                 titik awal: baca config → sambung DB → rakit repository/service/handler → routes
cmd/migrate, cmd/seed           (kosong) cadangan jika nanti migrasi/seed mau dijalankan sebagai program terpisah
domain/                         inti aplikasi, tidak tahu soal HTTP
  entity/                       struct tabel database + aturan bisnisnya (User, Lembaga, Item, Borrowing, BorrowingItem, Consignment, UserLog)
  dto/                          bentuk data request/response JSON (+ isi token JWT)
  contracts/                    interface repository & service ("janji" tiap lapisan)
  enums/                        konstanta role, status & tipe barang, status peminjaman & penitipan, jenis & aktivitas log
  errx/                         error bisnis + Kind-nya (400/401/403/404/409)
internal/
  app/<fitur>/                  satu folder per fitur: auth, user, lembaga, item, borrowing, consignment, userlog, history
    routes.go                   daftar URL → middleware → method handler
    handler.go                  baca request (params, body, query) → panggil service → balas JSON
    service.go                  aturan bisnis (hash password, cek stok, buat token, ...)
                                (borrowing: approval.go = tahap 2, return.go = tahap 3)
  config/config.go              membaca env
  infra/database/postgres.go    membuka koneksi GORM ke PostgreSQL
  infra/database/migrate.go     AutoMigrate: membuat/menyesuaikan tabel dari struct entity setiap app start
  infra/database/seed.go        data awal: user Max (ADMIN) & Raka (USER) + lembaga BEM, HMDTIF, DPM
                                (hanya saat APP_ENV=development)
  infra/repository/             query GORM (satu file per tabel)
  infra/storage/                penyimpanan foto (pengembalian & penitipan): Cloudinary, atau folder uploads/ (cadangan dev)
  middlewares/auth.go           Protected (cek token), AdminOnly & UserOnly (cek role)
  middlewares/error_handler.go  mengubah semua error menjadi JSON + status HTTP
helpers/pagination/             hitung page/limit/offset/totalPages
helpers/dates/                  TodayWIB(): "hari ini" menurut WIB, dipakai aturan tanggal peminjaman & penitipan
pkg/photo/                      aturan foto unggahan bersama: cek jenis dari isi file (maks 5 MB), nama acak, simpan/hapus
pkg/response/                   membuat respons sukses {"message","data","meta"}
pkg/validator/                  cek tag `validate:"..."` pada DTO, lalu ubah field yang gagal jadi errx
tests/unit/                     unit test: entity, service (repository palsu di fakes_test.go), middleware, validasi
```

Alur request, contoh `PUT /api/items/:id`:

1. `internal/app/item/routes.go` → middleware `Protected` (cek token) → `AdminOnly` (cek role)
2. `ItemHandler.Update` membaca `:id` & body JSON (JSON rusak / tipe salah → `errx.InvalidBody` → 400), lalu
   `validator.ValidateStruct(&req)` mengecek tag `validate` (gagal → errx milik field itu, mis. `ErrInvalidItemTotal`)
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

## Error validasi

Aturan ditulis di tag `validate:"..."` pada DTO dan dicek `validator.ValidateStruct` di handler. Setiap field punya
**satu** error di `domain/errx` (mis. `ErrInvalidEmail`) yang pesannya sudah menjelaskan aturannya; pemetaannya ada di
method `FieldErrors()` milik DTO. Field yang belum dipetakan dibalas cadangan `VALIDATION_ERROR` "Data tidak valid"
+ detail di `"error"`.
Jika ada beberapa kesalahan, yang dikirim hanya **yang pertama** sesuai urutan field di form. Spasi di awal/akhir
teks dibuang dulu, jadi `"   "` dianggap kosong (kecuali password). Semua berstatus 400.

```json
{"message": "Password wajib diisi, 8-72 karakter"}
```

| Form | Kode → pesan |
|---|---|
| Semua body JSON | `INVALID_BODY` "Format data tidak valid, kirim body JSON yang benar" (JSON rusak, atau header Content-Type tidak dikirim / tidak didukung seperti `text/plain`; FE selalu kirim `application/json`); `INVALID_FIELD_TYPE` "Field totalQuantity harus berupa angka bulat" (tipe salah; field di dalam daftar ditulis `items.0.quantity`, 0 = baris pertama) |
| Login | `INVALID_EMAIL`, `INVALID_PASSWORD` (kosong); email/password salah tetap 401 `INVALID_CREDENTIALS` |
| Registrasi / ubah user | `INVALID_FULL_NAME` "Nama lengkap wajib diisi, 3-255 karakter", `INVALID_NIM` "NIM wajib diisi, berupa angka 8-20 digit", `INVALID_EMAIL` "Email wajib diisi dengan format yang benar, mis. nama@mail.com", `INVALID_PASSWORD` "Password wajib diisi, 8-72 karakter", `INVALID_CONTACT` "Kontak harus 8-20 karakter", `NOTHING_TO_UPDATE` |
| Lembaga | `INVALID_LEMBAGA_NAME` "Nama lembaga wajib diisi, 2-150 karakter" |
| Barang | `INVALID_ITEM_NAME` "Nama barang wajib diisi, 2-255 karakter", `INVALID_ITEM_DESCRIPTION` "Keterangan barang maksimal 1000 karakter", `INVALID_ITEM_LEMBAGA` "Lembaga asal barang wajib dipilih", `INVALID_ITEM_TOTAL` "Total barang harus 0 sampai 1000000", `NOTHING_TO_UPDATE` |
| Peminjaman | `INVALID_BORROW_ITEMS` "Pilih 1-20 barang, setiap barang wajib dipilih dengan jumlah lebih dari 0", `INVALID_BORROW_LEMBAGA` "Lembaga wajib dipilih", `INVALID_PURPOSE` "Tujuan peminjaman wajib diisi, maksimal 500 karakter", `INVALID_PICKUP_DATE` / `INVALID_RETURN_DATE` "Tanggal ... wajib diisi dengan format YYYY-MM-DD". Aturan antar-field: lihat [API Peminjaman](#api-peminjaman) |
| Pengembalian | `INVALID_RETURN_PHOTO` "Foto pengembalian wajib diunggah: JPG, PNG, atau WebP, maksimal 5 MB" |
| Penitipan | `INVALID_FORM` (bukan multipart), `INVALID_CONSIGNMENT_LEMBAGA` "Lembaga asal wajib dipilih", `INVALID_EVENT_NAME` "Nama acara wajib diisi, 3-255 karakter", `INVALID_CONSIGNMENT_ITEMS` "Isi 1-20 barang titipan, ..." (juga untuk JSON `items` yang rusak), `INVALID_START_DATE` / `INVALID_END_DATE`, `INVALID_CONSIGNMENT_PHOTO`, `INVALID_CONSIGNMENT_NOTES` "Catatan maksimal 1000 karakter". Aturan antar-field: lihat [API Penitipan](#api-penitipan) |

URL yang tidak ada → 404 "Endpoint tidak ditemukan"; method salah → 405 "Method tidak diizinkan".

## API User

Format respons: `{"message": "...", "data": ...}`; list menambah `meta` (`page`, `limit`, `total`, `totalPages`).
Error: `{"message": "..."}` (pesan Bahasa Indonesia yang bisa langsung ditampilkan FE); untuk cadangan
`VALIDATION_ERROR` ditambah `"error"` berisi detail validasi. Lihat
[Error validasi](#error-validasi) untuk daftar pesan per form.

| Method | Path | Akses | Fungsi |
|---|---|---|---|
| POST | `/api/auth/login` | Semua | Login, membalas token |
| POST | `/api/auth/logout` | Login | Logout (FE lalu menghapus token-nya sendiri) |
| GET | `/api/auth/me` | Login | Data user yang sedang login |
| POST | `/api/users` | Semua | Registrasi user (role otomatis `USER`). Wajib: `fullName` (3–255 karakter), `nim` (8–20 digit angka, unik), `email` (unik), `password` (8 karakter – 72 byte). Opsional: `contact` (8–20 karakter) |
| GET | `/api/users` | Admin | Daftar user. Query: `page`, `limit` (1–100, default 10), `search`, `role` |
| GET | `/api/users/:id` | Admin / pemilik | Detail user |
| PUT | `/api/users/:id` | Admin / pemilik | Update parsial: `fullName`, `email`, `contact` (kosong = tidak diubah; aturan sama dengan registrasi) |
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
| PUT | `/api/lembaga/:id` | Admin | Ganti nama: `name` (wajib, 2–150 karakter) |
| DELETE | `/api/lembaga/:id` | Admin | Hapus lembaga (soft delete) |

- Nama kembar → 409. Seperti email user, nama lembaga yang sudah dihapus tetap terpakai.
- Lembaga yang masih menjadi asal barang tidak bisa dihapus (409). Hapus atau pindahkan barangnya dulu.

## API Barang

Semua endpoint butuh token. Melihat barang boleh semua user yang login; menambah, mengubah dan menghapus hanya admin.

| Method | Path | Akses | Fungsi |
|---|---|---|---|
| POST | `/api/items` | Admin | Tambah barang: `name` (2–255 karakter), `lembagaId` (wajib), `totalQuantity` (0–1.000.000), `description` (opsional, maks 1000 karakter) |
| GET | `/api/items` | Login | Daftar barang. Query: `page`, `limit` (1–100, default 10), `search` (nama atau keterangan), `available=true`, `type` (khusus admin: `BORROWABLE` / `CONSIGNMENT`; lainnya → 400) |
| GET | `/api/items/:id` | Login | Detail barang (barang titipan: hanya admin, user → 404) |
| PUT | `/api/items/:id` | Admin | Update parsial: `name`, `description` (`""` = dikosongkan), `lembagaId`, `totalQuantity`. Barang titipan → 409 |
| DELETE | `/api/items/:id` | Admin | Hapus barang (soft delete). Barang titipan → 409 |

```bash
curl -X POST localhost:3000/api/items -H "Authorization: Bearer <token admin>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Proyektor Epson","description":"Simpan di lemari B","lembagaId":1,"totalQuantity":5}'
```

Contoh data barang:

```json
{"id":1,"name":"Proyektor Epson","description":"Simpan di lemari B","lembaga":{"id":1,"name":"BEM"},
 "totalQuantity":5,"availableQuantity":3,"status":"TERSEDIA","type":"BORROWABLE","consignmentId":null, ...}
```

- `type`: `BORROWABLE` = barang lembaga yang dibuat admin dan bisa dipinjam (barang lama otomatis bertipe ini);
  `CONSIGNMENT` = barang titipan user yang dibuat lewat [API Penitipan](#api-penitipan), dengan `consignmentId` =
  id penitipannya. User hanya melihat `BORROWABLE` (daftar & detail); admin melihat semua dan bisa memfilter
  `?type=`. Barang titipan tidak bisa dipinjam (dianggap tidak ditemukan) dan tidak bisa diubah/dihapus lewat
  `/api/items` (409 `CONSIGNMENT_ITEM_READ_ONLY`).

- `description`: keterangan barang, boleh kosong (`""` jika tidak diisi).
- `lembagaId` (dikirim saat create/update) / `lembaga` (di response): lembaga asal barang. `lembagaId` yang tidak ada atau sudah dihapus → 404.
- `totalQuantity`: jumlah barang yang dimiliki (diatur admin), 0 sampai 1.000.000. Total 0 boleh (barang dicatat tapi `HABIS`).
- `availableQuantity`: sisa yang bisa dipinjam. Saat barang baru dibuat, nilainya sama dengan total; nanti
  berkurang saat pengajuan di-ACC dan bertambah saat barang dikembalikan.
- `status` tidak disimpan di database: `TERSEDIA` jika `availableQuantity > 0`, `HABIS` jika 0.
- Mengubah `totalQuantity` menggeser `availableQuantity` sebesar selisihnya. Total tidak boleh lebih kecil dari
  jumlah yang sedang dipinjam (409).
- Barang yang masih dipinjam tidak bisa dihapus (409).

## API Peminjaman

| Method | URL | Akses | Keterangan |
|---|---|---|---|
| POST | `/api/borrowings` | **User (role USER)** | Kirim pengajuan peminjaman (admin → 403) |
| GET | `/api/borrowings` | Admin | Daftar **semua** pengajuan, terbaru dulu. Query: `page`, `limit`, `status` (opsional, mis. `PENDING`; status tidak dikenal → 400) |
| GET | `/api/borrowings/me` | Login | Daftar pengajuan milik sendiri + status terkini (tanpa `timeline`). Query: `page`, `limit` |
| GET | `/api/borrowings/:id` | Pemilik / Admin | Detail **satu** pengajuan + `timeline` (riwayat aktivitas). Pengajuan orang lain → 404 |
| PATCH | `/api/borrowings/:id/approve` | Admin | ACC pengajuan PENDING (tanpa body). Stok berkurang + tolak otomatis pengajuan lain yang tidak tercukupi. Response tanpa `timeline` |
| PATCH | `/api/borrowings/:id/reject` | Admin | Tolak pengajuan PENDING (tanpa body). Stok tidak berubah |
| POST | `/api/borrowings/:id/return` | **User (pemilik)** | Unggah foto pengembalian (`multipart/form-data`, field `photo`) |
| PATCH | `/api/borrowings/:id/return/approve` | Admin | ACC pengembalian (tanpa body). Stok kembali, status `RETURNED` |
| PATCH | `/api/borrowings/:id/return/reject` | Admin | Tolak pengembalian (tanpa body). Foto dihapus, user unggah ulang |

```bash
curl -X POST localhost:3000/api/borrowings -H "Authorization: Bearer <token user>" \
  -H "Content-Type: application/json" \
  -d '{"lembagaId":2,"purpose":"Seminar nasional HMDTIF","pickUpDate":"2026-11-25","returnDate":"2026-11-28",
       "items":[{"itemId":1,"quantity":2},{"itemId":4,"quantity":1}]}'
```

Contoh detail (`GET /api/borrowings/1`):

```json
{"id":1,"status":"PENDING","purpose":"Seminar nasional HMDTIF","pickUpDate":"2026-11-25","returnDate":"2026-11-28",
 "user":{"id":2,"fullName":"Raka","nim":"22010001","contact":"081234567890"},"lembaga":{"id":2,"name":"HMDTIF"},
 "items":[{"itemId":1,"itemName":"Proyektor Epson","quantity":2}],
 "timeline":[{"activity":"SUBMITTED","actor":{"id":2,"fullName":"Raka","nim":"22010001","contact":"081234567890"},"createdAt":"..."}], ...}
```

`user` (peminjam / penitip) dan `actor` (pelaku di timeline & log) berisi `id`, `fullName`, `nim`, `contact`
(`contact` bisa `""` jika belum diisi).

Aturan submit. Aturan per field ada di tag `validate` `dto.CreateBorrowingRequest` (satu error per field, lihat
[Error validasi](#error-validasi)); aturan yang membandingkan beberapa field dicek di service dengan error sendiri:
- `items`: 1–20 jenis barang, `itemId` wajib, `quantity > 0` (`INVALID_BORROW_ITEMS`). Barang yang sama tidak boleh
  dipilih dua kali (`DUPLICATE_BORROW_ITEM` "Barang ke-3 sama dengan barang ke-1, gabungkan jumlahnya").
  Barang yang tidak ada → 404 (id-nya disebut). Jumlah melebihi `availableQuantity` → 409, semua barang yang
  kurang disebut sekaligus.
- `lembagaId` wajib (`INVALID_BORROW_LEMBAGA`) dan lembaganya harus ada (404).
- `purpose` wajib (spasi saja dianggap kosong), maksimal 500 karakter (`INVALID_PURPOSE`).
- `pickUpDate` & `returnDate` wajib, format `YYYY-MM-DD` (`INVALID_PICKUP_DATE` / `INVALID_RETURN_DATE`).
  `pickUpDate` tidak boleh sebelum hari ini WIB (`PICKUP_DATE_PAST`); `returnDate` harus **setelah** `pickUpDate`,
  hari yang sama ditolak (`RETURN_NOT_AFTER_PICKUP`), dan paling lambat `pickUpDate` + 7 hari (`BORROW_TOO_LONG`).
- Peminjam diambil dari token, bukan dari body. Nama barang diambil dari database dan disalin ke pengajuan.
- Submit **tidak** mengurangi stok; stok berkurang saat admin ACC.
- Pengajuan, barang-barangnya, dan log `SUBMITTED` disimpan dalam satu transaksi.

### Approval admin

```bash
curl -X PATCH localhost:3000/api/borrowings/5/approve -H "Authorization: Bearer <token admin>"
```

```json
{"message":"Pengajuan peminjaman disetujui",
 "data":{"borrowing":{"id":5,"status":"APPROVED","purpose":"...", "user":{...}, "items":[...], "returnPhotoUrl":null, ...},
         "autoRejectedIds":[7,9]}}
```

Response semua aksi (buat pengajuan, ACC/tolak, unggah/ACC/tolak pengembalian) **tanpa** `timeline`;
riwayat lengkap hanya di `GET /api/borrowings/:id`.

ACC (semua langkah dalam **satu transaksi**; gagal satu → semua dibatalkan):
1. Barang-barang pengajuan dikunci (`SELECT ... FOR UPDATE`, urut id), lalu pengajuannya.
2. Status harus `PENDING` (selain itu 409 "Pengajuan sudah diproses"). Barang yang sudah dihapus → 409; stok
   kurang → 409 dengan semua barang yang kurang disebut. Stok dicek ulang di sini karena bisa berubah sejak submit.
3. `availableQuantity` tiap barang dikurangi, status → `APPROVED`, log `APPROVED` (pelaku = admin).
4. **Tolak otomatis**: pengajuan `PENDING` lain yang memuat barang-barang tadi dan jumlahnya kini melebihi stok
   → `REJECTED` + log `AUTO_REJECTED` (pelaku = admin yang ACC). Id-nya dikembalikan di `autoRejectedIds`.
   Hanya barang yang stoknya berubah yang diperiksa (satu query, memakai index `borrowing_items.item_id`).

Tolak: status → `REJECTED` + log `REJECTED` (pelaku = admin); stok tidak disentuh.

Mengurangi `totalQuantity` barang lewat `PUT /api/items/:id` juga menolak otomatis pengajuan `PENDING` yang
tidak tercukupi lagi (log `AUTO_REJECTED`, pelaku = admin yang mengedit). Edit & hapus barang mengunci baris
barang, jadi tidak bisa bertabrakan dengan ACC yang berjalan bersamaan.

Status:

```
PENDING ─ACC─→ APPROVED ─user unggah foto─→ RETURN_PENDING ─ACC─→ RETURNED (stok kembali, selesai)
   └─REJECT─→ REJECTED            ▲                        │
              (final)             └──── RETURN_REJECTED ◄───┘ admin tolak foto (foto dihapus, unggah ulang)
```

ACC/tolak pengajuan hanya sekali; pengajuan yang ditolak selesai, user membuat pengajuan baru. Hanya foto
pengembalian yang boleh dikirim ulang (setelah ditolak).

`timeline` berasal dari tabel `log_users`. `actor` adalah **pelaku** aktivitas (user saat mengajukan, admin saat
ACC/menolak/memicu tolak otomatis), sedangkan pemilik pengajuan ada di `borrowings.user_id` (di response: `user`).
Timeline diambil per pengajuan (`request_type` + `request_id`), jadi aksi admin tetap terlihat oleh peminjam.
Aktivitas: `SUBMITTED`, `APPROVED`, `REJECTED`, `AUTO_REJECTED`, `RETURN_SUBMITTED` (pelaku = user),
`RETURN_APPROVED`, `RETURN_REJECTED` (pelaku = admin).

### Pengembalian (tahap 3)

```bash
# user (pemilik pengajuan) mengunggah foto bukti: multipart/form-data, field "photo"
curl -X POST localhost:3000/api/borrowings/5/return -H "Authorization: Bearer <token user>" \
  -F "photo=@foto-barang.jpg"

# admin memproses
curl -X PATCH localhost:3000/api/borrowings/5/return/approve -H "Authorization: Bearer <token admin>"
curl -X PATCH localhost:3000/api/borrowings/5/return/reject  -H "Authorization: Bearer <token admin>"
```

```js
// contoh di FE: jangan isi header Content-Type sendiri, browser yang mengisi boundary multipart
const form = new FormData();
form.append("photo", fileInput.files[0]);
await fetch(`http://localhost:3000/api/borrowings/${id}/return`, {
  method: "POST",
  headers: { Authorization: `Bearer ${token}` },
  body: form,
});
```

- **Unggah** (`POST /:id/return`, role USER): hanya pemilik (pengajuan orang lain → 404) dan hanya saat status
  `APPROVED` atau `RETURN_REJECTED` (selain itu 409 `BORROWING_NOT_RETURNABLE`). Foto wajib JPG/PNG/WebP, maksimal
  5 MB (`INVALID_RETURN_PHOTO`); jenisnya dicek dari **isi** file, bukan dari nama file. Body lebih dari 6 MB
  langsung ditolak Fiber (413). Status → `RETURN_PENDING`, log `RETURN_SUBMITTED` (pelaku = user). Response berisi
  data pengajuan **tanpa** `timeline`; riwayat lengkap ada di `GET /:id`.
- **ACC** (admin): status harus `RETURN_PENDING` (selain itu 409 `RETURN_NOT_PENDING`). Dalam satu transaksi,
  barang dikunci (urutan sama dengan ACC peminjaman), `availableQuantity` tiap barang **bertambah** sesuai jumlah di
  pengajuan, status → `RETURNED` (selesai), log `RETURN_APPROVED`. Foto tetap disimpan sebagai bukti.
- **Tolak** (admin): status → `RETURN_REJECTED`, foto lama **dihapus**, log `RETURN_REJECTED`; stok tidak berubah.
  User lalu mengunggah foto baru.

### Penyimpanan foto (Cloudinary)

Foto disimpan di **Cloudinary** jika env `CLOUDINARY_URL` diisi (wajib di production; API menolak jalan jika
kosong). Di development, jika `CLOUDINARY_URL` kosong, foto disimpan di folder `uploads/` proyek sebagai cadangan
(mis. saat offline). Log saat API start menunjukkan mana yang dipakai: `penyimpanan foto: Cloudinary` atau
`penyimpanan foto: folder lokal`.

Cara mengisi `CLOUDINARY_URL`:
1. Masuk ke dashboard Cloudinary → **Settings → API Keys**, salin *API environment variable*
   (`cloudinary://API_KEY:API_SECRET@CLOUD_NAME`).
2. Tempel ke `.env` (development, dan juga `.env` di server untuk production): `CLOUDINARY_URL=cloudinary://...`
   tanpa tanda kutip. Nilai ini **rahasia** (berisi API secret): jangan di-commit, jangan dibagikan.
3. Buat ulang container API agar env terbaca: `docker compose -f compose.dev.yaml up -d api`.

Database hanya menyimpan *key* foto di `borrowings.return_photo` (mis. `returns/8f3a9c...e1.jpg`, nama acak yang
tidak bisa ditebak). Response pengajuan memuat `returnPhotoUrl` yang dibuat dari key tersebut (`null` jika belum
ada foto):

| Penyimpanan | `returnPhotoUrl` |
|---|---|
| Cloudinary | `https://res.cloudinary.com/<cloud>/image/upload/sgeinventory/returns/8f3a...jpg` (FE langsung pakai di `<img src>`) |
| Folder lokal (dev) | `/uploads/returns/8f3a...jpg` (gabungkan dengan alamat API, mis. `http://localhost:3000/uploads/...`) |

Di Cloudinary semua foto (local maupun production) masuk ke folder `sgeinventory/returns` (bisa diganti lewat env
`CLOUDINARY_FOLDER`). Saat admin menolak, asset dihapus dari Cloudinary beserta cache CDN-nya, sehingga URL lama
menjadi 404 dalam beberapa detik. Route `GET /uploads/...` hanya aktif di mode folder lokal (folder bisa diganti
lewat env `UPLOAD_DIR`, default `uploads`, diabaikan git).

## API Penitipan

User menitipkan barang miliknya ke admin (status `PENDING`), lalu admin menyetujui (`APPROVED`) atau menolak
(`REJECTED`). Alur selesai di sini: tidak ada form lanjutan (mis. pengambilan barang); tenggat (`endDate`) cukup
ditampilkan/diatur FE.

| Method | URL | Akses | Keterangan |
|---|---|---|---|
| POST | `/api/consignments` | **User** | Ajukan penitipan (`multipart/form-data`, karena ada foto) |
| GET | `/api/consignments/me` | Login | Daftar penitipan milik sendiri (dari token), terbaru dulu. Query: `page`, `limit` |
| GET | `/api/consignments` | Admin | Semua penitipan, terbaru dulu (berisi `user` penitip). Query: `page`, `limit`, `status` (opsional: `PENDING` / `APPROVED` / `REJECTED`; lainnya → 400 `INVALID_CONSIGNMENT_STATUS`) |
| GET | `/api/consignments/:id` | Pemilik / Admin | Detail satu penitipan (tanpa timeline). Milik orang lain → 404 |
| PATCH | `/api/consignments/:id/approve` | Admin | ACC penitipan `PENDING` (tanpa body) |
| PATCH | `/api/consignments/:id/reject` | Admin | Tolak penitipan `PENDING` (tanpa body). Barang titipan & fotonya dihapus |

Field form: `lembagaId`, `eventName`, `items` (**teks JSON**, bentuknya sama dengan pengajuan peminjaman),
`startDate`, `endDate`, `photo` (file), `notes` (opsional).

```js
// contoh di FE: jangan isi header Content-Type sendiri, browser yang mengisi boundary multipart
const form = new FormData();
form.append("lembagaId", 2);
form.append("eventName", "Pameran HMDTIF");
form.append("items", JSON.stringify([{ name: "Kursi lipat", quantity: 10 }, { name: "Meja panjang", quantity: 2 }]));
form.append("startDate", "2026-11-26");
form.append("endDate", "2026-11-28");
form.append("photo", fileInput.files[0]);
form.append("notes", "Simpan di gudang belakang"); // boleh kosong
await fetch(`${API}/api/consignments`, { method: "POST", headers: { Authorization: `Bearer ${token}` }, body: form });
```

```bash
curl -X POST localhost:3000/api/consignments -H "Authorization: Bearer <token user>" \
  -F lembagaId=2 -F eventName="Pameran HMDTIF" \
  -F 'items=[{"name":"Kursi lipat","quantity":10}]' \
  -F startDate=2026-11-26 -F endDate=2026-11-28 -F photo=@barang.jpg -F notes=""
```

Response (201):

```json
{"message":"Pengajuan penitipan berhasil dikirim",
 "data":{"id":1,"status":"PENDING","eventName":"Pameran HMDTIF","startDate":"2026-11-26","endDate":"2026-11-28",
         "notes":"Simpan di gudang belakang","photoUrl":"https://res.cloudinary.com/<cloud>/image/upload/sgeinventory/consignments/9a17...png",
         "user":{"id":2,"fullName":"Raka","nim":"22010001","contact":"081234567890"},"lembaga":{"id":2,"name":"HMDTIF"},
         "items":[{"id":15,"name":"Kursi lipat","quantity":10},{"id":16,"name":"Meja panjang","quantity":2}], ...}}
```

Aturan (sama dengan pengajuan peminjaman):
- `items`: 1–20 barang, `name` 2–255 karakter, `quantity` 1–1.000.000. Nama yang sama tidak boleh ditulis dua kali
  (tidak membedakan huruf besar/kecil): `DUPLICATE_CONSIGNMENT_ITEM`. JSON rusak / tipe salah → `INVALID_CONSIGNMENT_ITEMS`.
- `lembagaId` wajib dan lembaganya harus ada (404). Teks (mis. `abc`) → `INVALID_FIELD_TYPE`.
- `startDate` tidak boleh sebelum hari ini WIB (`START_DATE_PAST`); `endDate` harus **setelah** `startDate`
  (`END_NOT_AFTER_START`), paling lambat `startDate` + 7 hari (`CONSIGNMENT_TOO_LONG`).
- `photo` wajib: JPG/PNG/WebP (dicek dari isi file), maksimal 5 MB. Disimpan di storage yang sama dengan foto
  pengembalian (folder `consignments/`), lihat [Penyimpanan foto](#penyimpanan-foto-cloudinary).
- `notes` boleh kosong, maksimal 1000 karakter.

Setelah lolos validasi, dalam **satu transaksi**: baris `consignments` (status `PENDING`) + setiap barang menjadi
baris baru di `items` (`type` `CONSIGNMENT`, `consignment_id` = id penitipan, lembaga = lembaga penitip, total =
tersedia = jumlah titipan) + log `SUBMITTED` (`request_type` `CONSIGNMENT`). Jika transaksi gagal, foto yang
sudah terunggah dihapus lagi. Catatan FE: tepat setelah unggah, `photoUrl` Cloudinary sesekali butuh beberapa
detik sebelum bisa dibuka.

### ACC / tolak penitipan (admin)

```bash
curl -X PATCH localhost:3000/api/consignments/5/approve -H "Authorization: Bearer <token admin>"
curl -X PATCH localhost:3000/api/consignments/5/reject  -H "Authorization: Bearer <token admin>"
```

- Hanya penitipan `PENDING` yang bisa diproses, sekali & final. Selain itu 409 `CONSIGNMENT_ALREADY_PROCESSED`
  "Pengajuan penitipan sudah diproses (status: ...)"; id tidak ada → 404. Baris penitipan dikunci
  (`SELECT ... FOR UPDATE`), jadi dua admin yang memproses bersamaan tidak sama-sama berhasil.
- **ACC**: status → `APPROVED`, log `APPROVED` (pelaku = admin). Barang titipan tetap ada dengan status
  `TERSEDIA` (status barang tetap dihitung dari stok).
- **Tolak** (satu transaksi): status → `REJECTED` (baris penitipan tetap ada), barang titipannya **dihapus
  permanen** dari `items`, log `REJECTED`. Setelah tersimpan, foto dihapus dari storage (gagal hapus hanya dicatat
  di log server). Response penitipan yang ditolak: `"items": []`, `"photoUrl": null`.
- Response ACC/tolak berisi data penitipan (tanpa timeline), sama seperti detail.

## API Riwayat Aktivitas (log)

| Method | URL | Akses | Keterangan |
|---|---|---|---|
| GET | `/api/logs` | Admin | Semua isi `log_users`, terbaru dulu. Query: `page`, `limit`, `requestType` (opsional: `BORROWING` / `CONSIGNMENT`; lainnya → 400), `search` (opsional: sebagian nama lengkap pelaku, tanpa beda huruf besar/kecil; bisa digabung `requestType`) |

```json
{"id":3,"requestType":"BORROWING","requestId":3,"activity":"SUBMITTED",
 "actor":{"id":2,"fullName":"Raka","nim":"22010001","contact":"081234567890"},"createdAt":"..."}
```

`requestType` + `requestId` menunjuk pengajuan yang dimaksud; detailnya bisa dibuka lewat
`GET /api/borrowings/{requestId}` (untuk `BORROWING`).

## API Riwayat Pengajuan User

| Method | URL | Akses | Keterangan |
|---|---|---|---|
| GET | `/api/history/me` | User (`UserOnly`) | Gabungan peminjaman + penitipan milik pemilik token, urut `createdAt` **terbaru dulu**. Query: `page`, `limit`. Admin → 403 |

Tiap baris punya `type`; hanya satu dari `borrowing` / `consignment` yang terisi, sisanya `null`. Isinya sama
dengan `GET /api/borrowings/me` dan `GET /api/consignments/me` (tanpa timeline). `meta.total` = jumlah
peminjaman + penitipan.

```json
{"message":"Riwayat pengajuan berhasil diambil","data":[
  {"type":"CONSIGNMENT","createdAt":"2026-11-26T09:00:00Z","borrowing":null,
   "consignment":{"id":3,"status":"PENDING","eventName":"Pameran","photoUrl":"https://...", "items":[...], "...":"..."}},
  {"type":"BORROWING","createdAt":"2026-11-25T08:00:00Z","consignment":null,
   "borrowing":{"id":5,"status":"APPROVED","purpose":"Rapat","items":[...], "...":"..."}}
 ],"meta":{"page":1,"limit":10,"total":2,"totalPages":1}}
```

## Struktur tabel (AutoMigrate)

Tabel tidak dibuat dari file SQL. Setiap kali API start, `database.Migrate` (GORM AutoMigrate) menyamakan
database dengan struct di `domain/entity` (tag `gorm:"..."`: ukuran kolom, `not null`, index unik, `check`).
Lalu, khusus `APP_ENV=development` (diset di `compose.dev.yaml`), `database.Seed` membuat user awal jika belum ada.

- Tambah kolom/tabel: ubah atau buat struct entity (tabel baru: daftarkan di `migrate.go`), lalu simpan — Air
  me-restart API dan AutoMigrate menambahkannya.
- AutoMigrate **tidak** menghapus atau mengganti nama kolom. Kolom yang dihapus dari struct tetap ada di database
  sampai dihapus manual (`ALTER TABLE ... DROP COLUMN ...`) atau database direset.
- AutoMigrate juga **tidak** mengubah `check:` yang namanya sudah ada di database. Jika isi CHECK di struct diubah,
  ganti manual. Contoh (dibutuhkan database yang dibuat sebelum aktivitas `AUTO_REJECTED` ada):

  ```sql
  BEGIN;
  ALTER TABLE log_users DROP CONSTRAINT chk_log_users_request_activity;
  ALTER TABLE log_users ADD CONSTRAINT chk_log_users_request_activity CHECK (request_activity IN
    ('SUBMITTED','APPROVED','REJECTED','AUTO_REJECTED','RETURN_SUBMITTED','RETURN_APPROVED','RETURN_REJECTED'));
  COMMIT;
  ```

  Untuk database yang dibuat sebelum status `RETURN_REJECTED` ada (tahap pengembalian):

  ```sql
  BEGIN;
  ALTER TABLE borrowings DROP CONSTRAINT chk_borrowings_status;
  ALTER TABLE borrowings ADD CONSTRAINT chk_borrowings_status CHECK (status IN
    ('PENDING','APPROVED','REJECTED','RETURN_PENDING','RETURN_REJECTED','RETURNED'));
  COMMIT;
  ```

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

`compose.prod.yaml` tidak berisi rahasia; semua nilai diambil dari file `.env` di server (jangan di-commit).
Daftar lengkapnya ada di komentar paling atas `compose.prod.yaml`: `DB_USER`, `DB_PASSWORD`, `DB_NAME`,
`JWT_SECRET` (string acak minimal 32 karakter, mis. `openssl rand -hex 32`), `CORS_ORIGINS` (domain FE),
`CLOUDINARY_URL` (lihat "Penyimpanan foto"), serta `API_PORT` (default 3000), `PGADMIN_EMAIL`,
`PGADMIN_PASSWORD`, `PGADMIN_PORT` (default 8081). Variabel wajib yang kosong membuat compose langsung berhenti
dengan pesan "... wajib diisi di .env". User/password/nama database dipakai bersama oleh service `api` dan `db`,
jadi otomatis sama.

API terbuka di port `API_PORT` tanpa HTTPS, jadi pasang reverse proxy/Cloudflare di depannya (lalu ubah port API
menjadi `127.0.0.1:...`, lihat komentar di file). Port database tidak dibuka ke host.

**pgAdmin** tidak ikut menyala (profile `tools`) dan tidak bisa dibuka dari internet (port hanya di
`127.0.0.1` server). Nyalakan saat perlu, akses lewat SSH tunnel, lalu matikan lagi:

```bash
docker compose -f compose.prod.yaml --profile tools up -d pgadmin   # di server
ssh -L 8081:127.0.0.1:8081 user@ip-server                            # di laptop → buka http://localhost:8081
docker compose -f compose.prod.yaml --profile tools stop pgadmin    # di server, setelah selesai
```

Di pgAdmin tambahkan server dengan host `db`, port `5432`. Jangan membuat domain/route reverse proxy ke pgAdmin.

Di produksi (`APP_ENV=production`) tabel tetap dibuat otomatis, tetapi user awal tidak; buat admin pertama
lewat register lalu `UPDATE users SET role='ADMIN' WHERE email='...'`. Foto pengembalian disimpan di Cloudinary,
bukan di server, jadi tidak hilang saat container dibuat ulang.
