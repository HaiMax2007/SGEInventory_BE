package unit

// File ini berisi "database palsu" di memori + repository palsu yang memenuhi interface di
// domain/contracts. Service diuji memakai ini, jadi unit test tidak butuh PostgreSQL/Docker.
//
// Aturan penting: repository palsu selalu mengembalikan SALINAN data (seperti GORM yang membaca ulang
// dari database). Kalau service mengubah data tapi lupa memanggil Update, perubahan itu tidak tersimpan,
// sama seperti di database sungguhan.

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"

	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
)

// ===================================================================================================
// fakeDB: isi "tabel" di memori, dipakai bersama oleh semua repository palsu
// ===================================================================================================

type fakeDB struct {
	nextID     uint
	users      map[uint]*entity.User
	lembagas   map[uint]*entity.Lembaga
	items      map[uint]*entity.Item
	borrowings map[uint]*entity.Borrowing
	logs       []entity.UserLog

	// calls mencatat langkah penting secara berurutan (mis. "lock items", "lock borrowing"),
	// dipakai untuk menguji urutan penguncian.
	calls []string
	// afterLockItems (opsional) dijalankan tepat setelah barang dikunci, untuk mensimulasikan
	// admin lain yang memproses pengajuan di saat yang sama.
	afterLockItems func()
}

func newFakeDB() *fakeDB {
	return &fakeDB{
		users:      map[uint]*entity.User{},
		lembagas:   map[uint]*entity.Lembaga{},
		items:      map[uint]*entity.Item{},
		borrowings: map[uint]*entity.Borrowing{},
	}
}

func (db *fakeDB) newID() uint {
	db.nextID++
	return db.nextID
}

func (db *fakeDB) record(call string) {
	db.calls = append(db.calls, call)
}

// clone membuat salinan seluruh isi tabel (untuk ROLLBACK di fakeTransactor).
func (db *fakeDB) clone() *fakeDB {
	return &fakeDB{
		nextID:     db.nextID,
		users:      cloneMap(db.users, copyUser),
		lembagas:   cloneMap(db.lembagas, copyLembaga),
		items:      cloneMap(db.items, copyItem),
		borrowings: cloneMap(db.borrowings, copyBorrowing),
		logs:       slices.Clone(db.logs),
	}
}

// restoreFrom mengembalikan isi tabel dari salinan (calls & hook tidak ikut dikembalikan).
func (db *fakeDB) restoreFrom(s *fakeDB) {
	db.nextID = s.nextID
	db.users = s.users
	db.lembagas = s.lembagas
	db.items = s.items
	db.borrowings = s.borrowings
	db.logs = s.logs
}

func (db *fakeDB) insertLog(l *entity.UserLog) {
	l.ID = db.newID()
	l.CreatedAt = time.Now()
	db.logs = append(db.logs, *l)
}

// logsFor mengembalikan log sebuah pengajuan peminjaman, urut dari yang paling lama.
func (db *fakeDB) logsFor(borrowingID uint) []entity.UserLog {
	var res []entity.UserLog
	for _, l := range db.logs {
		if l.RequestType == enums.RequestTypeBorrowing && l.RequestID == borrowingID {
			res = append(res, l)
		}
	}
	return res
}

// ---------- data awal untuk tes ----------

func (db *fakeDB) addUser(name, role string) *entity.User {
	id := db.newID()
	u := entity.NewUser(name, fmt.Sprintf("%08d", id), strings.ToLower(name)+"@example.com", "hash", "")
	u.ID = id
	u.Role = role
	db.users[id] = u
	return copyUser(u)
}

func (db *fakeDB) addLembaga(name string) *entity.Lembaga {
	l := entity.NewLembaga(name)
	l.ID = db.newID()
	db.lembagas[l.ID] = l
	return copyLembaga(l)
}

// addItem membuat barang dengan total & sisa tertentu (sisa < total = sebagian sedang dipinjam).
func (db *fakeDB) addItem(name string, total, available int, lembagaID uint) *entity.Item {
	i := entity.NewItem(name, "", lembagaID, total)
	i.ID = db.newID()
	i.AvailableQuantity = available
	db.items[i.ID] = i
	return copyItem(i)
}

func (db *fakeDB) softDeleteItem(id uint) {
	db.items[id].DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
}

// line adalah satu barang di pengajuan: itemID × qty.
type line struct {
	itemID uint
	qty    int
}

