package unit

import (
	"slices"
	"testing"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
	"SGEInventory_BE/internal/app/item"
)

const adminID uint = 100 // pelaku edit barang di tes ini

type itemFixture struct {
	db       *fakeDB
	rejecter *fakeRejecter
	svc      contracts.ItemService
	lembaga  *entity.Lembaga
}

// newItemFixture memakai fakeRejecter, jadi tes di sini hanya memeriksa KAPAN tolak otomatis dipanggil.
// Hasil tolak otomatis yang sebenarnya diuji di TestUpdateItem_WithRealBorrowingService.
func newItemFixture() *itemFixture {
	db := newFakeDB()
	f := &itemFixture{db: db, rejecter: &fakeRejecter{}}
	f.lembaga = db.addLembaga("BEM")
	f.svc = item.NewItemService(&fakeItemRepo{db}, &fakeLembagaRepo{db}, f.rejecter, &fakeTransactor{db: db})
	return f
}

func TestCreateItem(t *testing.T) {
	t.Parallel()
	f := newItemFixture()

	res, err := f.svc.CreateItem(t.Context(), dto.CreateItemRequest{Name: "Proyektor", LembagaID: f.lembaga.ID, TotalQuantity: 5})
	mustNoErr(t, err)
	if res.AvailableQuantity != 5 || res.Lembaga == nil || res.Lembaga.Name != "BEM" {
		t.Errorf("CreateItem = %+v; want tersedia 5 dan lembaga BEM", res)
	}

	_, err = f.svc.CreateItem(t.Context(), dto.CreateItemRequest{Name: "Kabel", LembagaID: 999, TotalQuantity: 1})
	assertDomainError(t, err, "LEMBAGA_NOT_FOUND", errx.KindNotFound)
}

func TestUpdateItem_AutoRejectOnlyWhenAvailableDrops(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		req           dto.UpdateItemRequest
		wantAvailable int
		wantRejectRun bool
	}{
		// awal: total 5, tersedia 3 (dipinjam 2)
		{name: "total dikurangi → tolak otomatis dijalankan", req: dto.UpdateItemRequest{TotalQuantity: ptr(4)}, wantAvailable: 2, wantRejectRun: true},
		{name: "total ditambah → tidak", req: dto.UpdateItemRequest{TotalQuantity: ptr(8)}, wantAvailable: 6},
		{name: "total sama → tidak", req: dto.UpdateItemRequest{TotalQuantity: ptr(5)}, wantAvailable: 3},
		{name: "hanya nama → tidak", req: dto.UpdateItemRequest{Name: "Proyektor Baru"}, wantAvailable: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newItemFixture()
			it := f.db.addItem("Proyektor", 5, 3, f.lembaga.ID)

			res, err := f.svc.UpdateItem(t.Context(), it.ID, adminID, tt.req)
			mustNoErr(t, err)
			if res.AvailableQuantity != tt.wantAvailable {
				t.Errorf("tersedia = %d; want %d", res.AvailableQuantity, tt.wantAvailable)
			}

			ran := len(f.rejecter.calls) > 0
			if ran != tt.wantRejectRun {
				t.Fatalf("tolak otomatis dijalankan = %v; want %v", ran, tt.wantRejectRun)
			}
			if ran {
				got := f.rejecter.calls[0]
				if !slices.Equal(got.itemIDs, []uint{it.ID}) || got.actorID != adminID {
					t.Errorf("RejectUnfulfillablePending(%v, %d); want ([%d], %d)", got.itemIDs, got.actorID, it.ID, adminID)
				}
			}
		})
	}
}

func TestUpdateItem_Errors_NothingSaved(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		req      dto.UpdateItemRequest
		rejErr   error
		wantCode string
		wantKind errx.Kind
	}{
		{name: "body kosong", req: dto.UpdateItemRequest{}, wantCode: "NOTHING_TO_UPDATE", wantKind: errx.KindInvalid},
		{name: "total di bawah jumlah dipinjam", req: dto.UpdateItemRequest{TotalQuantity: ptr(1)}, wantCode: "TOTAL_BELOW_BORROWED", wantKind: errx.KindConflict},
		{name: "lembaga baru tidak ada", req: dto.UpdateItemRequest{Name: "Baru", LembagaID: ptr(uint(999))}, wantCode: "LEMBAGA_NOT_FOUND", wantKind: errx.KindNotFound},
		{
			// tolak otomatis gagal → edit barang ikut dibatalkan (ROLLBACK)
			name: "tolak otomatis gagal", req: dto.UpdateItemRequest{Name: "Baru", TotalQuantity: ptr(4)},
			rejErr: errx.ErrBorrowingNotFound, wantCode: "BORROWING_NOT_FOUND", wantKind: errx.KindNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newItemFixture()
			f.rejecter.err = tt.rejErr
			it := f.db.addItem("Proyektor", 5, 3, f.lembaga.ID)

			_, err := f.svc.UpdateItem(t.Context(), it.ID, adminID, tt.req)
			assertDomainError(t, err, tt.wantCode, tt.wantKind)

			stored := f.db.items[it.ID]
			if stored.Name != "Proyektor" || stored.TotalQuantity != 5 || stored.AvailableQuantity != 3 {
				t.Errorf("barang berubah setelah error: %+v; want tetap Proyektor 5/3", stored)
			}
		})
	}

	t.Run("barang tidak ada", func(t *testing.T) {
		t.Parallel()
		f := newItemFixture()
		_, err := f.svc.UpdateItem(t.Context(), 999, adminID, dto.UpdateItemRequest{Name: "Kabel"})
		assertDomainError(t, err, "ITEM_NOT_FOUND", errx.KindNotFound)
	})
}

