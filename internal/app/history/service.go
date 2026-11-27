package history

import (
	"context"
	"slices"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/helpers/pagination"
)

type historyService struct {
	borrowingRepo   contracts.BorrowingRepository
	consignmentRepo contracts.ConsignmentRepository
	storage         contracts.FileStorage // mengubah key foto menjadi URL (Cloudinary / folder lokal)
}

// NewHistoryService membuat service riwayat pengajuan. Tidak punya repository sendiri: memakai repository
// peminjaman & penitipan yang sudah ada.
func NewHistoryService(
	borrowingRepo contracts.BorrowingRepository,
	consignmentRepo contracts.ConsignmentRepository,
	storage contracts.FileStorage,
) contracts.HistoryService {
	return &historyService{borrowingRepo: borrowingRepo, consignmentRepo: consignmentRepo, storage: storage}
}

// GetMyHistory menggabungkan peminjaman & penitipan milik userID, urut createdAt terbaru dulu.
//
// Cara menggabung dua tabel dengan halaman: baris-baris di halaman ke-p (offset..offset+limit) dari gabungan
// PASTI berasal dari (offset+limit) baris teratas masing-masing tabel. Jadi cukup ambil sebanyak itu dari tiap
// tabel, gabungkan, urutkan, lalu potong. Contoh page 2 limit 10 → ambil 20 teratas dari tiap tabel, urutkan
// 40 baris itu, kirim baris ke-11–20. Riwayat per user tidak banyak, jadi ini tetap ringan.
func (s *historyService) GetMyHistory(ctx context.Context, userID uint, query dto.HistoryListQuery) ([]dto.HistoryResponse, *pagination.Pagination, error) {
	page := pagination.New(query.Page, query.Limit)
	offset := page.Offset()
	need := offset + page.Limit

	borrowings, totalBorrowings, err := s.borrowingRepo.FindByUser(ctx, userID, 0, need)
	if err != nil {
		return nil, nil, err
	}
	consignments, totalConsignments, err := s.consignmentRepo.FindByUser(ctx, userID, 0, need)
	if err != nil {
		return nil, nil, err
	}
	page.SetTotal(totalBorrowings + totalConsignments)

	rows := make([]dto.HistoryResponse, 0, len(borrowings)+len(consignments))
	for i := range borrowings {
		res := dto.NewBorrowingResponse(&borrowings[i], s.storage.URL)
		rows = append(rows, dto.HistoryResponse{Type: enums.RequestTypeBorrowing, CreatedAt: res.CreatedAt, Borrowing: &res})
	}
	for i := range consignments {
		res := dto.NewConsignmentResponse(&consignments[i], s.storage.URL)
		rows = append(rows, dto.HistoryResponse{Type: enums.RequestTypeConsignment, CreatedAt: res.CreatedAt, Consignment: &res})
	}

	// Terbaru dulu. b.Compare(a) = urutan menurun.
	slices.SortStableFunc(rows, func(a, b dto.HistoryResponse) int { return b.CreatedAt.Compare(a.CreatedAt) })

	// Halaman di luar data (atau offset negatif karena page yang terlalu besar) → daftar kosong, bukan panic.
	if offset < 0 || offset >= len(rows) {
		return []dto.HistoryResponse{}, page, nil
	}
	return rows[offset:min(need, len(rows))], page, nil
}
