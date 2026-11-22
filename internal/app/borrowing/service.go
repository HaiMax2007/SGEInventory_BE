package borrowing

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
	"SGEInventory_BE/helpers/pagination"
)

// Batasan form peminjaman.
const (
	maxBorrowItems   = 20  // jenis barang per pengajuan
	maxPurposeLength = 500 // karakter
	maxBorrowDays    = 7   // tanggal kembali paling lambat tanggal ambil + 7 hari
)

// wib dipakai untuk menentukan "hari ini" (UTC+7). FixedZone tidak butuh data zona waktu di container.
var wib = time.FixedZone("WIB", 7*60*60)

type borrowingService struct {
	repo        contracts.BorrowingRepository
	logRepo     contracts.UserLogRepository
	itemRepo    contracts.ItemRepository
	lembagaRepo contracts.LembagaRepository
}

// NewBorrowingService membuat service peminjaman. Repository dioper dari main.go (dependency injection).
func NewBorrowingService(
	repo contracts.BorrowingRepository,
	logRepo contracts.UserLogRepository,
	itemRepo contracts.ItemRepository,
	lembagaRepo contracts.LembagaRepository,
) contracts.BorrowingService {
	return &borrowingService{repo: repo, logRepo: logRepo, itemRepo: itemRepo, lembagaRepo: lembagaRepo}
}

// CreateBorrowing menjalankan alur submit pengajuan:
//
//  1. validasi isi form (tanpa database): items, lembaga, tujuan, tanggal
//  2. lembaga harus ada
//  3. barang harus ada & stoknya cukup
//  4. simpan borrowings + borrowing_items + log_users dalam satu transaksi
//
// Stok TIDAK dikurangi di sini; stok berkurang saat admin menyetujui (ACC).
func (s *borrowingService) CreateBorrowing(ctx context.Context, userID uint, req dto.CreateBorrowingRequest) (*dto.BorrowingResponse, error) {
	form, err := validateCreateRequest(req, todayWIB())
	if err != nil {
		return nil, err
	}

	// Lembaga harus ada dan belum dihapus.
	lembaga, err := s.lembagaRepo.FindByID(ctx, req.LembagaID)
	if err != nil {
		return nil, err // errx.ErrLembagaNotFound
	}

	items, err := s.buildBorrowingItems(ctx, req.Items)
	if err != nil {
		return nil, err
	}

	borrowing := entity.NewBorrowing(userID, lembaga.ID, form.purpose, form.pickupDate, form.returnDate, items)
	// request_id diisi repository setelah pengajuan tersimpan (id-nya belum ada di sini).
	activity := entity.NewUserLog(userID, enums.RequestTypeBorrowing, 0, enums.ActivitySubmitted)
	if err := s.repo.Create(ctx, borrowing, activity); err != nil {
		return nil, err
	}

	// Ambil ulang dari database supaya peminjam, lembaga & barang ikut terisi
	// (bentuk respons sama persis dengan GET /api/borrowings/:id, tanpa timeline).
	saved, err := s.repo.FindByID(ctx, borrowing.ID)
	if err != nil {
		return nil, err
	}
	res := dto.NewBorrowingResponse(saved)
	return &res, nil
}

func (s *borrowingService) GetMyBorrowings(ctx context.Context, userID uint, query dto.BorrowingListQuery) ([]dto.BorrowingResponse, *pagination.Pagination, error) {
	page := pagination.New(query.Page, query.Limit)

	borrowings, total, err := s.repo.FindByUser(ctx, userID, page.Offset(), page.Limit)
	if err != nil {
		return nil, nil, err
	}
	page.SetTotal(total)
	return toBorrowingResponses(borrowings), page, nil
}

