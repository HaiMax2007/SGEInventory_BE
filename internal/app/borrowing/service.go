package borrowing

import (
	"context"
	"slices"
	"time"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
	"SGEInventory_BE/helpers/dates"
	"SGEInventory_BE/helpers/pagination"
)

// maxBorrowDays: tanggal kembali paling lambat tanggal ambil + 7 hari.
// Batas lain (1-20 barang, tujuan 500 karakter) ada di tag `validate` dto.CreateBorrowingRequest.
const maxBorrowDays = 7

type borrowingService struct {
	repo        contracts.BorrowingRepository
	logRepo     contracts.UserLogRepository
	itemRepo    contracts.ItemRepository
	lembagaRepo contracts.LembagaRepository
	tx          contracts.Transactor  // menjalankan beberapa repository dalam satu transaksi (ACC/REJECT)
	storage     contracts.FileStorage // tempat menyimpan foto pengembalian
}

// NewBorrowingService membuat service peminjaman. Repository dioper dari main.go (dependency injection).
func NewBorrowingService(
	repo contracts.BorrowingRepository,
	logRepo contracts.UserLogRepository,
	itemRepo contracts.ItemRepository,
	lembagaRepo contracts.LembagaRepository,
	tx contracts.Transactor,
	storage contracts.FileStorage,
) contracts.BorrowingService {
	return &borrowingService{
		repo: repo, logRepo: logRepo, itemRepo: itemRepo, lembagaRepo: lembagaRepo, tx: tx, storage: storage,
	}
}

// CreateBorrowing menjalankan alur submit pengajuan:
//
//  1. cek aturan antar-field (tanpa database). Aturan per field sudah dicek handler lewat tag `validate`.
//  2. lembaga harus ada
//  3. barang harus ada & stoknya cukup
//  4. simpan borrowings + borrowing_items + log_users dalam satu transaksi
//
// Stok TIDAK dikurangi di sini; stok berkurang saat admin menyetujui (ACC).
func (s *borrowingService) CreateBorrowing(ctx context.Context, userID uint, req dto.CreateBorrowingRequest) (*dto.BorrowingResponse, error) {
	form, err := validateCreateRequest(req, dates.TodayWIB())
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

	// Ambil ulang dari database supaya peminjam, lembaga & barang ikut terisi.
	return s.findResponse(ctx, borrowing.ID)
}

func (s *borrowingService) GetMyBorrowings(ctx context.Context, userID uint, query dto.BorrowingListQuery) ([]dto.BorrowingResponse, *pagination.Pagination, error) {
	page := pagination.New(query.Page, query.Limit)

	borrowings, total, err := s.repo.FindByUser(ctx, userID, page.Offset(), page.Limit)
	if err != nil {
		return nil, nil, err
	}
	page.SetTotal(total)
	return s.toBorrowingResponses(borrowings), page, nil
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
	return s.toBorrowingResponses(borrowings), page, nil
}

// CountBorrowings mengembalikan jumlah semua pengajuan peminjaman untuk admin.
func (s *borrowingService) CountBorrowings(ctx context.Context) (*dto.CountResponse, error) {
	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, err
	}
	return &dto.CountResponse{Total: total}, nil
}

// toBorrowingResponses mengubah daftar entity menjadi daftar response (tanpa timeline).
func (s *borrowingService) toBorrowingResponses(borrowings []entity.Borrowing) []dto.BorrowingResponse {
	res := make([]dto.BorrowingResponse, 0, len(borrowings))
	for i := range borrowings {
		res = append(res, s.toResponse(&borrowings[i]))
	}
	return res
}

// findResponse mengambil satu pengajuan (lengkap dengan peminjam, lembaga & barang) lalu mengubahnya menjadi
// response TANPA timeline. Dipakai sebagai balasan setelah aksi: buat pengajuan, ACC/tolak oleh admin,
// unggah/ACC/tolak pengembalian. Timeline hanya ada di GET /api/borrowings/:id (GetBorrowing).
func (s *borrowingService) findResponse(ctx context.Context, id uint) (*dto.BorrowingResponse, error) {
	borrowing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	res := s.toResponse(borrowing)
	return &res, nil
}

// toResponse = dto.NewBorrowingResponse dengan URL foto pengembalian dari storage yang dipakai.
// Database hanya menyimpan key (mis. "returns/8f3a...jpg"), storage yang tahu alamat publiknya.
func (s *borrowingService) toResponse(b *entity.Borrowing) dto.BorrowingResponse {
	return dto.NewBorrowingResponse(b, s.storage.URL)
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

	res := s.toResponse(borrowing)
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
		// Barang titipan user tidak bisa dipinjam: diperlakukan sama seperti barang yang tidak ada.
		if found[i].IsConsignment() {
			continue
		}
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
		if !item.CanLend(it.Quantity) {
			shortages = append(shortages, entity.LendShortage(item.Name, it.Quantity, item.AvailableQuantity))
			continue
		}
		items = append(items, entity.NewBorrowingItem(item.ID, item.Name, it.Quantity))
	}
	if len(shortages) > 0 {
		return nil, errx.NewInsufficientStock(shortages)
	}
	return items, nil
}

// borrowingForm adalah hasil validasi form (tanggal sudah di-parse).
type borrowingForm struct {
	purpose    string
	pickupDate time.Time
	returnDate time.Time
}

// validateCreateRequest mengecek aturan yang membandingkan beberapa field sekaligus, tanpa menyentuh database.
// Aturan per field (wajib diisi, 1-20 barang, jumlah > 0, tujuan maks 500 karakter, format tanggal) sudah dicek
// lebih dulu di handler lewat tag `validate` (validator.ValidateStruct), yang juga sudah men-trim teks.
// today adalah tanggal hari ini (WIB), dijadikan parameter agar mudah diuji.
func validateCreateRequest(req dto.CreateBorrowingRequest, today time.Time) (*borrowingForm, error) {
	// 1. Barang yang sama tidak boleh dipilih dua kali.
	firstPosition := make(map[uint]int, len(req.Items)) // itemId → urutan pertama kali muncul
	for i, it := range req.Items {
		position := i + 1 // urutan untuk pesan error dimulai dari 1
		if first, ok := firstPosition[it.ItemID]; ok {
			return nil, errx.NewDuplicateBorrowItem(position, first)
		}
		firstPosition[it.ItemID] = position
	}

	// 2. Tanggal. Formatnya sudah dicek tag `datetime`, jadi gagal parse di sini hanya jika service
	// dipanggil tanpa lewat handler.
	pickupDate, err := time.Parse(dto.DateLayout, req.PickUpDate)
	if err != nil {
		return nil, errx.ErrInvalidPickupDate
	}
	returnDate, err := time.Parse(dto.DateLayout, req.ReturnDate)
	if err != nil {
		return nil, errx.ErrInvalidReturnDate
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

	return &borrowingForm{purpose: req.Purpose, pickupDate: pickupDate, returnDate: returnDate}, nil
}
