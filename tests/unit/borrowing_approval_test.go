package unit

import (
	"slices"
	"strings"
	"testing"

	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
)

// ---------- ACC ----------

func TestApproveBorrowing_Success(t *testing.T) {
	t.Parallel()
	f := newBorrowingFixture(t)
	proyektor := f.db.addItem("Proyektor", 5, 5, f.lembaga.ID)
	kabel := f.db.addItem("Kabel", 3, 3, f.lembaga.ID)
	b := f.db.addBorrowing(f.user.ID, f.lembaga.ID, enums.BorrowingPending, line{proyektor.ID, 3}, line{kabel.ID, 1})

	res, err := f.svc.ApproveBorrowing(t.Context(), b.ID, f.admin.ID)
	mustNoErr(t, err)

	// stok berkurang, total tetap
	for _, want := range []struct {
		id               uint
		total, available int
	}{{proyektor.ID, 5, 2}, {kabel.ID, 3, 2}} {
		got := f.db.items[want.id]
		if got.TotalQuantity != want.total || got.AvailableQuantity != want.available {
			t.Errorf("%s: total %d tersedia %d; want %d dan %d", got.Name, got.TotalQuantity, got.AvailableQuantity, want.total, want.available)
		}
	}
	if got := f.db.borrowings[b.ID].Status; got != enums.BorrowingApproved {
		t.Errorf("status di database = %s; want APPROVED", got)
	}
	if res.Borrowing.Status != enums.BorrowingApproved {
		t.Errorf("status di response = %s; want APPROVED", res.Borrowing.Status)
	}
	// [] bukan null, supaya FE selalu menerima array
	if res.AutoRejectedIDs == nil || len(res.AutoRejectedIDs) != 0 {
		t.Errorf("autoRejectedIds = %#v; want [] (slice kosong, bukan nil)", res.AutoRejectedIDs)
	}
	logs := f.db.logsFor(b.ID)
	if got := activities(logs); !slices.Equal(got, []string{enums.ActivitySubmitted, enums.ActivityApproved}) {
		t.Fatalf("log = %v; want [SUBMITTED APPROVED]", got)
	}
	if logs[1].UserID != f.admin.ID {
		t.Errorf("pelaku APPROVED = user %d; want admin %d", logs[1].UserID, f.admin.ID)
	}
	// Response ACC tanpa timeline (riwayat lengkap ada di GET /api/borrowings/:id).
	if res.Borrowing.Timeline != nil {
		t.Errorf("timeline response = %v; want nil (tanpa timeline)", res.Borrowing.Timeline)
	}
}

