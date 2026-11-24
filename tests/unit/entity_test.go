package unit

import (
	"strings"
	"testing"

	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
)

// ---------- Item ----------

func TestNewItem_AvailableEqualsTotal(t *testing.T) {
	t.Parallel()
	i := entity.NewItem("Proyektor", "Lemari B", 1, 5)
	if i.TotalQuantity != 5 || i.AvailableQuantity != 5 {
		t.Errorf("NewItem(total 5): total=%d available=%d; want 5 dan 5", i.TotalQuantity, i.AvailableQuantity)
	}
	if i.IsBorrowed() {
		t.Error("barang baru IsBorrowed() = true; want false")
	}
}

func TestItemChangeTotal(t *testing.T) {
	t.Parallel()
	// Awal: total 5, tersedia 2 → sedang dipinjam 3.
	tests := []struct {
		name          string
		newTotal      int
		wantTotal     int
		wantAvailable int
		wantErrCode   string
	}{
		{name: "total dinaikkan, sisa ikut naik", newTotal: 10, wantTotal: 10, wantAvailable: 7},
		{name: "total sama dengan jumlah dipinjam", newTotal: 3, wantTotal: 3, wantAvailable: 0},
		{name: "total di bawah jumlah dipinjam ditolak", newTotal: 2, wantTotal: 5, wantAvailable: 2, wantErrCode: "TOTAL_BELOW_BORROWED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			i := entity.NewItem("Proyektor", "", 1, 5)
			i.AvailableQuantity = 2

			err := i.ChangeTotal(tt.newTotal)
			if tt.wantErrCode != "" {
				de := assertDomainError(t, err, tt.wantErrCode, errx.KindConflict)
				if !strings.Contains(de.Message, "(3)") {
					t.Errorf("pesan %q; want menyebut jumlah dipinjam (3)", de.Message)
				}
			} else {
				mustNoErr(t, err)
			}
			if i.TotalQuantity != tt.wantTotal || i.AvailableQuantity != tt.wantAvailable {
				t.Errorf("ChangeTotal(%d): total=%d available=%d; want %d dan %d",
					tt.newTotal, i.TotalQuantity, i.AvailableQuantity, tt.wantTotal, tt.wantAvailable)
			}
		})
	}
}

func TestItemCanLend(t *testing.T) {
	t.Parallel()
	i := entity.NewItem("Kabel", "", 1, 3) // tersedia 3
	tests := []struct {
		qty  int
		want bool
	}{
		{qty: -1, want: false},
		{qty: 0, want: false},
		{qty: 1, want: true},
		{qty: 3, want: true},
		{qty: 4, want: false},
	}
	for _, tt := range tests {
		if got := i.CanLend(tt.qty); got != tt.want {
			t.Errorf("CanLend(%d) dengan tersedia 3 = %v; want %v", tt.qty, got, tt.want)
		}
	}
}

func TestItemLend(t *testing.T) {
	t.Parallel()

	t.Run("stok cukup, sisa berkurang", func(t *testing.T) {
		t.Parallel()
		i := entity.NewItem("Kabel", "", 1, 3)
		mustNoErr(t, i.Lend(2))
		if i.AvailableQuantity != 1 || i.TotalQuantity != 3 {
			t.Errorf("setelah Lend(2): available=%d total=%d; want 1 dan 3", i.AvailableQuantity, i.TotalQuantity)
		}
	})

	t.Run("stok kurang ditolak, sisa tidak berubah", func(t *testing.T) {
		t.Parallel()
		i := entity.NewItem("Kabel", "", 1, 3)
		de := assertDomainError(t, i.Lend(4), "INSUFFICIENT_STOCK", errx.KindConflict)
		if !strings.Contains(de.Message, "Kabel (diminta 4, tersedia 3)") {
			t.Errorf("pesan %q; want menyebut nama, jumlah diminta & tersedia", de.Message)
		}
		if i.AvailableQuantity != 3 {
			t.Errorf("available = %d setelah gagal; want tetap 3", i.AvailableQuantity)
		}
	})
}

