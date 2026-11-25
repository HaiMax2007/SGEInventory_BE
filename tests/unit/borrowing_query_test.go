package unit

import (
	"slices"
	"testing"

	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
)

func TestGetBorrowing_Access(t *testing.T) {
	t.Parallel()
	f := newBorrowingFixture(t)
	other := f.db.addUser("Budi", enums.RoleUser)
	it := f.db.addItem("Proyektor", 5, 5, f.lembaga.ID)
	b := f.db.addBorrowing(f.user.ID, f.lembaga.ID, enums.BorrowingPending, line{it.ID, 1})

	tests := []struct {
		name     string
		userID   uint
		isAdmin  bool
		id       uint
		wantCode string // kosong = berhasil
	}{
		{name: "pemilik", userID: f.user.ID, id: b.ID},
		{name: "admin", userID: f.admin.ID, isAdmin: true, id: b.ID},
		// milik orang lain dibalas 404 (bukan 403) supaya id pengajuan orang lain tidak bisa ditebak
		{name: "user lain", userID: other.ID, id: b.ID, wantCode: "BORROWING_NOT_FOUND"},
		{name: "tidak ada", userID: f.admin.ID, isAdmin: true, id: 999, wantCode: "BORROWING_NOT_FOUND"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := f.svc.GetBorrowing(t.Context(), tt.id, tt.userID, tt.isAdmin)
			if tt.wantCode != "" {
				assertDomainError(t, err, tt.wantCode, errx.KindNotFound)
				return
			}
			mustNoErr(t, err)
			if res.ID != b.ID || len(res.Timeline) != 1 || res.Timeline[0].Actor.ID != f.user.ID {
				t.Errorf("GetBorrowing = id %d timeline %+v; want id %d dengan 1 log oleh peminjam", res.ID, res.Timeline, b.ID)
			}
		})
	}
}

// Timeline diambil per PENGAJUAN, jadi aksi admin ikut terlihat oleh peminjam walau user_id log = admin.
func TestGetBorrowing_TimelineIncludesAdminAction(t *testing.T) {
	t.Parallel()
	f := newBorrowingFixture(t)
	it := f.db.addItem("Proyektor", 5, 5, f.lembaga.ID)
	b := f.db.addBorrowing(f.user.ID, f.lembaga.ID, enums.BorrowingPending, line{it.ID, 1})
	_, err := f.svc.ApproveBorrowing(t.Context(), b.ID, f.admin.ID)
	mustNoErr(t, err)

	res, err := f.svc.GetBorrowing(t.Context(), b.ID, f.user.ID, false)
	mustNoErr(t, err)
	if len(res.Timeline) != 2 {
		t.Fatalf("timeline = %+v; want 2 baris", res.Timeline)
	}
	first, second := res.Timeline[0], res.Timeline[1]
	if first.Activity != enums.ActivitySubmitted || first.Actor.ID != f.user.ID ||
		second.Activity != enums.ActivityApproved || second.Actor.ID != f.admin.ID {
		t.Errorf("timeline = [%s oleh %d, %s oleh %d]; want [SUBMITTED oleh peminjam, APPROVED oleh admin]",
			first.Activity, first.Actor.ID, second.Activity, second.Actor.ID)
	}
}

func TestGetMyBorrowings_OnlyOwnNewestFirst(t *testing.T) {
	t.Parallel()
	f := newBorrowingFixture(t)
	other := f.db.addUser("Budi", enums.RoleUser)
	it := f.db.addItem("Proyektor", 5, 5, f.lembaga.ID)
	older := f.db.addBorrowing(f.user.ID, f.lembaga.ID, enums.BorrowingPending, line{it.ID, 1})
	f.db.addBorrowing(other.ID, f.lembaga.ID, enums.BorrowingPending, line{it.ID, 1})
	newer := f.db.addBorrowing(f.user.ID, f.lembaga.ID, enums.BorrowingRejected, line{it.ID, 1})

	res, page, err := f.svc.GetMyBorrowings(t.Context(), f.user.ID, dto.BorrowingListQuery{Page: 1, Limit: 10})
	mustNoErr(t, err)
	if got := borrowingIDs(res); !slices.Equal(got, []uint{newer.ID, older.ID}) || page.Total != 2 {
		t.Errorf("GetMyBorrowings = %v (total %d); want [%d %d] (total 2)", got, page.Total, newer.ID, older.ID)
	}
	if res[0].Timeline != nil {
		t.Error("daftar berisi timeline; want timeline hanya di detail")
	}
}

func TestGetBorrowings_Admin(t *testing.T) {
	t.Parallel()
	f := newBorrowingFixture(t)
	it := f.db.addItem("Proyektor", 5, 5, f.lembaga.ID)
	p1 := f.db.addBorrowing(f.user.ID, f.lembaga.ID, enums.BorrowingPending, line{it.ID, 1})
	f.db.addBorrowing(f.user.ID, f.lembaga.ID, enums.BorrowingApproved, line{it.ID, 1})
	p3 := f.db.addBorrowing(f.user.ID, f.lembaga.ID, enums.BorrowingPending, line{it.ID, 1})

	t.Run("filter PENDING", func(t *testing.T) {
		res, _, err := f.svc.GetBorrowings(t.Context(), dto.BorrowingListQuery{Page: 1, Limit: 10, Status: enums.BorrowingPending})
		mustNoErr(t, err)
		if got := borrowingIDs(res); !slices.Equal(got, []uint{p3.ID, p1.ID}) {
			t.Errorf("status=PENDING → %v; want [%d %d]", got, p3.ID, p1.ID)
		}
		if res[0].User == nil || res[0].User.FullName != "Raka" {
			t.Errorf("user = %+v; want peminjam ikut ditampilkan", res[0].User)
		}
	})

	t.Run("user berisi NIM & kontak peminjam", func(t *testing.T) {
		f.db.users[f.user.ID].Contact = "081234567890"
		res, _, err := f.svc.GetBorrowings(t.Context(), dto.BorrowingListQuery{Page: 1, Limit: 10})
		mustNoErr(t, err)
		want := dto.UserSummary{ID: f.user.ID, FullName: "Raka", NIM: f.user.NIM, Contact: "081234567890"}
		if res[0].User == nil || *res[0].User != want {
			t.Errorf("user = %+v; want %+v", res[0].User, want)
		}
	})

	t.Run("tanpa filter + halaman", func(t *testing.T) {
		res, page, err := f.svc.GetBorrowings(t.Context(), dto.BorrowingListQuery{Page: 2, Limit: 2})
		mustNoErr(t, err)
		if len(res) != 1 || page.Total != 3 || page.TotalPages != 2 {
			t.Errorf("page 2 limit 2 → %d data, total %d, %d halaman; want 1, 3, 2", len(res), page.Total, page.TotalPages)
		}
	})

	t.Run("status tidak dikenal", func(t *testing.T) {
		_, _, err := f.svc.GetBorrowings(t.Context(), dto.BorrowingListQuery{Status: "pending"})
		assertDomainError(t, err, "INVALID_BORROWING_STATUS", errx.KindInvalid)
	})
}

func borrowingIDs(rows []dto.BorrowingResponse) []uint {
	ids := make([]uint, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	return ids
}
