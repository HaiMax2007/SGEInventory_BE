package unit

import (
	"testing"
	"time"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
	"SGEInventory_BE/internal/app/borrowing"
)

// mustNoErr menghentikan tes jika err tidak nil (dipakai untuk langkah yang harus berhasil).
func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("error tidak terduga: %v", err)
	}
}

// assertDomainError memastikan err adalah errx.DomainError dengan Code & Kind tertentu.
// Kind menentukan status HTTP (KindInvalid=400, KindNotFound=404, KindConflict=409, ...).
func assertDomainError(t *testing.T, err error, wantCode string, wantKind errx.Kind) *errx.DomainError {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil; want %s", wantCode)
	}
	de, ok := errx.As(err)
	if !ok {
		t.Fatalf("error = %v (%T); want DomainError %s", err, err, wantCode)
	}
	if de.Code != wantCode || de.Kind != wantKind {
		t.Fatalf("error = %s (kind %d) %q; want %s (kind %d)", de.Code, de.Kind, de.Message, wantCode, wantKind)
	}
	return de
}

var wib = time.FixedZone("WIB", 7*60*60)

// dateWIB mengembalikan tanggal "YYYY-MM-DD" hari ini (WIB) ditambah days hari.
// Sama dengan cara service menentukan "hari ini", jadi tes tidak bergantung pada tanggal tertentu.
func dateWIB(days int) string {
	return time.Now().In(wib).AddDate(0, 0, days).Format(dto.DateLayout)
}

// parseDay mengubah "YYYY-MM-DD" menjadi time.Time (jam 00:00 UTC, sama seperti service).
func parseDay(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse(dto.DateLayout, s)
	mustNoErr(t, err)
	return d
}

// ---------- fixture peminjaman ----------

// borrowingFixture menyiapkan BorrowingService dengan repository palsu + 1 user, 1 admin, 1 lembaga.
type borrowingFixture struct {
	db      *fakeDB
	tx      *fakeTransactor
	storage *fakeStorage
	svc     contracts.BorrowingService
	user    *entity.User
	admin   *entity.User
	lembaga *entity.Lembaga
}

func newBorrowingFixture(t *testing.T) *borrowingFixture {
	t.Helper()
	db := newFakeDB()
	f := &borrowingFixture{db: db, tx: &fakeTransactor{db: db}, storage: newFakeStorage()}
	f.user = db.addUser("Raka", enums.RoleUser)
	f.admin = db.addUser("Max", enums.RoleAdmin)
	f.lembaga = db.addLembaga("BEM")
	f.svc = borrowing.NewBorrowingService(
		&fakeBorrowingRepo{db}, &fakeUserLogRepo{db}, &fakeItemRepo{db}, &fakeLembagaRepo{db}, f.tx, f.storage,
	)
	return f
}

// activities mengubah daftar log menjadi daftar nama aktivitas, mis. [SUBMITTED APPROVED].
func activities(logs []entity.UserLog) []string {
	res := make([]string, 0, len(logs))
	for _, l := range logs {
		res = append(res, l.RequestActivity)
	}
	return res
}
