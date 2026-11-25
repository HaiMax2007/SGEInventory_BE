package unit

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
)

// approvalFixture: Raka mengajukan 2 penitipan (masing-masing 2 barang) lewat service sungguhan.
type approvalFixture struct {
	*consignmentFixture
	admin       *entity.User
	first       uint // penitipan yang diproses di tes
	other       uint // penitipan lain, tidak boleh ikut berubah
	firstPhoto  string
	otherItemID []uint
}

func newApprovalFixture(t *testing.T) *approvalFixture {
	t.Helper()
	f := &approvalFixture{consignmentFixture: newConsignmentFixture()}
	f.admin = f.db.addUser("Max", enums.RoleAdmin)
	submit := func() uint {
		res, err := f.svc.CreateConsignment(t.Context(), f.user.ID, validConsignment(t, f.lembaga.ID))
		mustNoErr(t, err)
		return res.ID
	}
	f.first = submit()
	f.other = submit()
	f.firstPhoto = f.db.consignments[f.first].Photo
	f.otherItemID = f.itemIDsOf(f.other)
	return f
}

// itemIDsOf mengembalikan id barang titipan milik satu penitipan (urut naik).
func (f *approvalFixture) itemIDsOf(consignmentID uint) []uint {
	var ids []uint
	for _, id := range sortedIDs(f.db.items) {
		if c := f.db.items[id].ConsignmentID; c != nil && *c == consignmentID {
			ids = append(ids, id)
		}
	}
	return ids
}

func (f *approvalFixture) lastLog(consignmentID uint) entity.UserLog {
	logs := f.db.consignmentLogs(consignmentID)
	return logs[len(logs)-1]
}

func TestApproveConsignment(t *testing.T) {
	t.Parallel()
	f := newApprovalFixture(t)
	itemsBefore := f.itemIDsOf(f.first)

	res, err := f.svc.ApproveConsignment(t.Context(), f.first, f.admin.ID)
	mustNoErr(t, err)

	if got := f.db.consignments[f.first].Status; got != enums.ConsignmentApproved || res.Status != enums.ConsignmentApproved {
		t.Errorf("status tersimpan %q, response %q; want APPROVED", got, res.Status)
	}
	// Barang titipan tetap ada dan statusnya TERSEDIA (dihitung dari stok).
	if got := f.itemIDsOf(f.first); !slices.Equal(got, itemsBefore) || len(res.Items) != 2 {
		t.Errorf("barang = %v (response %d); want tetap %v", got, len(res.Items), itemsBefore)
	}
	for _, id := range itemsBefore {
		if st := f.db.items[id].Status(); st != enums.ItemStatusAvailable {
			t.Errorf("status barang %d = %s; want TERSEDIA", id, st)
		}
	}
	if l := f.lastLog(f.first); l.RequestActivity != enums.ActivityApproved || l.UserID != f.admin.ID {
		t.Errorf("log terakhir = %s oleh %d; want APPROVED oleh admin", l.RequestActivity, l.UserID)
	}
	if res.PhotoURL == nil || len(f.storage.deleted) != 0 {
		t.Errorf("photoUrl %v, foto dihapus %v; want foto tetap ada", res.PhotoURL, f.storage.deleted)
	}
	if f.db.consignments[f.other].Status != enums.ConsignmentPending {
		t.Error("penitipan lain ikut berubah")
	}
	if !slices.Contains(f.db.calls, "lock consignment") {
		t.Error("baris penitipan tidak dikunci (FindByIDForUpdate tidak dipanggil)")
	}
}