// addBorrowing membuat pengajuan dengan status tertentu + log SUBMITTED dari pemiliknya.
func (db *fakeDB) addBorrowing(userID, lembagaID uint, status string, lines ...line) *entity.Borrowing {
	items := make([]entity.BorrowingItem, 0, len(lines))
	for _, ln := range lines {
		bi := entity.NewBorrowingItem(ln.itemID, db.items[ln.itemID].Name, ln.qty)
		items = append(items, bi)
	}
	pickup := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	b := entity.NewBorrowing(userID, lembagaID, "Kegiatan", pickup, pickup.AddDate(0, 0, 2), items)
	b.ID = db.newID()
	b.Status = status
	b.CreatedAt = time.Now()
	for i := range b.Items {
		b.Items[i].ID = db.newID()
		b.Items[i].BorrowingID = b.ID
	}
	db.borrowings[b.ID] = b
	db.insertLog(entity.NewUserLog(userID, enums.RequestTypeBorrowing, b.ID, enums.ActivitySubmitted))
	return copyBorrowing(b)
}

// ---------- salinan ----------

func cloneMap[T any](m map[uint]*T, cp func(*T) *T) map[uint]*T {
	res := make(map[uint]*T, len(m))
	for id, v := range m {
		res[id] = cp(v)
	}
	return res
}

func copyUser(u *entity.User) *entity.User { c := *u; return &c }

func copyLembaga(l *entity.Lembaga) *entity.Lembaga { c := *l; c.Items = nil; return &c }

func copyItem(i *entity.Item) *entity.Item { c := *i; return &c }

func copyBorrowing(b *entity.Borrowing) *entity.Borrowing {
	c := *b
	c.Items = slices.Clone(b.Items)
	return &c
}

// sortedIDs mengembalikan id dari sebuah "tabel", urut naik.
func sortedIDs[T any](m map[uint]*T) []uint {
	return slices.Sorted(maps.Keys(m))
}

// pageOf mengambil satu halaman dari hasil query (OFFSET/LIMIT).
func pageOf[T any](rows []T, offset, limit int) []T {
	if offset >= len(rows) {
		return []T{}
	}
	return rows[offset:min(offset+limit, len(rows))]
}

// ===================================================================================================
// fakeTransactor: memenuhi contracts.Transactor
// ===================================================================================================

type txMarker struct{}

// fakeTransactor menyalin isi fakeDB sebelum transaksi dimulai, dan mengembalikannya jika fn error
// (meniru ROLLBACK). Transaksi di dalam transaksi tidak membuka transaksi baru (sama seperti aslinya).
type fakeTransactor struct {
	db      *fakeDB
	started int // jumlah transaksi "sungguhan" yang dibuka
}

func (t *fakeTransactor) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if ctx.Value(txMarker{}) != nil {
		return fn(ctx)
	}
	t.started++
	before := t.db.clone()
	if err := fn(context.WithValue(ctx, txMarker{}, true)); err != nil {
		t.db.restoreFrom(before)
		return err
	}
	return nil
}

// ===================================================================================================
// fakeUserRepo: memenuhi contracts.UserRepository
// ===================================================================================================

type fakeUserRepo struct{ db *fakeDB }

func (r *fakeUserRepo) Create(_ context.Context, u *entity.User) error {
	for _, x := range r.db.users {
		if x.Email == u.Email {
			return errx.ErrEmailTaken
		}
		if x.NIM == u.NIM {
			return errx.ErrNIMTaken
		}
	}
	u.ID = r.db.newID()
	u.CreatedAt = time.Now()
	r.db.users[u.ID] = copyUser(u)
	return nil
}

func (r *fakeUserRepo) FindByID(_ context.Context, id uint) (*entity.User, error) {
	u, ok := r.db.users[id]
	if !ok || u.DeletedAt.Valid {
		return nil, errx.ErrUserNotFound
	}
	return copyUser(u), nil
}

func (r *fakeUserRepo) FindByEmail(_ context.Context, email string) (*entity.User, error) {
	for _, u := range r.db.users {
		if u.Email == email && !u.DeletedAt.Valid {
			return copyUser(u), nil
		}
	}
	return nil, errx.ErrUserNotFound
}