func TestUpdateItem_LocksItemRow(t *testing.T) {
	t.Parallel()
	f := newItemFixture()
	it := f.db.addItem("Proyektor", 5, 5, f.lembaga.ID)

	_, err := f.svc.UpdateItem(t.Context(), it.ID, adminID, dto.UpdateItemRequest{TotalQuantity: ptr(6)})
	mustNoErr(t, err)
	if !slices.Contains(f.db.calls, "lock item") {
		t.Errorf("langkah = %v; want barang dikunci (FindByIDForUpdate) sebelum diubah", f.db.calls)
	}
}

func TestDeleteItem(t *testing.T) {
	t.Parallel()

	t.Run("masih dipinjam ditolak", func(t *testing.T) {
		t.Parallel()
		f := newItemFixture()
		it := f.db.addItem("Proyektor", 5, 4, f.lembaga.ID)
		assertDomainError(t, f.svc.DeleteItem(t.Context(), it.ID), "ITEM_STILL_BORROWED", errx.KindConflict)
		if f.db.items[it.ID].DeletedAt.Valid {
			t.Error("barang terhapus; want tidak")
		}
	})

	t.Run("tidak dipinjam → terhapus", func(t *testing.T) {
		t.Parallel()
		f := newItemFixture()
		it := f.db.addItem("Proyektor", 5, 5, f.lembaga.ID)
		mustNoErr(t, f.svc.DeleteItem(t.Context(), it.ID))
		_, err := f.svc.GetItem(t.Context(), it.ID)
		assertDomainError(t, err, "ITEM_NOT_FOUND", errx.KindNotFound)
	})
}

func TestGetItems_OnlyAvailable(t *testing.T) {
	t.Parallel()
	f := newItemFixture()
	f.db.addItem("Proyektor", 5, 0, f.lembaga.ID)
	f.db.addItem("Kabel", 2, 2, f.lembaga.ID)

	res, page, err := f.svc.GetItems(t.Context(), dto.ItemListQuery{Page: 1, Limit: 10, OnlyAvailable: true})
	mustNoErr(t, err)
	if len(res) != 1 || res[0].Name != "Kabel" || res[0].Status != enums.ItemStatusAvailable || page.Total != 1 {
		t.Errorf("GetItems(available=true) = %+v; want hanya Kabel (TERSEDIA)", res)
	}
}

// Tes gabungan: ItemService memakai BorrowingService ASLI sebagai rejecter (repository tetap palsu).
func TestUpdateItem_WithRealBorrowingService(t *testing.T) {
	t.Parallel()
	bf := newBorrowingFixture(t)
	svc := item.NewItemService(&fakeItemRepo{bf.db}, &fakeLembagaRepo{bf.db}, bf.svc, bf.tx)

	it := bf.db.addItem("Proyektor", 5, 5, bf.lembaga.ID)
	needs4 := bf.db.addBorrowing(bf.user.ID, bf.lembaga.ID, enums.BorrowingPending, line{it.ID, 4})
	needs2 := bf.db.addBorrowing(bf.user.ID, bf.lembaga.ID, enums.BorrowingPending, line{it.ID, 2})

	_, err := svc.UpdateItem(t.Context(), it.ID, bf.admin.ID, dto.UpdateItemRequest{TotalQuantity: ptr(3)})
	mustNoErr(t, err)

	if got := bf.db.borrowings[needs4.ID].Status; got != enums.BorrowingRejected {
		t.Errorf("pengajuan butuh 4 (stok kini 3) status = %s; want REJECTED", got)
	}
	if got := bf.db.borrowings[needs2.ID].Status; got != enums.BorrowingPending {
		t.Errorf("pengajuan butuh 2 (stok kini 3) status = %s; want tetap PENDING", got)
	}
	logs := bf.db.logsFor(needs4.ID)
	last := logs[len(logs)-1]
	if last.RequestActivity != enums.ActivityAutoRejected || last.UserID != bf.admin.ID {
		t.Errorf("log terakhir = %s oleh user %d; want AUTO_REJECTED oleh admin %d", last.RequestActivity, last.UserID, bf.admin.ID)
	}
	if bf.tx.started != 1 {
		t.Errorf("transaksi dibuka %d kali; want 1 (edit barang + tolak otomatis dalam satu transaksi)", bf.tx.started)
	}
}