func TestRejectConsignment_DeletesItemsAndPhoto(t *testing.T) {
	t.Parallel()
	f := newApprovalFixture(t)

	res, err := f.svc.RejectConsignment(t.Context(), f.first, f.admin.ID)
	mustNoErr(t, err)

	stored := f.db.consignments[f.first]
	if stored.Status != enums.ConsignmentRejected || stored.Photo != "" {
		t.Errorf("tersimpan status %q foto %q; want REJECTED tanpa foto (baris penitipan tetap ada)", stored.Status, stored.Photo)
	}
	if got := f.itemIDsOf(f.first); len(got) != 0 {
		t.Errorf("barang penitipan yang ditolak masih ada: %v; want terhapus permanen", got)
	}
	if got := f.itemIDsOf(f.other); !slices.Equal(got, f.otherItemID) {
		t.Errorf("barang penitipan lain = %v; want tetap %v", got, f.otherItemID)
	}
	if !slices.Equal(f.storage.deleted, []string{f.firstPhoto}) {
		t.Errorf("foto dihapus = %v; want [%s]", f.storage.deleted, f.firstPhoto)
	}
	if l := f.lastLog(f.first); l.RequestActivity != enums.ActivityRejected || l.UserID != f.admin.ID {
		t.Errorf("log terakhir = %s oleh %d; want REJECTED oleh admin", l.RequestActivity, l.UserID)
	}
	if res.Status != enums.ConsignmentRejected || len(res.Items) != 0 || res.Items == nil || res.PhotoURL != nil {
		t.Errorf("response status %q items %#v photoUrl %v; want REJECTED, [] dan null", res.Status, res.Items, res.PhotoURL)
	}
}

// Gagal menghapus file foto tidak membatalkan penolakan (database sudah benar); errornya hanya dicatat di log.
func TestRejectConsignment_DeletePhotoFails_StillRejected(t *testing.T) {
	t.Parallel()
	f := newApprovalFixture(t)
	f.storage.deleteErr = errors.New("cloudinary tidak bisa dihubungi")

	_, err := f.svc.RejectConsignment(t.Context(), f.first, f.admin.ID)
	mustNoErr(t, err)
	if f.db.consignments[f.first].Status != enums.ConsignmentRejected || len(f.itemIDsOf(f.first)) != 0 {
		t.Error("penitipan tidak ditolak walau hanya hapus foto yang gagal")
	}
}

// Transaksi gagal di tengah jalan (status sudah diubah, lalu hapus barang gagal) → semua dibatalkan,
// dan foto TIDAK dihapus (karena penolakan tidak jadi).
func TestRejectConsignment_TxFails_Rollback(t *testing.T) {
	t.Parallel()
	f := newApprovalFixture(t)
	f.repo.deleteItemsErr = errors.New("database mati")
	itemsBefore := f.itemIDsOf(f.first)
	logsBefore := len(f.db.logs)

	_, err := f.svc.RejectConsignment(t.Context(), f.first, f.admin.ID)
	if err == nil || !strings.Contains(err.Error(), "database mati") {
		t.Fatalf("error = %v; want error database", err)
	}
	stored := f.db.consignments[f.first]
	if stored.Status != enums.ConsignmentPending || stored.Photo != f.firstPhoto {
		t.Errorf("status %q foto %q; want tetap PENDING dengan foto lama", stored.Status, stored.Photo)
	}
	if got := f.itemIDsOf(f.first); !slices.Equal(got, itemsBefore) || len(f.db.logs) != logsBefore {
		t.Errorf("barang %v log %d; want tetap %v dan %d", got, len(f.db.logs), itemsBefore, logsBefore)
	}
	if len(f.storage.deleted) != 0 {
		t.Errorf("foto dihapus %v; want tidak ada (penolakan batal)", f.storage.deleted)
	}
}

