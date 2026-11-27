package unit

import (
	"slices"
	"testing"

	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
	"SGEInventory_BE/internal/app/userlog"
)

func TestGetLogs(t *testing.T) {
	t.Parallel()
	db := newFakeDB()
	raka := db.addUser("Raka", enums.RoleUser)
	lembaga := db.addLembaga("BEM")
	it := db.addItem("Proyektor", 5, 5, lembaga.ID)
	b := db.addBorrowing(raka.ID, lembaga.ID, enums.BorrowingPending, line{it.ID, 1}) // log SUBMITTED (BORROWING)
	db.insertLog(entity.NewUserLog(raka.ID, enums.RequestTypeConsignment, 1, enums.ActivitySubmitted))
	svc := userlog.NewUserLogService(&fakeUserLogRepo{db})

	t.Run("semua, terbaru dulu", func(t *testing.T) {
		res, page, err := svc.GetLogs(t.Context(), dto.UserLogListQuery{Page: 1, Limit: 10})
		mustNoErr(t, err)
		if page.Total != 2 || res[0].RequestType != enums.RequestTypeConsignment || res[1].RequestID != b.ID {
			t.Errorf("GetLogs = %+v; want 2 log, CONSIGNMENT (terbaru) lalu BORROWING", res)
		}
		if res[1].Actor == nil || res[1].Actor.FullName != "Raka" {
			t.Errorf("actor = %+v; want pelaku ikut ditampilkan", res[1].Actor)
		}
	})

	t.Run("filter BORROWING", func(t *testing.T) {
		res, _, err := svc.GetLogs(t.Context(), dto.UserLogListQuery{Page: 1, Limit: 10, RequestType: enums.RequestTypeBorrowing})
		mustNoErr(t, err)
		if len(res) != 1 || res[0].RequestType != enums.RequestTypeBorrowing {
			t.Errorf("requestType=BORROWING → %+v; want 1 log BORROWING", res)
		}
	})

	t.Run("search nama pelaku", func(t *testing.T) {
		budi := db.addUser("Budi Santoso", enums.RoleUser)
		db.insertLog(entity.NewUserLog(budi.ID, enums.RequestTypeBorrowing, 9, enums.ActivitySubmitted))

		tests := []struct {
			name        string
			search      string
			requestType string
			wantActors  []uint // urut terbaru dulu
		}{
			{name: "sebagian nama, huruf beda", search: "sANTo", wantActors: []uint{budi.ID}},
			{name: "cocok beberapa log", search: "rak", wantActors: []uint{raka.ID, raka.ID}},
			{name: "digabung requestType", search: "raka", requestType: enums.RequestTypeConsignment, wantActors: []uint{raka.ID}},
			{name: "tidak ada yang cocok", search: "zzz", wantActors: []uint{}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				q := dto.UserLogListQuery{Page: 1, Limit: 10, Search: tt.search, RequestType: tt.requestType}
				res, page, err := svc.GetLogs(t.Context(), q)
				mustNoErr(t, err)
				got := make([]uint, 0, len(res))
				for _, r := range res {
					got = append(got, r.Actor.ID)
				}
				if !slices.Equal(got, tt.wantActors) || page.Total != int64(len(tt.wantActors)) {
					t.Errorf("search=%q requestType=%q → pelaku %v (total %d); want %v", tt.search, tt.requestType, got, page.Total, tt.wantActors)
				}
			})
		}
	})

	t.Run("jenis tidak dikenal", func(t *testing.T) {
		_, _, err := svc.GetLogs(t.Context(), dto.UserLogListQuery{RequestType: "borrowing"})
		assertDomainError(t, err, "INVALID_REQUEST_TYPE", errx.KindInvalid)
	})
}