func TestApproveBorrowing_AutoRejectsOnlyUnfulfillablePending(t *testing.T) {
	t.Parallel()
	f := newBorrowingFixture(t)
	a := f.db.addItem("Proyektor", 10, 5, f.lembaga.ID) // tersedia 5
	b := f.db.addItem("Kabel", 3, 3, f.lembaga.ID)      // tersedia 3

	approve := f.db.addBorrowing(f.user.ID, f.lembaga.ID, enums.BorrowingPending, line{a.ID, 3}) // A: 5 → 2
	needs3 := f.db.addBorrowing(f.user.ID, f.lembaga.ID, enums.BorrowingPending, line{a.ID, 3})  // 3 > 2 → ditolak
	needs2 := f.db.addBorrowing(f.user.ID, f.lembaga.ID, enums.BorrowingPending, line{a.ID, 2})  // 2 ≤ 2 → tetap
	// Kabel stoknya tidak berubah oleh ACC ini, jadi tidak diperiksa walau 5 > 3.
	otherItem := f.db.addBorrowing(f.user.ID, f.lembaga.ID, enums.BorrowingPending, line{b.ID, 5})
	mixed := f.db.addBorrowing(f.user.ID, f.lembaga.ID, enums.BorrowingPending, line{b.ID, 1}, line{a.ID, 3}) // A kurang → ditolak
	done := f.db.addBorrowing(f.user.ID, f.lembaga.ID, enums.BorrowingApproved, line{a.ID, 4})                // bukan PENDING → tidak disentuh

	res, err := f.svc.ApproveBorrowing(t.Context(), approve.ID, f.admin.ID)
	mustNoErr(t, err)

	if want := []uint{needs3.ID, mixed.ID}; !slices.Equal(res.AutoRejectedIDs, want) {
		t.Errorf("autoRejectedIds = %v; want %v", res.AutoRejectedIDs, want)
	}
	wantStatus := map[uint]string{
		needs3.ID:    enums.BorrowingRejected,
		needs2.ID:    enums.BorrowingPending,
		otherItem.ID: enums.BorrowingPending,
		mixed.ID:     enums.BorrowingRejected,
		done.ID:      enums.BorrowingApproved,
	}
	for id, want := range wantStatus {
		if got := f.db.borrowings[id].Status; got != want {
			t.Errorf("pengajuan %d status = %s; want %s", id, got, want)
		}
	}
	for _, id := range []uint{needs3.ID, mixed.ID} {
		logs := f.db.logsFor(id)
		last := logs[len(logs)-1]
		if last.RequestActivity != enums.ActivityAutoRejected || last.UserID != f.admin.ID {
			t.Errorf("log terakhir pengajuan %d = %s oleh %d; want AUTO_REJECTED oleh admin %d",
				id, last.RequestActivity, last.UserID, f.admin.ID)
		}
	}
	if got := activities(f.db.logsFor(needs2.ID)); !slices.Equal(got, []string{enums.ActivitySubmitted}) {
		t.Errorf("log pengajuan yang tetap PENDING = %v; want hanya [SUBMITTED]", got)
	}
}

// Semua error ACC: tidak ada yang berubah (stok, status, log) karena transaksi dibatalkan.
func TestApproveBorrowing_Errors_NothingChanges(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		setup       func(f *borrowingFixture) uint // mengembalikan id pengajuan yang di-ACC
		wantCode    string
		wantKind    errx.Kind
		wantMessage string
	}{
		{
			name:     "pengajuan tidak ada",
			setup:    func(*borrowingFixture) uint { return 999 },
			wantCode: "BORROWING_NOT_FOUND", wantKind: errx.KindNotFound,
		},
		{
			name: "sudah APPROVED",
			setup: func(f *borrowingFixture) uint {
				it := f.db.addItem("Proyektor", 5, 5, f.lembaga.ID)
				return f.db.addBorrowing(f.user.ID, f.lembaga.ID, enums.BorrowingApproved, line{it.ID, 1}).ID
			},
			wantCode: "BORROWING_ALREADY_PROCESSED", wantKind: errx.KindConflict, wantMessage: "APPROVED",
		},
		{
			name: "sudah REJECTED",
			setup: func(f *borrowingFixture) uint {
				it := f.db.addItem("Proyektor", 5, 5, f.lembaga.ID)
				return f.db.addBorrowing(f.user.ID, f.lembaga.ID, enums.BorrowingRejected, line{it.ID, 1}).ID
			},
			wantCode: "BORROWING_ALREADY_PROCESSED", wantKind: errx.KindConflict, wantMessage: "REJECTED",
		},
		{
			// stok berubah sejak submit (mis. dipakai pengajuan lain): dicek ulang saat ACC
			name: "stok kurang di 2 barang",
			setup: func(f *borrowingFixture) uint {
				a := f.db.addItem("Proyektor", 5, 2, f.lembaga.ID)
				b := f.db.addItem("Kabel", 3, 0, f.lembaga.ID)
				ok := f.db.addItem("Terminal", 3, 3, f.lembaga.ID)
				return f.db.addBorrowing(f.user.ID, f.lembaga.ID, enums.BorrowingPending,
					line{ok.ID, 1}, line{a.ID, 3}, line{b.ID, 1}).ID
			},
			wantCode: "INSUFFICIENT_STOCK", wantKind: errx.KindConflict,
			wantMessage: "Proyektor (diminta 3, tersedia 2), Kabel (diminta 1, tersedia 0)",
		},
		{
			name: "barang sudah dihapus",
			setup: func(f *borrowingFixture) uint {
				ok := f.db.addItem("Terminal", 3, 3, f.lembaga.ID)
				gone := f.db.addItem("Proyektor", 5, 5, f.lembaga.ID)
				id := f.db.addBorrowing(f.user.ID, f.lembaga.ID, enums.BorrowingPending, line{ok.ID, 1}, line{gone.ID, 1}).ID
				f.db.softDeleteItem(gone.ID)
				return id
			},
			wantCode: "BORROW_ITEMS_UNAVAILABLE", wantKind: errx.KindConflict, wantMessage: "Proyektor",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newBorrowingFixture(t)
			id := tt.setup(f)
			before := f.db.clone()

			_, err := f.svc.ApproveBorrowing(t.Context(), id, f.admin.ID)
			de := assertDomainError(t, err, tt.wantCode, tt.wantKind)
			if !strings.Contains(de.Message, tt.wantMessage) {
				t.Errorf("pesan %q; want berisi %q", de.Message, tt.wantMessage)
			}
			assertUnchanged(t, before, f.db)
		})
	}
}