// ACC / tolak hanya sekali (final). Selain PENDING → 409 dan tidak ada yang berubah.
func TestApproveRejectConsignment_AlreadyProcessed(t *testing.T) {
	t.Parallel()
	actions := map[string]func(f *approvalFixture) error{
		"ACC": func(f *approvalFixture) error {
			_, err := f.svc.ApproveConsignment(t.Context(), f.first, f.admin.ID)
			return err
		},
		"tolak": func(f *approvalFixture) error {
			_, err := f.svc.RejectConsignment(t.Context(), f.first, f.admin.ID)
			return err
		},
	}
	for _, from := range []string{enums.ConsignmentApproved, enums.ConsignmentRejected} {
		for name, action := range actions {
			t.Run(name+" dari "+from, func(t *testing.T) {
				f := newApprovalFixture(t)
				f.db.consignments[f.first].Status = from
				itemsBefore := f.itemIDsOf(f.first)
				logsBefore := len(f.db.logs)

				de := assertDomainError(t, action(f), "CONSIGNMENT_ALREADY_PROCESSED", errx.KindConflict)
				if !strings.Contains(de.Message, from) {
					t.Errorf("pesan %q; want menyebut status %s", de.Message, from)
				}
				if f.db.consignments[f.first].Status != from || !slices.Equal(f.itemIDsOf(f.first), itemsBefore) ||
					len(f.db.logs) != logsBefore || len(f.storage.deleted) != 0 {
					t.Error("data berubah walau ditolak dengan 409")
				}
			})
		}
	}

	t.Run("penitipan tidak ada → 404", func(t *testing.T) {
		f := newApprovalFixture(t)
		_, err := f.svc.ApproveConsignment(t.Context(), 999, f.admin.ID)
		assertDomainError(t, err, "CONSIGNMENT_NOT_FOUND", errx.KindNotFound)
		_, err = f.svc.RejectConsignment(t.Context(), 999, f.admin.ID)
		assertDomainError(t, err, "CONSIGNMENT_NOT_FOUND", errx.KindNotFound)
	})
}

// Detail: pemilik & admin boleh melihat; user lain mendapat 404 (bukan 403).
func TestGetConsignment(t *testing.T) {
	t.Parallel()
	f := newApprovalFixture(t)
	budi := f.db.addUser("Budi", enums.RoleUser)

	res, err := f.svc.GetConsignment(t.Context(), f.first, f.user.ID, false)
	mustNoErr(t, err)
	if res.ID != f.first || len(res.Items) != 2 || res.User == nil || res.User.ID != f.user.ID {
		t.Errorf("detail pemilik = %+v; want penitipan %d lengkap", res, f.first)
	}

	_, err = f.svc.GetConsignment(t.Context(), f.first, f.admin.ID, true)
	mustNoErr(t, err)

	_, err = f.svc.GetConsignment(t.Context(), f.first, budi.ID, false)
	assertDomainError(t, err, "CONSIGNMENT_NOT_FOUND", errx.KindNotFound)

	_, err = f.svc.GetConsignment(t.Context(), 999, f.admin.ID, true)
	assertDomainError(t, err, "CONSIGNMENT_NOT_FOUND", errx.KindNotFound)
}

// Aturan entity: Approve/Reject hanya dari PENDING; Reject mengosongkan foto.
func TestConsignmentEntity_ApproveReject(t *testing.T) {
	t.Parallel()
	newC := func(status string) *entity.Consignment {
		c := entity.NewConsignment(1, 1, "Acara", parseDay(t, dateWIB(1)), parseDay(t, dateWIB(2)), "consignments/a.png", "", nil)
		c.Status = status
		return c
	}

	c := newC(enums.ConsignmentPending)
	mustNoErr(t, c.Approve())
	if c.Status != enums.ConsignmentApproved || c.Photo == "" {
		t.Errorf("Approve: status %q foto %q; want APPROVED, foto tetap", c.Status, c.Photo)
	}

	c = newC(enums.ConsignmentPending)
	mustNoErr(t, c.Reject())
	if c.Status != enums.ConsignmentRejected || c.Photo != "" {
		t.Errorf("Reject: status %q foto %q; want REJECTED tanpa foto", c.Status, c.Photo)
	}

	for _, from := range []string{enums.ConsignmentApproved, enums.ConsignmentRejected} {
		assertDomainError(t, newC(from).Approve(), "CONSIGNMENT_ALREADY_PROCESSED", errx.KindConflict)
		assertDomainError(t, newC(from).Reject(), "CONSIGNMENT_ALREADY_PROCESSED", errx.KindConflict)
	}
}