// GET /api/history/me: hanya aktivitas yang PELAKUNYA user itu sendiri. Aksi admin pada pengajuannya
// (mis. APPROVED) dan aktivitas user lain tidak ikut.
func TestGetMyLogs(t *testing.T) {
	t.Parallel()
	db := newFakeDB()
	raka := db.addUser("Raka", enums.RoleUser)
	budi := db.addUser("Budi", enums.RoleUser)
	admin := db.addUser("Max", enums.RoleAdmin)
	lembaga := db.addLembaga("BEM")
	it := db.addItem("Proyektor", 5, 5, lembaga.ID)

	b := db.addBorrowing(raka.ID, lembaga.ID, enums.BorrowingApproved, line{it.ID, 1}) // SUBMITTED oleh Raka
	db.insertLog(entity.NewUserLog(admin.ID, enums.RequestTypeBorrowing, b.ID, enums.ActivityApproved))
	db.insertLog(entity.NewUserLog(raka.ID, enums.RequestTypeConsignment, 7, enums.ActivitySubmitted))
	db.insertLog(entity.NewUserLog(raka.ID, enums.RequestTypeBorrowing, b.ID, enums.ActivityReturnSubmitted))
	db.addBorrowing(budi.ID, lembaga.ID, enums.BorrowingPending, line{it.ID, 1}) // milik Budi
	svc := userlog.NewUserLogService(&fakeUserLogRepo{db})

	type row struct {
		requestType string
		requestID   uint
		activity    string
	}
	rowsOf := func(t *testing.T, res []dto.UserLogListItemResponse) []row {
		t.Helper()
		got := make([]row, 0, len(res))
		for _, r := range res {
			if r.Actor == nil || r.Actor.ID != raka.ID {
				t.Fatalf("actor = %+v; want Raka (pelaku selalu user itu sendiri)", r.Actor)
			}
			got = append(got, row{r.RequestType, r.RequestID, r.Activity})
		}
		return got
	}
	B, C := enums.RequestTypeBorrowing, enums.RequestTypeConsignment

	tests := []struct {
		name      string
		query     dto.UserLogListQuery
		want      []row
		wantTotal int64
		wantPages int
	}{
		{name: "semua milik sendiri, terbaru dulu", query: dto.UserLogListQuery{Page: 1, Limit: 10},
			want:      []row{{B, b.ID, enums.ActivityReturnSubmitted}, {C, 7, enums.ActivitySubmitted}, {B, b.ID, enums.ActivitySubmitted}},
			wantTotal: 3, wantPages: 1},
		{name: "filter CONSIGNMENT", query: dto.UserLogListQuery{Page: 1, Limit: 10, RequestType: C},
			want: []row{{C, 7, enums.ActivitySubmitted}}, wantTotal: 1, wantPages: 1},
		{name: "halaman 2 limit 2", query: dto.UserLogListQuery{Page: 2, Limit: 2},
			want: []row{{B, b.ID, enums.ActivitySubmitted}}, wantTotal: 3, wantPages: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, page, err := svc.GetMyLogs(t.Context(), raka.ID, tt.query)
			mustNoErr(t, err)
			if got := rowsOf(t, res); !slices.Equal(got, tt.want) {
				t.Errorf("GetMyLogs(%+v) = %v; want %v", tt.query, got, tt.want)
			}
			if page.Total != tt.wantTotal || page.TotalPages != tt.wantPages {
				t.Errorf("meta = total %d, %d halaman; want %d, %d", page.Total, page.TotalPages, tt.wantTotal, tt.wantPages)
			}
		})
	}

	t.Run("belum punya aktivitas → [] bukan null", func(t *testing.T) {
		sari := db.addUser("Sari", enums.RoleUser)
		res, page, err := svc.GetMyLogs(t.Context(), sari.ID, dto.UserLogListQuery{Page: 1, Limit: 10})
		mustNoErr(t, err)
		if res == nil || len(res) != 0 || page.Total != 0 {
			t.Errorf("GetMyLogs = %#v (total %d); want [] dan total 0", res, page.Total)
		}
	})

	t.Run("jenis tidak dikenal", func(t *testing.T) {
		_, _, err := svc.GetMyLogs(t.Context(), raka.ID, dto.UserLogListQuery{RequestType: "borrowing"})
		assertDomainError(t, err, "INVALID_REQUEST_TYPE", errx.KindInvalid)
	})
}