func (r *fakeUserRepo) FindAll(_ context.Context, search, role string, offset, limit int) ([]entity.User, int64, error) {
	var rows []entity.User
	for _, id := range slices.Backward(sortedIDs(r.db.users)) { // terbaru dulu
		u := r.db.users[id]
		if u.DeletedAt.Valid || (role != "" && u.Role != role) {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(u.FullName+u.Email), strings.ToLower(search)) {
			continue
		}
		rows = append(rows, *u)
	}
	return pageOf(rows, offset, limit), int64(len(rows)), nil
}

func (r *fakeUserRepo) Update(_ context.Context, u *entity.User) error {
	for _, x := range r.db.users {
		if x.ID != u.ID && x.Email == u.Email {
			return errx.ErrEmailTaken
		}
	}
	r.db.users[u.ID] = copyUser(u)
	return nil
}

func (r *fakeUserRepo) Delete(_ context.Context, id uint) error {
	u, ok := r.db.users[id]
	if !ok || u.DeletedAt.Valid {
		return errx.ErrUserNotFound
	}
	u.DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
	return nil
}

// ===================================================================================================
// fakeLembagaRepo: memenuhi contracts.LembagaRepository
// ===================================================================================================

type fakeLembagaRepo struct{ db *fakeDB }

func (r *fakeLembagaRepo) Create(_ context.Context, l *entity.Lembaga) error {
	for _, x := range r.db.lembagas {
		if x.Name == l.Name {
			return errx.ErrLembagaNameTaken // termasuk lembaga yang sudah dihapus (sama seperti index unik)
		}
	}
	l.ID = r.db.newID()
	r.db.lembagas[l.ID] = copyLembaga(l)
	return nil
}

func (r *fakeLembagaRepo) FindByID(_ context.Context, id uint) (*entity.Lembaga, error) {
	l, ok := r.db.lembagas[id]
	if !ok || l.DeletedAt.Valid {
		return nil, errx.ErrLembagaNotFound
	}
	return copyLembaga(l), nil
}

func (r *fakeLembagaRepo) FindAll(_ context.Context, search string, offset, limit int) ([]entity.Lembaga, int64, error) {
	var rows []entity.Lembaga
	for _, l := range r.db.lembagas {
		if l.DeletedAt.Valid || !strings.Contains(strings.ToLower(l.Name), strings.ToLower(search)) {
			continue
		}
		rows = append(rows, *copyLembaga(l))
	}
	slices.SortFunc(rows, func(a, b entity.Lembaga) int { return strings.Compare(a.Name, b.Name) })
	return pageOf(rows, offset, limit), int64(len(rows)), nil
}

func (r *fakeLembagaRepo) Update(_ context.Context, l *entity.Lembaga) error {
	for _, x := range r.db.lembagas {
		if x.ID != l.ID && x.Name == l.Name {
			return errx.ErrLembagaNameTaken
		}
	}
	r.db.lembagas[l.ID] = copyLembaga(l)
	return nil
}

func (r *fakeLembagaRepo) Delete(_ context.Context, l *entity.Lembaga) error {
	r.db.lembagas[l.ID].DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
	return nil
}

func (r *fakeLembagaRepo) CountItems(_ context.Context, id uint) (int64, error) {
	var n int64
	for _, i := range r.db.items {
		if i.LembagaID == id && !i.DeletedAt.Valid {
			n++
		}
	}
	return n, nil
}

// ===================================================================================================
// fakeItemRepo: memenuhi contracts.ItemRepository
// ===================================================================================================

type fakeItemRepo struct{ db *fakeDB }

func (r *fakeItemRepo) find(id uint) (*entity.Item, bool) {
	i, ok := r.db.items[id]
	if !ok || i.DeletedAt.Valid {
		return nil, false
	}
	return copyItem(i), true
}

func (r *fakeItemRepo) withLembaga(i *entity.Item) *entity.Item {
	if l, ok := r.db.lembagas[i.LembagaID]; ok {
		i.Lembaga = copyLembaga(l)
	}
	return i
}

func (r *fakeItemRepo) Create(_ context.Context, i *entity.Item) error {
	i.ID = r.db.newID()
	r.db.items[i.ID] = copyItem(i)
	return nil
}

func (r *fakeItemRepo) FindByID(_ context.Context, id uint) (*entity.Item, error) {
	i, ok := r.find(id)
	if !ok {
		return nil, errx.ErrItemNotFound
	}
	return r.withLembaga(i), nil
}

