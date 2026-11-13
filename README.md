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
| phpMyAdmin | http://localhost:8081 (login otomatis) |
| MariaDB | `localhost:3307` (user `db_admin`, password `db_admin`, db `sgeinventorydb_main`) |

Untuk mengubah port, salin `.env.example` menjadi `.env` lalu ubah nilainya.

Kode di-mount ke container dan di-reload otomatis oleh [air](https://github.com/air-verse/air).

Konfigurasi database dibaca dari environment: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`.
Compose sudah mengisinya; jika menjalankan `go run .` di luar Docker, isi sendiri
(mis. `DB_HOST=localhost DB_PORT=3307`).

## Struktur kode

```
main.go                         titik awal: sambung DB → buat app Fiber → middleware → routes → listen
config/database.go              koneksi database (variabel global config.DB)
config/validator.go             membuat tag `validate:"..."` dicek otomatis saat c.Bind().Body()
models/user.go                  struct User (tabel users) + struct request register/update
models/auth.go                  struct request login + isi token JWT (claims)
controllers/user_controller.go  logika tiap endpoint user
controllers/auth_controller.go  login (membuat token) & /me
routes/routes.go                daftar URL → (middleware) → fungsi controller
middlewares/auth.go             Protected (cek token) & AdminOnly (cek role)
```

Alur request: `routes.go` → middleware (jika ada) → fungsi di `controllers` → baca/tulis lewat `config.DB`
memakai struct di `models` → balas JSON.

## Auth (JWT)

1. FE memanggil `POST /api/auth/login` dengan `{"email","password"}`.
2. Respons berisi `data.token` (berlaku 24 jam) dan `data.user`. FE menyimpan token, mis. di `localStorage`.
3. Setiap request yang butuh login mengirim header `Authorization: Bearer <token>`.
4. Logout cukup menghapus token di FE.

```js
// contoh di FE
const res = await fetch("http://localhost:3000/api/auth/login", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ email, password }),
});
const { data } = await res.json();
localStorage.setItem("token", data.token);

const me = await fetch("http://localhost:3000/api/auth/me", {
  headers: { Authorization: `Bearer ${localStorage.getItem("token")}` },
});
```

| Status | Arti |
|---|---|
| 401 | Token tidak ada / salah / kedaluwarsa, atau email/password salah saat login → arahkan ke halaman login |
| 403 | Token valid tapi tidak punya izin (bukan admin / bukan data miliknya) |
| 429 | Login lebih dari 5x per menit dari IP yang sama |

Alamat FE yang boleh memanggil API diatur lewat env `CORS_ORIGINS` (default dev:
`http://localhost:5173,http://localhost:3001`). Jika FE jalan di port lain, isi `CORS_ORIGINS` di `.env` lalu
`docker compose -f compose.dev.yaml up -d`.

Token menyimpan role saat login. Jika role user diubah atau user dihapus, token lama tetap berlaku sampai
kedaluwarsa; minta user login ulang.

## API User

Format respons: `{"message": "...", "data": ...}`; list menambah `meta` (`page`, `limit`, `total`, `totalPages`).
Error: `{"message": "..."}`; untuk data tidak valid (400) ditambah `"error"` berisi detail validasi.

| Method | Path | Akses | Fungsi |
|---|---|---|---|
| POST | `/api/auth/login` | Semua | Login, membalas token |
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
Belum ada data awal (seed); tambahkan data sendiri lewat phpMyAdmin bila perlu.

Catatan:
- Admin tidak bisa dibuat lewat API. Daftar seperti biasa, lalu ubah lewat phpMyAdmin:
  `UPDATE users SET role='ADMIN' WHERE email='...'`, kemudian login ulang agar token memuat role baru.
- User yang dihapus tetap menempati email-nya (soft delete), jadi email itu tidak bisa didaftarkan ulang.

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

Ganti semua `CHANGE_ME` di `compose.prod.yaml` terlebih dahulu: password database, `JWT_SECRET`
(string acak minimal 32 karakter, mis. `openssl rand -hex 32`; API menolak jalan jika kurang) dan
`CORS_ORIGINS` (domain FE). API terbuka di port `3000` tanpa HTTPS,
jadi pasang reverse proxy/Cloudflare di depannya. phpMyAdmin hanya bisa diakses dari server itu sendiri
(`127.0.0.1:8081`, gunakan SSH tunnel) dan port database tidak dibuka ke host.
