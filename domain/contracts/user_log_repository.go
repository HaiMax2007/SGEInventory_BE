package contracts

import (
	"context"

	"SGEInventory_BE/domain/entity"
)

// UserLogRepository membaca riwayat aktivitas (tabel log_users). Dipakai peminjaman, nanti juga penitipan.
// Log ditulis di dalam transaksi masing-masing fitur (mis. BorrowingRepository.Create).
// Implementasinya ada di internal/infra/repository/user_log_repository.go (GORM).
type UserLogRepository interface {
	// FindByRequest mengambil semua log sebuah pengajuan (urut dari yang paling lama) beserta pelakunya.
	FindByRequest(ctx context.Context, requestType string, requestID uint) ([]entity.UserLog, error)
	// FindAll mengambil satu halaman SEMUA log (terbaru dulu) beserta pelakunya + jumlah seluruhnya.
	// requestType kosong = semua jenis pengajuan.
	FindAll(ctx context.Context, requestType string, offset, limit int) ([]entity.UserLog, int64, error)
}