func (r *fakeItemRepo) FindByIDs(_ context.Context, ids []uint) ([]entity.Item, error) {
	res := []entity.Item{}
	for _, id := range slices.Sorted(slices.Values(ids)) {
		if i, ok := r.find(id); ok {
			res = append(res, *i)
		}
	}
	return res, nil
}

func (r *fakeItemRepo) FindAll(_ context.Context, search string, onlyAvailable bool, offset, limit int) ([]entity.Item, int64, error) {
	var rows []entity.Item
	for _, id := range slices.Backward(sortedIDs(r.db.items)) {
		i, ok := r.find(id)
		if !ok || (onlyAvailable && i.AvailableQuantity == 0) {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(i.Name+i.Description), strings.ToLower(search)) {
			continue
		}
		rows = append(rows, *r.withLembaga(i))
	}
	return pageOf(rows, offset, limit), int64(len(rows)), nil
}

func (r *fakeItemRepo) Update(_ context.Context, i *entity.Item) error {
	stored, ok := r.db.items[i.ID]
	if !ok {
		return errx.ErrItemNotFound
	}
	c := copyItem(i)
	c.Lembaga = nil
	c.DeletedAt = stored.DeletedAt
	r.db.items[i.ID] = c
	return nil
}

func (r *fakeItemRepo) Delete(_ context.Context, i *entity.Item) error {
	r.db.softDeleteItem(i.ID)
	return nil
}

func (r *fakeItemRepo) FindByIDForUpdate(_ context.Context, id uint) (*entity.Item, error) {
	r.db.record("lock item")
	i, ok := r.find(id)
	if !ok {
		return nil, errx.ErrItemNotFound
	}
	return i, nil
}

func (r *fakeItemRepo) FindByIDsForUpdate(ctx context.Context, ids []uint) ([]entity.Item, error) {
	r.db.record("lock items")
	items, err := r.FindByIDs(ctx, ids) // sudah urut id
	if r.db.afterLockItems != nil {
		r.db.afterLockItems()
	}
	return items, err
}

// ===================================================================================================
// fakeBorrowingRepo: memenuhi contracts.BorrowingRepository
// ===================================================================================================

type fakeBorrowingRepo struct{ db *fakeDB }

// withDetails meniru Preload("User").Preload("Lembaga"): termasuk data yang sudah dihapus (Unscoped).
func (r *fakeBorrowingRepo) withDetails(b *entity.Borrowing) *entity.Borrowing {
	c := copyBorrowing(b)
	if u, ok := r.db.users[b.UserID]; ok {
		c.User = copyUser(u)
	}
	if l, ok := r.db.lembagas[b.LembagaID]; ok {
		c.Lembaga = copyLembaga(l)
	}
	return c
}

func (r *fakeBorrowingRepo) Create(_ context.Context, b *entity.Borrowing, activity *entity.UserLog) error {
	b.ID = r.db.newID()
	b.CreatedAt = time.Now()
	b.UpdatedAt = b.CreatedAt
	for i := range b.Items {
		b.Items[i].ID = r.db.newID()
		b.Items[i].BorrowingID = b.ID
	}
	stored := copyBorrowing(b)
	stored.User, stored.Lembaga = nil, nil
	r.db.borrowings[b.ID] = stored

	activity.RequestID = b.ID
	r.db.insertLog(activity)
	return nil
}

func (r *fakeBorrowingRepo) FindByID(_ context.Context, id uint) (*entity.Borrowing, error) {
	b, ok := r.db.borrowings[id]
	if !ok {
		return nil, errx.ErrBorrowingNotFound
	}
	return r.withDetails(b), nil
}

func (r *fakeBorrowingRepo) list(match func(*entity.Borrowing) bool, offset, limit int) ([]entity.Borrowing, int64, error) {
	var rows []entity.Borrowing
	for _, id := range slices.Backward(sortedIDs(r.db.borrowings)) { // terbaru dulu
		if b := r.db.borrowings[id]; match(b) {
			rows = append(rows, *r.withDetails(b))
		}
	}
	return pageOf(rows, offset, limit), int64(len(rows)), nil
}

func (r *fakeBorrowingRepo) FindByUser(_ context.Context, userID uint, offset, limit int) ([]entity.Borrowing, int64, error) {
	return r.list(func(b *entity.Borrowing) bool { return b.UserID == userID }, offset, limit)
}