func TestItemStatus(t *testing.T) {
	t.Parallel()
	i := entity.NewItem("Kabel", "", 1, 1)
	if got := i.Status(); got != enums.ItemStatusAvailable {
		t.Errorf("Status() dengan tersedia 1 = %q; want %q", got, enums.ItemStatusAvailable)
	}
	i.AvailableQuantity = 0
	if got := i.Status(); got != enums.ItemStatusEmpty {
		t.Errorf("Status() dengan tersedia 0 = %q; want %q", got, enums.ItemStatusEmpty)
	}
	if !i.IsBorrowed() || i.BorrowedQuantity() != 1 {
		t.Errorf("IsBorrowed()=%v BorrowedQuantity()=%d; want true dan 1", i.IsBorrowed(), i.BorrowedQuantity())
	}
}

func TestLendShortage(t *testing.T) {
	t.Parallel()
	got := entity.LendShortage("Proyektor", 5, 3)
	if want := "Proyektor (diminta 5, tersedia 3)"; got != want {
		t.Errorf("LendShortage() = %q; want %q", got, want)
	}
}

// ---------- Borrowing ----------

func TestNewBorrowing_StatusPending(t *testing.T) {
	t.Parallel()
	b := entity.NewBorrowing(1, 2, "Seminar", parseDay(t, "2026-12-01"), parseDay(t, "2026-12-03"), nil)
	if b.Status != enums.BorrowingPending {
		t.Errorf("NewBorrowing status = %q; want %q", b.Status, enums.BorrowingPending)
	}
}

func TestBorrowingApproveAndReject(t *testing.T) {
	t.Parallel()
	actions := []struct {
		name       string
		do         func(*entity.Borrowing) error
		wantStatus string
	}{
		{name: "Approve", do: (*entity.Borrowing).Approve, wantStatus: enums.BorrowingApproved},
		{name: "Reject", do: (*entity.Borrowing).Reject, wantStatus: enums.BorrowingRejected},
	}
	for _, act := range actions {
		t.Run(act.name+" dari PENDING", func(t *testing.T) {
			t.Parallel()
			b := &entity.Borrowing{Status: enums.BorrowingPending}
			mustNoErr(t, act.do(b))
			if b.Status != act.wantStatus {
				t.Errorf("status = %q; want %q", b.Status, act.wantStatus)
			}
		})

		// Setiap tahap hanya sekali: dari status selain PENDING selalu ditolak & status tidak berubah.
		for _, from := range []string{enums.BorrowingApproved, enums.BorrowingRejected, enums.BorrowingReturnPending, enums.BorrowingReturned} {
			t.Run(act.name+" dari "+from, func(t *testing.T) {
				t.Parallel()
				b := &entity.Borrowing{Status: from}
				de := assertDomainError(t, act.do(b), "BORROWING_ALREADY_PROCESSED", errx.KindConflict)
				if !strings.Contains(de.Message, from) {
					t.Errorf("pesan %q; want menyebut status %s", de.Message, from)
				}
				if b.Status != from {
					t.Errorf("status berubah menjadi %q; want tetap %q", b.Status, from)
				}
			})
		}
	}
}

// ---------- User ----------

func TestNewUser(t *testing.T) {
	t.Parallel()
	u := entity.NewUser("Budi", "22105110", "Budi@Mail.COM", "hash", "0812")
	if u.Email != "budi@mail.com" {
		t.Errorf("email = %q; want huruf kecil semua", u.Email)
	}
	if u.Role != enums.RoleUser || u.IsAdmin() {
		t.Errorf("role = %q, IsAdmin() = %v; user baru harus USER", u.Role, u.IsAdmin())
	}
}

func TestUserApplyUpdate(t *testing.T) {
	t.Parallel()
	u := entity.NewUser("Budi", "22105110", "budi@mail.com", "hash", "0812")

	u.ApplyUpdate("", "", "") // field kosong = tidak diubah
	if u.FullName != "Budi" || u.Email != "budi@mail.com" || u.Contact != "0812" {
		t.Errorf("ApplyUpdate kosong mengubah data: %+v", u)
	}

	u.ApplyUpdate("Budi S", "BARU@mail.com", "0899")
	if u.FullName != "Budi S" || u.Email != "baru@mail.com" || u.Contact != "0899" {
		t.Errorf("ApplyUpdate: nama=%q email=%q kontak=%q", u.FullName, u.Email, u.Contact)
	}
}
