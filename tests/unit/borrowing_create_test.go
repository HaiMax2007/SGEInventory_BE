package unit

import (
	"strings"
	"testing"

	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
)

// validRequest membuat form peminjaman yang lolos semua aturan: ambil besok, kembali lusa.
func validRequest(lembagaID uint, items ...dto.BorrowingItemRequest) dto.CreateBorrowingRequest {
	return dto.CreateBorrowingRequest{
		LembagaID:  lembagaID,
		Purpose:    "Seminar nasional",
		PickUpDate: dateWIB(1),
		ReturnDate: dateWIB(3),
		Items:      items,
	}
}

func TestCreateBorrowing_Success(t *testing.T) {
	t.Parallel()
	f := newBorrowingFixture(t)
	proyektor := f.db.addItem("Proyektor", 5, 5, f.lembaga.ID)
	kabel := f.db.addItem("Kabel HDMI", 3, 3, f.lembaga.ID)

	req := validRequest(f.lembaga.ID,
		dto.BorrowingItemRequest{ItemID: proyektor.ID, Quantity: 5}, // tepat sebanyak stok: boleh
		dto.BorrowingItemRequest{ItemID: kabel.ID, Quantity: 1},
	)
	req.PickUpDate = dateWIB(0) // hari ini boleh
	req.ReturnDate = dateWIB(7) // maksimal +7 hari

	res, err := f.svc.CreateBorrowing(t.Context(), f.user.ID, req)
	mustNoErr(t, err)

	if res.Status != enums.BorrowingPending || res.Purpose != "Seminar nasional" {
		t.Errorf("status %q purpose %q; want PENDING dan tujuan dari form", res.Status, res.Purpose)
	}
	if res.User == nil || res.User.ID != f.user.ID || res.Lembaga == nil || res.Lembaga.ID != f.lembaga.ID {
		t.Errorf("user %+v lembaga %+v; want peminjam dari token & lembaga dari form", res.User, res.Lembaga)
	}
	if len(res.Items) != 2 || res.Items[0].ItemName != "Proyektor" || res.Items[1].Quantity != 1 {
		t.Errorf("items = %+v; want 2 barang dengan nama diambil dari database", res.Items)
	}
	// Submit hanya mengecek stok, tidak menguranginya.
	if got := f.db.items[proyektor.ID].AvailableQuantity; got != 5 {
		t.Errorf("stok Proyektor = %d setelah submit; want tetap 5", got)
	}
	logs := f.db.logsFor(res.ID)
	if len(logs) != 1 || logs[0].RequestActivity != enums.ActivitySubmitted || logs[0].UserID != f.user.ID {
		t.Errorf("log = %+v; want 1 log SUBMITTED oleh peminjam", logs)
	}
}

// Aturan antar-field & data di database. Aturan per field (tag `validate`) diuji di validator_test.go,
// karena dicek di handler sebelum service dipanggil. Untuk semua kasus gagal: tidak ada data yang tersimpan.
func TestCreateBorrowing_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		change      func(r *dto.CreateBorrowingRequest, itemA, itemB uint)
		wantCode    string
		wantKind    errx.Kind
		wantMessage string // potongan pesan yang harus ada (opsional)
	}{
		// --- barang ---
		{name: "barang dobel", change: func(r *dto.CreateBorrowingRequest, a, b uint) {
			r.Items = []dto.BorrowingItemRequest{{ItemID: a, Quantity: 1}, {ItemID: b, Quantity: 1}, {ItemID: a, Quantity: 2}}
		}, wantCode: "DUPLICATE_BORROW_ITEM", wantKind: errx.KindInvalid, wantMessage: "Barang ke-3 sama dengan barang ke-1"},

		// --- lembaga ---
		{name: "lembaga tidak ada", change: func(r *dto.CreateBorrowingRequest, _, _ uint) { r.LembagaID = 999 },
			wantCode: "LEMBAGA_NOT_FOUND", wantKind: errx.KindNotFound},

		// --- tanggal ---
		{name: "ambil kemarin", change: func(r *dto.CreateBorrowingRequest, _, _ uint) { r.PickUpDate = dateWIB(-1) },
			wantCode: "PICKUP_DATE_PAST", wantKind: errx.KindInvalid},
		{name: "kembali di hari yang sama", change: func(r *dto.CreateBorrowingRequest, _, _ uint) { r.ReturnDate = r.PickUpDate },
			wantCode: "RETURN_NOT_AFTER_PICKUP", wantKind: errx.KindInvalid},
		{name: "kembali sebelum ambil", change: func(r *dto.CreateBorrowingRequest, _, _ uint) { r.PickUpDate, r.ReturnDate = dateWIB(3), dateWIB(2) },
			wantCode: "RETURN_NOT_AFTER_PICKUP", wantKind: errx.KindInvalid},
		{name: "pinjam 8 hari", change: func(r *dto.CreateBorrowingRequest, _, _ uint) { r.PickUpDate, r.ReturnDate = dateWIB(1), dateWIB(9) },
			wantCode: "BORROW_TOO_LONG", wantKind: errx.KindInvalid, wantMessage: "7 hari"},

		// --- data di database ---
		{name: "barang tidak ada", change: func(r *dto.CreateBorrowingRequest, a, _ uint) {
			r.Items = []dto.BorrowingItemRequest{{ItemID: a, Quantity: 1}, {ItemID: 998, Quantity: 1}, {ItemID: 999, Quantity: 1}}
		}, wantCode: "BORROW_ITEMS_NOT_FOUND", wantKind: errx.KindNotFound, wantMessage: "998, 999"},
		{name: "stok kurang di 2 barang (disebut semua)", change: func(r *dto.CreateBorrowingRequest, a, b uint) {
			r.Items = []dto.BorrowingItemRequest{{ItemID: a, Quantity: 6}, {ItemID: b, Quantity: 4}}
		}, wantCode: "INSUFFICIENT_STOCK", wantKind: errx.KindConflict,
			wantMessage: "Proyektor (diminta 6, tersedia 5), Kabel HDMI (diminta 4, tersedia 3)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newBorrowingFixture(t)
			a := f.db.addItem("Proyektor", 5, 5, f.lembaga.ID)
			b := f.db.addItem("Kabel HDMI", 3, 3, f.lembaga.ID)
			req := validRequest(f.lembaga.ID, dto.BorrowingItemRequest{ItemID: a.ID, Quantity: 1})
			tt.change(&req, a.ID, b.ID)

			_, err := f.svc.CreateBorrowing(t.Context(), f.user.ID, req)
			de := assertDomainError(t, err, tt.wantCode, tt.wantKind)
			if !strings.Contains(de.Message, tt.wantMessage) {
				t.Errorf("pesan %q; want berisi %q", de.Message, tt.wantMessage)
			}
			if len(f.db.borrowings) != 0 || len(f.db.logs) != 0 {
				t.Errorf("tersimpan %d pengajuan & %d log; want tidak ada", len(f.db.borrowings), len(f.db.logs))
			}
		})
	}
}

func TestCreateBorrowing_DeletedItemCountsAsNotFound(t *testing.T) {
	t.Parallel()
	f := newBorrowingFixture(t)
	it := f.db.addItem("Proyektor", 5, 5, f.lembaga.ID)
	f.db.softDeleteItem(it.ID)

	_, err := f.svc.CreateBorrowing(t.Context(), f.user.ID, validRequest(f.lembaga.ID, dto.BorrowingItemRequest{ItemID: it.ID, Quantity: 1}))
	assertDomainError(t, err, "BORROW_ITEMS_NOT_FOUND", errx.KindNotFound)
}
