package unit

import (
	"testing"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/errx"
	"SGEInventory_BE/internal/app/lembaga"
)

func newLembagaService() (*fakeDB, contracts.LembagaService) {
	db := newFakeDB()
	return db, lembaga.NewLembagaService(&fakeLembagaRepo{db})
}

func TestCreateLembaga(t *testing.T) {
	t.Parallel()
	db, svc := newLembagaService()

	res, err := svc.CreateLembaga(t.Context(), dto.CreateLembagaRequest{Name: "BEM"})
	mustNoErr(t, err)
	if res.ID == 0 || res.Name != "BEM" {
		t.Errorf("CreateLembaga = %+v; want id terisi & nama BEM", res)
	}

	_, err = svc.CreateLembaga(t.Context(), dto.CreateLembagaRequest{Name: "BEM"})
	assertDomainError(t, err, "LEMBAGA_NAME_TAKEN", errx.KindConflict)
	if len(db.lembagas) != 1 {
		t.Errorf("jumlah lembaga = %d; want 1", len(db.lembagas))
	}
}

func TestUpdateLembaga(t *testing.T) {
	t.Parallel()
	db, svc := newLembagaService()
	bem := db.addLembaga("BEM")
	db.addLembaga("DPM")

	res, err := svc.UpdateLembaga(t.Context(), bem.ID, dto.UpdateLembagaRequest{Name: "BEM Pusat"})
	mustNoErr(t, err)
	if res.Name != "BEM Pusat" {
		t.Errorf("nama = %q; want BEM Pusat", res.Name)
	}

	_, err = svc.UpdateLembaga(t.Context(), bem.ID, dto.UpdateLembagaRequest{Name: "DPM"})
	assertDomainError(t, err, "LEMBAGA_NAME_TAKEN", errx.KindConflict)

	_, err = svc.UpdateLembaga(t.Context(), 999, dto.UpdateLembagaRequest{Name: "HIMA"})
	assertDomainError(t, err, "LEMBAGA_NOT_FOUND", errx.KindNotFound)
}

func TestDeleteLembaga(t *testing.T) {
	t.Parallel()

	t.Run("masih menjadi asal barang ditolak", func(t *testing.T) {
		t.Parallel()
		db, svc := newLembagaService()
		bem := db.addLembaga("BEM")
		db.addItem("Proyektor", 1, 1, bem.ID)

		assertDomainError(t, svc.DeleteLembaga(t.Context(), bem.ID), "LEMBAGA_IN_USE", errx.KindConflict)
		if db.lembagas[bem.ID].DeletedAt.Valid {
			t.Error("lembaga terhapus; want tidak")
		}
	})

	t.Run("barangnya sudah dihapus semua → boleh", func(t *testing.T) {
		t.Parallel()
		db, svc := newLembagaService()
		bem := db.addLembaga("BEM")
		item := db.addItem("Proyektor", 1, 1, bem.ID)
		db.softDeleteItem(item.ID)

		mustNoErr(t, svc.DeleteLembaga(t.Context(), bem.ID))
		_, err := svc.GetLembaga(t.Context(), bem.ID)
		assertDomainError(t, err, "LEMBAGA_NOT_FOUND", errx.KindNotFound)
	})
}

func TestGetLembagas_SortedByName(t *testing.T) {
	t.Parallel()
	db, svc := newLembagaService()
	db.addLembaga("HMDTIF")
	db.addLembaga("BEM")
	db.addLembaga("DPM")

	res, page, err := svc.GetLembagas(t.Context(), dto.LembagaListQuery{Page: 1, Limit: 2})
	mustNoErr(t, err)
	if len(res) != 2 || res[0].Name != "BEM" || res[1].Name != "DPM" || page.Total != 3 {
		t.Errorf("GetLembagas(limit 2) = %+v total %d; want [BEM DPM] total 3", res, page.Total)
	}
}