// Admin lain memproses pengajuan yang sama di antara "cek cepat" dan "kunci": status dicek ulang
// setelah terkunci, jadi ACC kedua ditolak dan stok tidak berkurang dua kali.
func TestApproveBorrowing_RechecksStatusAfterLocking(t *testing.T) {
	t.Parallel()
	f := newBorrowingFixture(t)
	it := f.db.addItem("Proyektor", 5, 5, f.lembaga.ID)
	b := f.db.addBorrowing(f.user.ID, f.lembaga.ID, enums.BorrowingPending, line{it.ID, 2})

	f.db.afterLockItems = func() { f.db.borrowings[b.ID].Status = enums.BorrowingApproved }

	_, err := f.svc.ApproveBorrowing(t.Context(), b.ID, f.admin.ID)
	assertDomainError(t, err, "BORROWING_ALREADY_PROCESSED", errx.KindConflict)
	if got := f.db.items[it.ID].AvailableQuantity; got != 5 {
		t.Errorf("stok = %d; want tetap 5 (tidak dikurangi)", got)
	}
}

// Urutan kunci harus: barang → pengajuan → pengajuan PENDING lain. Urutan yang sama di semua alur
// mencegah dua transaksi saling menunggu (deadlock).
func TestApproveBorrowing_LockOrderAndSingleTransaction(t *testing.T) {
	t.Parallel()
	f := newBorrowingFixture(t)
	it := f.db.addItem("Proyektor", 5, 5, f.lembaga.ID)
	b := f.db.addBorrowing(f.user.ID, f.lembaga.ID, enums.BorrowingPending, line{it.ID, 1})

	_, err := f.svc.ApproveBorrowing(t.Context(), b.ID, f.admin.ID)
	mustNoErr(t, err)

	want := []string{"lock items", "lock borrowing", "lock pending"}
	if !slices.Equal(f.db.calls, want) {
		t.Errorf("urutan kunci = %v; want %v", f.db.calls, want)
	}
	if f.tx.started != 1 {
		t.Errorf("transaksi dibuka %d kali; want 1 (ACC + tolak otomatis dalam satu transaksi)", f.tx.started)
	}
}

// ---------- Tolak ----------

