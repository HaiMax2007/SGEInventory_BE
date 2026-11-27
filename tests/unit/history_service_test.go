package unit

import (
	"slices"
	"testing"
	"time"

	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/internal/app/history"
)

// historyRow meringkas satu baris riwayat untuk dibandingkan: jenis + id pengajuan.
type historyRow struct {
	typ string
	id  uint
}

// Riwayat Raka (createdAt selang-seling antar tabel): B1 (1 jam), C1 (2 jam), B2 (3 jam), C2 (4 jam), B3 (5 jam).
// Budi punya 1 peminjaman & 1 penitipan yang lebih baru, tapi tidak boleh ikut.
// Di tiap tabel id yang lebih besar juga lebih baru, sama seperti di database asli.
func TestGetMyHistory(t *testing.T) {
	t.Parallel()
	f := newConsignmentFixture()
	budi := f.db.addUser("Budi", enums.RoleUser)
	it := f.db.addItem("Proyektor", 5, 5, f.lembaga.ID)
	base := time.Date(2026, 11, 26, 8, 0, 0, 0, time.UTC)
	at := func(hours int) time.Time { return base.Add(time.Duration(hours) * time.Hour) }

	borrow := func(userID uint, hours int) uint {
		b := f.db.addBorrowing(userID, f.lembaga.ID, enums.BorrowingPending, line{it.ID, 1})
		f.db.borrowings[b.ID].CreatedAt = at(hours)
		return b.ID
	}
	consign := func(userID uint, hours int) uint {
		res, err := f.svc.CreateConsignment(t.Context(), userID, validConsignment(t, f.lembaga.ID))
		mustNoErr(t, err)
		f.db.consignments[res.ID].CreatedAt = at(hours)
		return res.ID
	}

	b1 := borrow(f.user.ID, 1)
	c1 := consign(f.user.ID, 2)
	b2 := borrow(f.user.ID, 3)
	c2 := consign(f.user.ID, 4)
	b3 := borrow(f.user.ID, 5)
	borrow(budi.ID, 6)
	consign(budi.ID, 7)

	returnPhoto := "returns/abc.png"
	f.db.borrowings[b3].ReturnPhoto = &returnPhoto

	svc := history.NewHistoryService(&fakeBorrowingRepo{db: f.db}, f.repo, f.storage)
	B, C := enums.RequestTypeBorrowing, enums.RequestTypeConsignment

	tests := []struct {
		name           string
		page, limit    int
		want           []historyRow
		wantTotalPages int
	}{
		{name: "semua, terbaru dulu", page: 1, limit: 10,
			want: []historyRow{{B, b3}, {C, c2}, {B, b2}, {C, c1}, {B, b1}}, wantTotalPages: 1},
		{name: "halaman 2 limit 2", page: 2, limit: 2, want: []historyRow{{B, b2}, {C, c1}}, wantTotalPages: 3},
		{name: "halaman terakhir tidak penuh", page: 3, limit: 2, want: []historyRow{{B, b1}}, wantTotalPages: 3},
		{name: "halaman di luar data", page: 4, limit: 2, want: []historyRow{}, wantTotalPages: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, page, err := svc.GetMyHistory(t.Context(), f.user.ID, dto.HistoryListQuery{Page: tt.page, Limit: tt.limit})
			mustNoErr(t, err)
			if got := historyRows(t, res); !slices.Equal(got, tt.want) {
				t.Errorf("page %d limit %d → %v; want %v", tt.page, tt.limit, got, tt.want)
			}
			if page.Total != 5 || page.TotalPages != tt.wantTotalPages {
				t.Errorf("meta = total %d, %d halaman; want total 5, %d halaman", page.Total, page.TotalPages, tt.wantTotalPages)
			}
		})
	}

	t.Run("URL foto dari storage", func(t *testing.T) {
		res, _, err := svc.GetMyHistory(t.Context(), f.user.ID, dto.HistoryListQuery{Page: 1, Limit: 2})
		mustNoErr(t, err)
		if got := res[0].Borrowing.ReturnPhotoURL; got == nil || *got != "https://cdn.test/"+returnPhoto {
			t.Errorf("borrowing.returnPhotoUrl = %v; want https://cdn.test/%s", got, returnPhoto)
		}
		if got := res[1].Consignment.PhotoURL; got == nil || *got != "https://cdn.test/"+f.db.consignments[c2].Photo {
			t.Errorf("consignment.photoUrl = %v; want URL storage dari key foto", got)
		}
	})

	t.Run("belum punya pengajuan", func(t *testing.T) {
		kosong := f.db.addUser("Sari", enums.RoleUser)
		res, page, err := svc.GetMyHistory(t.Context(), kosong.ID, dto.HistoryListQuery{Page: 1, Limit: 10})
		mustNoErr(t, err)
		if res == nil || len(res) != 0 || page.Total != 0 {
			t.Errorf("riwayat = %v (total %d); want [] (bukan null), total 0", res, page.Total)
		}
	})
}

// historyRows mengubah response menjadi daftar (jenis, id) sambil memastikan hanya field yang sesuai
// jenisnya yang terisi dan createdAt-nya sama dengan pengajuannya.
func historyRows(t *testing.T, res []dto.HistoryResponse) []historyRow {
	t.Helper()
	rows := make([]historyRow, 0, len(res))
	for _, r := range res {
		switch {
		case r.Type == enums.RequestTypeBorrowing && r.Borrowing != nil && r.Consignment == nil &&
			r.CreatedAt.Equal(r.Borrowing.CreatedAt):
			rows = append(rows, historyRow{r.Type, r.Borrowing.ID})
		case r.Type == enums.RequestTypeConsignment && r.Consignment != nil && r.Borrowing == nil &&
			r.CreatedAt.Equal(r.Consignment.CreatedAt):
			rows = append(rows, historyRow{r.Type, r.Consignment.ID})
		default:
			t.Fatalf("baris riwayat tidak konsisten: %+v", r)
		}
	}
	return rows
}