func (r *fakeBorrowingRepo) FindAll(_ context.Context, status string, offset, limit int) ([]entity.Borrowing, int64, error) {
	return r.list(func(b *entity.Borrowing) bool { return status == "" || b.Status == status }, offset, limit)
}

func (r *fakeBorrowingRepo) FindByIDForUpdate(_ context.Context, id uint) (*entity.Borrowing, error) {
	r.db.record("lock borrowing")
	b, ok := r.db.borrowings[id]
	if !ok {
		return nil, errx.ErrBorrowingNotFound
	}
	c := copyBorrowing(b)
	c.Items = nil // query aslinya tanpa Preload
	return c, nil
}

func (r *fakeBorrowingRepo) UpdateStatus(_ context.Context, b *entity.Borrowing) error {
	r.db.borrowings[b.ID].Status = b.Status
	r.db.borrowings[b.ID].UpdatedAt = time.Now()
	return nil
}

// LockUnfulfillablePending meniru query aslinya: PENDING yang memuat salah satu itemIDs dengan
// jumlah > stok tersedia saat ini, urut id.
func (r *fakeBorrowingRepo) LockUnfulfillablePending(_ context.Context, itemIDs []uint) ([]uint, error) {
	r.db.record("lock pending")
	ids := []uint{}
	for _, id := range sortedIDs(r.db.borrowings) {
		b := r.db.borrowings[id]
		if b.Status != enums.BorrowingPending {
			continue
		}
		for _, bi := range b.Items {
			item, ok := r.db.items[bi.ItemID]
			if ok && slices.Contains(itemIDs, bi.ItemID) && bi.Quantity > item.AvailableQuantity {
				ids = append(ids, id)
				break
			}
		}
	}
	return ids, nil
}

func (r *fakeBorrowingRepo) UpdateStatuses(_ context.Context, ids []uint, status string) error {
	for _, id := range ids {
		r.db.borrowings[id].Status = status
	}
	return nil
}

// ===================================================================================================
// fakeUserLogRepo: memenuhi contracts.UserLogRepository
// ===================================================================================================

type fakeUserLogRepo struct{ db *fakeDB }

func (r *fakeUserLogRepo) withUser(l entity.UserLog) entity.UserLog {
	if u, ok := r.db.users[l.UserID]; ok {
		l.User = copyUser(u)
	}
	return l
}

func (r *fakeUserLogRepo) Create(_ context.Context, l *entity.UserLog) error {
	r.db.insertLog(l)
	return nil
}

func (r *fakeUserLogRepo) CreateMany(_ context.Context, logs []entity.UserLog) error {
	for i := range logs {
		r.db.insertLog(&logs[i])
	}
	return nil
}

func (r *fakeUserLogRepo) FindByRequest(_ context.Context, requestType string, requestID uint) ([]entity.UserLog, error) {
	res := []entity.UserLog{}
	for _, l := range r.db.logs { // logs disimpan urut id = urut waktu
		if l.RequestType == requestType && l.RequestID == requestID {
			res = append(res, r.withUser(l))
		}
	}
	return res, nil
}

func (r *fakeUserLogRepo) FindAll(_ context.Context, requestType string, offset, limit int) ([]entity.UserLog, int64, error) {
	var rows []entity.UserLog
	for _, l := range slices.Backward(r.db.logs) { // terbaru dulu
		if requestType == "" || l.RequestType == requestType {
			rows = append(rows, r.withUser(l))
		}
	}
	return pageOf(rows, offset, limit), int64(len(rows)), nil
}

// ===================================================================================================
// fakeRejecter: memenuhi contracts.PendingBorrowingRejecter (untuk tes ItemService)
// ===================================================================================================

type rejectCall struct {
	itemIDs []uint
	actorID uint
}

type fakeRejecter struct {
	calls []rejectCall
	err   error // jika diisi, RejectUnfulfillablePending gagal dengan error ini
}

func (f *fakeRejecter) RejectUnfulfillablePending(_ context.Context, itemIDs []uint, actorID uint) ([]uint, error) {
	f.calls = append(f.calls, rejectCall{itemIDs: slices.Clone(itemIDs), actorID: actorID})
	if f.err != nil {
		return nil, f.err
	}
	return []uint{}, nil
}