func TestRejectBorrowing(t *testing.T) {
	t.Parallel()

	t.Run("berhasil, stok tidak disentuh", func(t *testing.T) {
		t.Parallel()
		f := newBorrowingFixture(t)
		it := f.db.addItem("Proyektor", 5, 5, f.lembaga.ID)
		b := f.db.addBorrowing(f.user.ID, f.lembaga.ID, enums.BorrowingPending, line{it.ID, 3})

		res, err := f.svc.RejectBorrowing(t.Context(), b.ID, f.admin.ID)
		mustNoErr(t, err)
		if res.Timeline != nil {
			t.Errorf("timeline = %v; want nil (response tolak tanpa timeline)", res.Timeline)
		}
		if res.Status != enums.BorrowingRejected || f.db.borrowings[b.ID].Status != enums.BorrowingRejected {
			t.Errorf("status = %s; want REJECTED", res.Status)
		}
		if got := f.db.items[it.ID].AvailableQuantity; got != 5 {
			t.Errorf("stok = %d; want tetap 5", got)
		}
		logs := f.db.logsFor(b.ID)
		if got := activities(logs); !slices.Equal(got, []string{enums.ActivitySubmitted, enums.ActivityRejected}) || logs[1].UserID != f.admin.ID {
			t.Errorf("log = %v (pelaku terakhir %d); want [SUBMITTED REJECTED] oleh admin", got, logs[len(logs)-1].UserID)
		}
	})

	for _, from := range []string{enums.BorrowingApproved, enums.BorrowingRejected} {
		t.Run("dari "+from+" ditolak", func(t *testing.T) {
			t.Parallel()
			f := newBorrowingFixture(t)
			it := f.db.addItem("Proyektor", 5, 5, f.lembaga.ID)
			b := f.db.addBorrowing(f.user.ID, f.lembaga.ID, from, line{it.ID, 1})
			before := f.db.clone()

			_, err := f.svc.RejectBorrowing(t.Context(), b.ID, f.admin.ID)
			assertDomainError(t, err, "BORROWING_ALREADY_PROCESSED", errx.KindConflict)
			assertUnchanged(t, before, f.db)
		})
	}

	t.Run("tidak ada", func(t *testing.T) {
		t.Parallel()
		f := newBorrowingFixture(t)
		_, err := f.svc.RejectBorrowing(t.Context(), 999, f.admin.ID)
		assertDomainError(t, err, "BORROWING_NOT_FOUND", errx.KindNotFound)
	})
}

// ---------- Tolak otomatis ----------

func TestRejectUnfulfillablePending_EmptyResultIsNotNil(t *testing.T) {
	t.Parallel()
	f := newBorrowingFixture(t)

	got, err := f.svc.RejectUnfulfillablePending(t.Context(), nil, f.admin.ID)
	mustNoErr(t, err)
	if got == nil || len(got) != 0 || f.tx.started != 0 {
		t.Errorf("tanpa barang: hasil %#v, transaksi %d; want [] dan tanpa transaksi", got, f.tx.started)
	}

	it := f.db.addItem("Proyektor", 5, 5, f.lembaga.ID)
	f.db.addBorrowing(f.user.ID, f.lembaga.ID, enums.BorrowingPending, line{it.ID, 1})
	got, err = f.svc.RejectUnfulfillablePending(t.Context(), []uint{it.ID}, f.admin.ID)
	mustNoErr(t, err)
	if got == nil || len(got) != 0 {
		t.Errorf("semua tercukupi: hasil %#v; want []", got)
	}
}

// ---------- helper ----------

// assertUnchanged memastikan stok, status, dan jumlah log sama seperti sebelum aksi dijalankan.
func assertUnchanged(t *testing.T, before, after *fakeDB) {
	t.Helper()
	for id, b := range before.items {
		if a := after.items[id]; a.AvailableQuantity != b.AvailableQuantity || a.TotalQuantity != b.TotalQuantity {
			t.Errorf("stok %s berubah: %d/%d → %d/%d", b.Name, b.AvailableQuantity, b.TotalQuantity, a.AvailableQuantity, a.TotalQuantity)
		}
	}
	for id, b := range before.borrowings {
		if a := after.borrowings[id]; a.Status != b.Status {
			t.Errorf("status pengajuan %d berubah: %s → %s", id, b.Status, a.Status)
		}
	}
	if len(after.logs) != len(before.logs) {
		t.Errorf("jumlah log berubah: %d → %d", len(before.logs), len(after.logs))
	}
}
