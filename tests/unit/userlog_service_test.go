package unit

import (
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

	t.Run("jenis tidak dikenal", func(t *testing.T) {
		_, _, err := svc.GetLogs(t.Context(), dto.UserLogListQuery{RequestType: "borrowing"})
		assertDomainError(t, err, "INVALID_REQUEST_TYPE", errx.KindInvalid)
	})
}
