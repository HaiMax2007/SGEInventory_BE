package contracts

import "context"

// Transactor menjalankan beberapa langkah database dalam SATU transaksi, walaupun langkah-langkahnya
// memakai repository yang berbeda (mis. ACC pengajuan: ubah stok items + status borrowings + log_users).
//
// Cara pakai di service:
//
//	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
//		// semua repository yang dipanggil dengan ctx INI ikut transaksi
//		return nil // nil → COMMIT, error → ROLLBACK (semua langkah dibatalkan)
//	})
//
// Implementasinya ada di internal/infra/repository/transactor.go (GORM).
type Transactor interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
