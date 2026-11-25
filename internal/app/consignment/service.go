// Package consignment berisi fitur penitipan barang: user menitipkan barang miliknya ke admin.
package consignment

import (
	"context"
	"errors"
	"strings"
	"time"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
	"SGEInventory_BE/helpers/dates"
	"SGEInventory_BE/pkg/photo"
)

// maxConsignmentDays: tanggal selesai paling lambat tanggal mulai + 7 hari (sama dengan peminjaman).
// Batas lain (1-20 barang, panjang teks) ada di tag `validate` dto.CreateConsignmentRequest.
const maxConsignmentDays = 7

// photoFolder: folder key foto penitipan, mis. "consignments/8f3a...jpg".
const photoFolder = "consignments"

type consignmentService struct {
	repo        contracts.ConsignmentRepository
	lembagaRepo contracts.LembagaRepository
	storage     contracts.FileStorage // tempat menyimpan foto barang titipan
}

// NewConsignmentService membuat service penitipan. Repository dioper dari main.go (dependency injection).
func NewConsignmentService(
	repo contracts.ConsignmentRepository,
	lembagaRepo contracts.LembagaRepository,
	storage contracts.FileStorage,
) contracts.ConsignmentService {
	return &consignmentService{repo: repo, lembagaRepo: lembagaRepo, storage: storage}
}

// CreateConsignment menjalankan alur submit penitipan:
//
//  1. cek aturan antar-field (tanpa database). Aturan per field sudah dicek handler lewat tag `validate`.
//  2. lembaga harus ada
//  3. cek foto (ukuran & jenis dari isi file), lalu simpan dengan nama acak
//  4. dalam SATU transaksi: consignments + items (tipe CONSIGNMENT) + log SUBMITTED
//
// Foto disimpan sebelum transaksi (menyimpan file tidak bisa di-rollback). Jika transaksi gagal, foto itu
// dihapus lagi supaya tidak ada file yatim.
func (s *consignmentService) CreateConsignment(ctx context.Context, userID uint, req dto.CreateConsignmentRequest) (*dto.ConsignmentResponse, error) {
	form, err := validateCreateRequest(req, dates.TodayWIB())
	if err != nil {
		return nil, err
	}

	// Lembaga harus ada dan belum dihapus.
	lembaga, err := s.lembagaRepo.FindByID(ctx, req.LembagaID)
	if err != nil {
		return nil, err // errx.ErrLembagaNotFound
	}

	ext, err := photo.Extension(req.Photo)
	if errors.Is(err, photo.ErrInvalid) {
		return nil, errx.ErrInvalidConsignmentPhoto
	}
	if err != nil {
		return nil, err
	}
	photoKey, err := photo.NewKey(photoFolder, ext)
	if err != nil {
		return nil, err
	}
	if err := photo.Save(ctx, s.storage, req.Photo, photoKey); err != nil {
		return nil, err
	}

	// Setiap barang titipan menjadi baris baru di tabel items (tipe CONSIGNMENT, lembaga = lembaga penitip).
	items := make([]entity.Item, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, entity.NewConsignmentItem(it.Name, lembaga.ID, it.Quantity))
	}
	consignment := entity.NewConsignment(userID, lembaga.ID, req.EventName, form.startDate, form.endDate,
		photoKey, req.Notes, items)
	// request_id diisi repository setelah penitipan tersimpan (id-nya belum ada di sini).
	activity := entity.NewUserLog(userID, enums.RequestTypeConsignment, 0, enums.ActivitySubmitted)

	if err := s.repo.Create(ctx, consignment, activity); err != nil {
		photo.Delete(ctx, s.storage, photoKey)
		return nil, err
	}

	// Ambil ulang dari database supaya penitip, lembaga & barang ikut terisi (tanpa timeline).
	return s.findResponse(ctx, consignment.ID)
}

// findResponse mengambil satu penitipan lalu mengubahnya menjadi response, termasuk URL foto dari storage.
func (s *consignmentService) findResponse(ctx context.Context, id uint) (*dto.ConsignmentResponse, error) {
	consignment, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	res := dto.NewConsignmentResponse(consignment)
	res.PhotoURL = s.storage.URL(consignment.Photo)
	return &res, nil
}

// consignmentForm adalah hasil validasi form (tanggal sudah di-parse).
type consignmentForm struct {
	startDate time.Time
	endDate   time.Time
}

// validateCreateRequest mengecek aturan yang membandingkan beberapa field sekaligus, tanpa menyentuh database.
// Aturannya sama dengan pengajuan peminjaman. today adalah tanggal hari ini (WIB), dijadikan parameter
// agar mudah diuji.
func validateCreateRequest(req dto.CreateConsignmentRequest, today time.Time) (*consignmentForm, error) {
	// 1. Nama barang yang sama tidak boleh ditulis dua kali (tidak membedakan huruf besar/kecil).
	firstPosition := make(map[string]int, len(req.Items)) // nama → urutan pertama kali muncul
	for i, it := range req.Items {
		position := i + 1 // urutan untuk pesan error dimulai dari 1
		name := strings.ToLower(strings.TrimSpace(it.Name))
		if first, ok := firstPosition[name]; ok {
			return nil, errx.NewDuplicateConsignmentItem(position, first)
		}
		firstPosition[name] = position
	}

	// 2. Tanggal. Formatnya sudah dicek tag `datetime`, jadi gagal parse di sini hanya jika service
	// dipanggil tanpa lewat handler.
	startDate, err := time.Parse(dto.DateLayout, req.StartDate)
	if err != nil {
		return nil, errx.ErrInvalidStartDate
	}
	endDate, err := time.Parse(dto.DateLayout, req.EndDate)
	if err != nil {
		return nil, errx.ErrInvalidEndDate
	}
	if startDate.Before(today) {
		return nil, errx.ErrStartDatePast
	}
	if !endDate.After(startDate) { // hari yang sama juga ditolak
		return nil, errx.ErrEndNotAfterStart
	}
	if endDate.After(startDate.AddDate(0, 0, maxConsignmentDays)) {
		return nil, errx.NewConsignmentTooLong(maxConsignmentDays)
	}

	return &consignmentForm{startDate: startDate, endDate: endDate}, nil
}
