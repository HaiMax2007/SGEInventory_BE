package contracts

import (
	"context"

	"SGEInventory_BE/domain/entity"
)

// UserLogRepository membaca & menulis riwayat aktivitas (tabel log_users). Dipakai peminjaman, nanti juga
// penitipan. Log ditulis di dalam transaksi fitur yang bersangkutan (BorrowingRepository.Create, atau
// Transactor.WithinTransaction), jadi aktivitas & perubahan datanya selalu tersimpan bersama.
// Implementasinya ada di internal/infra/repository/user_log_repository.go (GORM).
type UserLogRepository interface {
	// Create menyimpan satu log.
	Create(ctx context.Context, log *entity.UserLog) error
	// CreateMany menyimpan banyak log dengan satu INSERT.
	CreateMany(ctx context.Context, logs []entity.UserLog) error
	// FindByRequest mengambil semua log sebuah pengajuan (urut dari yang paling lama) beserta pelakunya.
	FindByRequest(ctx context.Context, requestType string, requestID uint) ([]entity.UserLog, error)
	// FindAll mengambil satu halaman SEMUA log (terbaru dulu) beserta pelakunya + jumlah seluruhnya.
	// requestType kosong = semua jenis pengajuan.
	FindAll(ctx context.Context, requestType string, offset, limit int) ([]entity.UserLog, int64, error)
}