// GetBorrowings mengambil semua pengajuan untuk admin. Filter status kosong = semua status.
func (s *borrowingService) GetBorrowings(ctx context.Context, query dto.BorrowingListQuery) ([]dto.BorrowingResponse, *pagination.Pagination, error) {
	// Status yang tidak dikenal ditolak (400), bukan dibalas daftar kosong,
	// supaya salah ketik di FE (mis. "pending" huruf kecil) langsung ketahuan.
	if query.Status != "" && !slices.Contains(enums.BorrowingStatuses, query.Status) {
		return nil, nil, errx.ErrInvalidBorrowingStatus
	}

	page := pagination.New(query.Page, query.Limit)
	borrowings, total, err := s.repo.FindAll(ctx, query.Status, page.Offset(), page.Limit)
	if err != nil {
		return nil, nil, err
	}
	page.SetTotal(total)
	return toBorrowingResponses(borrowings), page, nil
}

// toBorrowingResponses mengubah daftar entity menjadi daftar response (tanpa timeline).
func toBorrowingResponses(borrowings []entity.Borrowing) []dto.BorrowingResponse {
	res := make([]dto.BorrowingResponse, 0, len(borrowings))
	for i := range borrowings {
		res = append(res, dto.NewBorrowingResponse(&borrowings[i]))
	}
	return res
}

// GetBorrowing mengambil detail pengajuan + timeline (semua log pengajuan ini, termasuk aksi admin).
func (s *borrowingService) GetBorrowing(ctx context.Context, id, userID uint, isAdmin bool) (*dto.BorrowingResponse, error) {
	borrowing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Pengajuan orang lain dibalas "tidak ditemukan" (bukan 403), supaya user tidak bisa menebak
	// id mana saja yang ada.
	if !isAdmin && borrowing.UserID != userID {
		return nil, errx.ErrBorrowingNotFound
	}

	// Timeline diambil berdasarkan PENGAJUAN (request_type + request_id), bukan berdasarkan user_id log.
	// Karena itu aksi admin (user_id log = admin) tetap terlihat oleh peminjam.
	logs, err := s.logRepo.FindByRequest(ctx, enums.RequestTypeBorrowing, borrowing.ID)
	if err != nil {
		return nil, err
	}

	res := dto.NewBorrowingResponse(borrowing)
	res.Timeline = make([]dto.UserLogResponse, 0, len(logs))
	for i := range logs {
		res.Timeline = append(res.Timeline, dto.NewUserLogResponse(&logs[i]))
	}
	return &res, nil
}

// buildBorrowingItems memastikan semua barang ada & stoknya cukup, lalu membuat baris borrowing_items.
// Nama barang diambil dari database (bukan dari FE) dan disalin ke borrowing_items.item_name.
//
// Cek stok di sini hanya untuk memberi tahu user lebih awal. Pengecekan yang menentukan dilakukan
// lagi saat admin ACC (dengan baris barang dikunci), karena stok bisa berubah sejak pengajuan dikirim.
func (s *borrowingService) buildBorrowingItems(ctx context.Context, reqItems []dto.BorrowingItemRequest) ([]entity.BorrowingItem, error) {
	ids := make([]uint, 0, len(reqItems))
	for _, it := range reqItems {
		ids = append(ids, it.ItemID)
	}

	found, err := s.itemRepo.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[uint]*entity.Item, len(found))
	for i := range found {
		byID[found[i].ID] = &found[i]
	}

	var missing []uint
	for _, it := range reqItems {
		if _, ok := byID[it.ItemID]; !ok {
			missing = append(missing, it.ItemID)
		}
	}
	if len(missing) > 0 {
		return nil, errx.NewBorrowItemsNotFound(missing)
	}

	// Kumpulkan SEMUA barang yang stoknya kurang, supaya user bisa memperbaiki semuanya sekaligus.
	var shortages []string
	items := make([]entity.BorrowingItem, 0, len(reqItems))
	for _, it := range reqItems {
		item := byID[it.ItemID]
		if it.Quantity > item.AvailableQuantity {
			shortages = append(shortages,
				fmt.Sprintf("%s (diminta %d, tersedia %d)", item.Name, it.Quantity, item.AvailableQuantity))
			continue
		}
		items = append(items, entity.NewBorrowingItem(item.ID, item.Name, it.Quantity))
	}
	if len(shortages) > 0 {
		return nil, errx.NewInsufficientStock(shortages)
	}
	return items, nil
}

