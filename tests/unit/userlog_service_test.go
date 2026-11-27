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
