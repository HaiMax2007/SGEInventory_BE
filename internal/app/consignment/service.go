// Package consignment berisi fitur penitipan barang: user menitipkan barang miliknya ke admin.
package consignment

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
	"SGEInventory_BE/helpers/dates"
	"SGEInventory_BE/helpers/pagination"
	"SGEInventory_BE/pkg/photo"
)

// maxConsignmentDays: tanggal selesai paling lambat tanggal mulai + 7 hari (sama dengan peminjaman).
// Batas lain (1-20 barang, panjang teks) ada di tag `validate` dto.CreateConsignmentRequest.
const maxConsignmentDays = 7

// photoFolder: folder key foto penitipan, mis. "consignments/8f3a...jpg".
const photoFolder = "consignments"

type consignmentService struct {
	repo        contracts.ConsignmentRepository
	logRepo     contracts.UserLogRepository // log ACC/tolak (log submit disimpan repo.Create)
	lembagaRepo contracts.LembagaRepository
	tx          contracts.Transactor  // menjalankan beberapa repository dalam satu transaksi (ACC/tolak)
	storage     contracts.FileStorage // tempat menyimpan foto barang titipan
}

// NewConsignmentService membuat service penitipan. Repository dioper dari main.go (dependency injection).
func NewConsignmentService(
	repo contracts.ConsignmentRepository,
	logRepo contracts.UserLogRepository,
	lembagaRepo contracts.LembagaRepository,
	tx contracts.Transactor,
	storage contracts.FileStorage,
) contracts.ConsignmentService {
	return &consignmentService{repo: repo, logRepo: logRepo, lembagaRepo: lembagaRepo, tx: tx, storage: storage}
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

// GetMyConsignments: daftar penitipan milik user yang login (terbaru dulu).
func (s *consignmentService) GetMyConsignments(ctx context.Context, userID uint, query dto.ConsignmentListQuery) ([]dto.ConsignmentResponse, *pagination.Pagination, error) {
	page := pagination.New(query.Page, query.Limit)
	consignments, total, err := s.repo.FindByUser(ctx, userID, page.Offset(), page.Limit)
	if err != nil {
		return nil, nil, err
	}
	page.SetTotal(total)
	return s.toResponses(consignments), page, nil
}

// GetConsignments: daftar SEMUA penitipan untuk admin. Filter status kosong = semua status.
func (s *consignmentService) GetConsignments(ctx context.Context, query dto.ConsignmentListQuery) ([]dto.ConsignmentResponse, *pagination.Pagination, error) {
	// Status yang tidak dikenal ditolak (400), bukan dibalas daftar kosong,
	// supaya salah ketik di FE (mis. "pending" huruf kecil) langsung ketahuan.
	if query.Status != "" && !slices.Contains(enums.ConsignmentStatuses, query.Status) {
		return nil, nil, errx.ErrInvalidConsignmentStatus
	}

	page := pagination.New(query.Page, query.Limit)
	consignments, total, err := s.repo.FindAll(ctx, query.Status, page.Offset(), page.Limit)
	if err != nil {
		return nil, nil, err
	}
	page.SetTotal(total)
	return s.toResponses(consignments), page, nil
}

// CountConsignments mengembalikan jumlah semua penitipan untuk admin.
func (s *consignmentService) CountConsignments(ctx context.Context) (*dto.CountResponse, error) {
	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, err
	}
	return &dto.CountResponse{Total: total}, nil
}

// GetConsignment: detail satu penitipan (tanpa timeline). Penitipan orang lain dibalas "tidak ditemukan"
// (bukan 403), supaya user tidak bisa menebak id mana saja yang ada — sama seperti detail peminjaman.
func (s *consignmentService) GetConsignment(ctx context.Context, id, userID uint, isAdmin bool) (*dto.ConsignmentResponse, error) {
	consignment, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !isAdmin && consignment.UserID != userID {
		return nil, errx.ErrConsignmentNotFound
	}
	res := s.toResponse(consignment)
	return &res, nil
}

// findResponse mengambil satu penitipan lalu mengubahnya menjadi response.
func (s *consignmentService) findResponse(ctx context.Context, id uint) (*dto.ConsignmentResponse, error) {
	consignment, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	res := s.toResponse(consignment)
	return &res, nil
}

func (s *consignmentService) toResponses(consignments []entity.Consignment) []dto.ConsignmentResponse {
	res := make([]dto.ConsignmentResponse, 0, len(consignments))
	for i := range consignments {
		res = append(res, s.toResponse(&consignments[i]))
	}
	return res
}

// toResponse = dto.NewConsignmentResponse dengan URL foto dari storage yang dipakai (Cloudinary / folder lokal).
// Penitipan yang ditolak tidak punya foto lagi → photoUrl null.
func (s *consignmentService) toResponse(c *entity.Consignment) dto.ConsignmentResponse {
	return dto.NewConsignmentResponse(c, s.storage.URL)
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