// borrowingForm adalah hasil validasi form yang sudah dirapikan (tujuan di-trim, tanggal sudah di-parse).
type borrowingForm struct {
	purpose    string
	pickupDate time.Time
	returnDate time.Time
}

// validateCreateRequest mengecek isi form tanpa menyentuh database.
// Setiap aturan mengembalikan error sendiri (lihat domain/errx/borrowing_errx.go).
// today adalah tanggal hari ini (WIB), dijadikan parameter agar mudah diuji.
func validateCreateRequest(req dto.CreateBorrowingRequest, today time.Time) (*borrowingForm, error) {
	// 1. Barang
	if len(req.Items) == 0 {
		return nil, errx.ErrBorrowItemsRequired
	}
	if len(req.Items) > maxBorrowItems {
		return nil, errx.NewBorrowTooManyItems(maxBorrowItems)
	}
	firstPosition := make(map[uint]int, len(req.Items)) // itemId → urutan pertama kali muncul
	for i, it := range req.Items {
		position := i + 1 // urutan untuk pesan error dimulai dari 1
		if it.ItemID == 0 {
			return nil, errx.NewBorrowItemIDRequired(position)
		}
		if it.Quantity <= 0 {
			return nil, errx.NewBorrowQuantityInvalid(position)
		}
		if first, ok := firstPosition[it.ItemID]; ok {
			return nil, errx.NewDuplicateBorrowItem(position, first)
		}
		firstPosition[it.ItemID] = position
	}

	// 2. Lembaga (keberadaannya dicek di database setelah ini)
	if req.LembagaID == 0 {
		return nil, errx.ErrBorrowLembagaRequired
	}

	// 3. Tujuan: spasi di awal/akhir dibuang, jadi "   " dianggap kosong.
	purpose := strings.TrimSpace(req.Purpose)
	if purpose == "" {
		return nil, errx.ErrPurposeRequired
	}
	// Hitung karakter (rune), bukan byte, agar huruf non-ASCII tidak dihitung lebih dari satu.
	if utf8.RuneCountInString(purpose) > maxPurposeLength {
		return nil, errx.NewPurposeTooLong(maxPurposeLength)
	}

	// 4. Tanggal
	pickupDate, err := parseDate(req.PickUpDate, errx.ErrPickupDateRequired, errx.ErrPickupDateInvalid)
	if err != nil {
		return nil, err
	}
	returnDate, err := parseDate(req.ReturnDate, errx.ErrReturnDateRequired, errx.ErrReturnDateInvalid)
	if err != nil {
		return nil, err
	}
	if pickupDate.Before(today) {
		return nil, errx.ErrPickupDatePast
	}
	if !returnDate.After(pickupDate) { // hari yang sama juga ditolak
		return nil, errx.ErrReturnNotAfterPickup
	}
	if returnDate.After(pickupDate.AddDate(0, 0, maxBorrowDays)) {
		return nil, errx.NewBorrowTooLong(maxBorrowDays)
	}

	return &borrowingForm{purpose: purpose, pickupDate: pickupDate, returnDate: returnDate}, nil
}

// parseDate mengubah "YYYY-MM-DD" menjadi time.Time (jam 00:00 UTC).
func parseDate(value string, errRequired, errInvalid *errx.DomainError) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, errRequired
	}
	date, err := time.Parse(dto.DateLayout, value)
	if err != nil {
		return time.Time{}, errInvalid
	}
	return date, nil
}

// todayWIB mengembalikan tanggal hari ini menurut WIB, dalam bentuk yang sama dengan parseDate
// (jam 00:00 UTC) supaya bisa dibandingkan langsung.
func todayWIB() time.Time {
	now := time.Now().In(wib)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}
