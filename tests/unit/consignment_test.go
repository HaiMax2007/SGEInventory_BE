package unit

import (
	"errors"
	"regexp"
	"slices"
	"strings"
	"testing"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
	"SGEInventory_BE/internal/app/consignment"
	"SGEInventory_BE/pkg/validator"
)

// consignmentPhotoPattern: key foto harus acak (32 karakter hex) di folder consignments/.
var consignmentPhotoPattern = regexp.MustCompile(`^consignments/[0-9a-f]{32}\.(jpg|png|webp)$`)

// ---------- fixture ----------

type consignmentFixture struct {
	db      *fakeDB
	repo    *fakeConsignmentRepo
	storage *fakeStorage
	svc     contracts.ConsignmentService
	user    *entity.User
	lembaga *entity.Lembaga
}

func newConsignmentFixture() *consignmentFixture {
	db := newFakeDB()
	f := &consignmentFixture{db: db, repo: &fakeConsignmentRepo{db: db}, storage: newFakeStorage()}
	f.user = db.addUser("Raka", enums.RoleUser)
	f.lembaga = db.addLembaga("HMDTIF")
	f.svc = consignment.NewConsignmentService(f.repo, &fakeUserLogRepo{db}, &fakeLembagaRepo{db}, &fakeTransactor{db: db}, f.storage)
	return f
}

// validConsignment membuat form penitipan yang lolos semua aturan: mulai besok, selesai lusa.
func validConsignment(t *testing.T, lembagaID uint) dto.CreateConsignmentRequest {
	t.Helper()
	return dto.CreateConsignmentRequest{
		LembagaID: lembagaID,
		EventName: "Pameran HMDTIF",
		Items: []dto.ConsignmentItemRequest{
			{Name: "Kursi lipat", Quantity: 10},
			{Name: "Meja panjang", Quantity: 2},
		},
		StartDate: dateWIB(1),
		EndDate:   dateWIB(3),
		Photo:     photoFile(t, "barang.png", pngContent),
		Notes:     "Tolong disimpan di gudang belakang",
	}
}

func withConsignment(t *testing.T, change func(r *dto.CreateConsignmentRequest)) *dto.CreateConsignmentRequest {
	t.Helper()
	r := validConsignment(t, 1)
	change(&r)
	return &r
}

// ---------- validasi per field (tag `validate` + FieldErrors) ----------

func TestValidateConsignmentRequest(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   *dto.CreateConsignmentRequest
		wantErr error // nil = harus lolos
	}{
		{name: "valid", input: withConsignment(t, func(r *dto.CreateConsignmentRequest) {})},
		{name: "catatan kosong boleh", input: withConsignment(t, func(r *dto.CreateConsignmentRequest) { r.Notes = "" })},
		{name: "tanpa lembaga", input: withConsignment(t, func(r *dto.CreateConsignmentRequest) { r.LembagaID = 0 }), wantErr: errx.ErrInvalidConsignmentLembaga},
		{name: "nama acara hanya spasi", input: withConsignment(t, func(r *dto.CreateConsignmentRequest) { r.EventName = "   " }), wantErr: errx.ErrInvalidEventName},
		{name: "nama acara 2 karakter", input: withConsignment(t, func(r *dto.CreateConsignmentRequest) { r.EventName = "ab" }), wantErr: errx.ErrInvalidEventName},
		{name: "tanpa barang", input: withConsignment(t, func(r *dto.CreateConsignmentRequest) { r.Items = nil }), wantErr: errx.ErrInvalidConsignmentItems},
		{name: "daftar barang kosong", input: withConsignment(t, func(r *dto.CreateConsignmentRequest) { r.Items = []dto.ConsignmentItemRequest{} }), wantErr: errx.ErrInvalidConsignmentItems},
		{name: "21 barang", input: withConsignment(t, func(r *dto.CreateConsignmentRequest) {
			r.Items = make([]dto.ConsignmentItemRequest, 21)
			for i := range r.Items {
				r.Items[i] = dto.ConsignmentItemRequest{Name: "Barang " + strings.Repeat("x", i+1), Quantity: 1}
			}
		}), wantErr: errx.ErrInvalidConsignmentItems},
		{name: "nama barang 1 karakter", input: withConsignment(t, func(r *dto.CreateConsignmentRequest) { r.Items[1].Name = "A" }), wantErr: errx.ErrInvalidConsignmentItems},
		{name: "jumlah barang 0", input: withConsignment(t, func(r *dto.CreateConsignmentRequest) { r.Items[0].Quantity = 0 }), wantErr: errx.ErrInvalidConsignmentItems},
		{name: "jumlah barang lebih dari 1000000", input: withConsignment(t, func(r *dto.CreateConsignmentRequest) { r.Items[0].Quantity = 1000001 }), wantErr: errx.ErrInvalidConsignmentItems},
		{name: "tanggal mulai kosong", input: withConsignment(t, func(r *dto.CreateConsignmentRequest) { r.StartDate = "" }), wantErr: errx.ErrInvalidStartDate},
		{name: "format tanggal selesai salah", input: withConsignment(t, func(r *dto.CreateConsignmentRequest) { r.EndDate = "30-11-2026" }), wantErr: errx.ErrInvalidEndDate},
		{name: "tanpa foto", input: withConsignment(t, func(r *dto.CreateConsignmentRequest) { r.Photo = nil }), wantErr: errx.ErrInvalidConsignmentPhoto},
		{name: "catatan 1001 karakter", input: withConsignment(t, func(r *dto.CreateConsignmentRequest) { r.Notes = strings.Repeat("a", 1001) }), wantErr: errx.ErrInvalidConsignmentNotes},
		// yang dikirim error field PERTAMA sesuai urutan form: lembaga → nama acara → barang → tanggal → foto
		{name: "semua kosong → lembaga dulu", input: &dto.CreateConsignmentRequest{}, wantErr: errx.ErrInvalidConsignmentLembaga},
		{name: "barang & foto salah → barang dulu", input: withConsignment(t, func(r *dto.CreateConsignmentRequest) {
			r.Items, r.Photo = nil, nil
		}), wantErr: errx.ErrInvalidConsignmentItems},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validator.ValidateStruct(tt.input)
			if tt.wantErr == nil {
				mustNoErr(t, err)
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidateStruct = %v; want %v", err, tt.wantErr)
			}
		})
	}
}

// ParseItems: field "items" berisi teks JSON. JSON yang salah membuat Items kosong, lalu validator
// membalasnya dengan ErrInvalidConsignmentItems.
func TestCreateConsignmentRequest_ParseItems(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
		want []dto.ConsignmentItemRequest // nil = Items kosong
	}{
		{name: "valid + nama di-trim", raw: `[{"name":"  Kursi lipat ","quantity":10},{"name":"Meja","quantity":2}]`,
			want: []dto.ConsignmentItemRequest{{Name: "Kursi lipat", Quantity: 10}, {Name: "Meja", Quantity: 2}}},
		{name: "kosong", raw: ""},
		{name: "JSON rusak", raw: `[{"name":"Kursi"`},
		{name: "quantity berupa teks", raw: `[{"name":"Kursi","quantity":"sepuluh"}]`},
		{name: "bukan array", raw: `{"name":"Kursi","quantity":1}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var r dto.CreateConsignmentRequest
			r.ParseItems(tt.raw)
			if len(r.Items) != len(tt.want) {
				t.Fatalf("Items = %+v; want %+v", r.Items, tt.want)
			}
			for i := range tt.want {
				if r.Items[i] != tt.want[i] {
					t.Errorf("Items[%d] = %+v; want %+v", i, r.Items[i], tt.want[i])
				}
			}
		})
	}
}

// ---------- service ----------

func TestCreateConsignment_Success(t *testing.T) {
	t.Parallel()
	f := newConsignmentFixture()
	req := validConsignment(t, f.lembaga.ID)
	req.StartDate = dateWIB(0) // hari ini boleh
	req.EndDate = dateWIB(7)   // maksimal +7 hari

	res, err := f.svc.CreateConsignment(t.Context(), f.user.ID, req)
	mustNoErr(t, err)

	stored := f.db.consignments[res.ID]
	if stored == nil || stored.Status != enums.ConsignmentPending || stored.UserID != f.user.ID || stored.LembagaID != f.lembaga.ID {
		t.Fatalf("tersimpan %+v; want PENDING milik user & lembaga form", stored)
	}
	if !consignmentPhotoPattern.MatchString(stored.Photo) || string(f.storage.files[stored.Photo]) != string(pngContent) {
		t.Errorf("foto %q; want key acak consignments/<hex>.png berisi file yang diunggah", stored.Photo)
	}

	// Setiap barang menjadi baris items bertipe CONSIGNMENT milik penitipan ini.
	var titipan []*entity.Item
	for _, id := range sortedIDs(f.db.items) {
		titipan = append(titipan, f.db.items[id])
	}
	if len(titipan) != 2 {
		t.Fatalf("items tersimpan %d; want 2", len(titipan))
	}
	for i, want := range req.Items {
		it := titipan[i]
		if it.Name != want.Name || it.TotalQuantity != want.Quantity || it.AvailableQuantity != want.Quantity ||
			it.Type != enums.ItemTypeConsignment || it.LembagaID != f.lembaga.ID || it.ConsignmentID == nil || *it.ConsignmentID != res.ID {
			t.Errorf("item %d = %+v; want %s ×%d CONSIGNMENT milik penitipan %d", i, it, want.Name, want.Quantity, res.ID)
		}
	}

	logs := f.db.consignmentLogs(res.ID)
	if len(logs) != 1 || logs[0].RequestActivity != enums.ActivitySubmitted || logs[0].UserID != f.user.ID {
		t.Errorf("log = %+v; want 1 SUBMITTED oleh penitip", logs)
	}

	if res.Status != enums.ConsignmentPending || res.EventName != "Pameran HMDTIF" || res.StartDate != req.StartDate ||
		res.EndDate != req.EndDate || res.Notes != req.Notes || res.User == nil || res.Lembaga == nil || len(res.Items) != 2 {
		t.Errorf("response = %+v; want data form lengkap", res)
	}
	if res.PhotoURL == nil || *res.PhotoURL != "https://cdn.test/"+stored.Photo {
		t.Errorf("photoUrl = %v; want URL dari storage", res.PhotoURL)
	}
	if res.Items[0].Name != "Kursi lipat" || res.Items[0].Quantity != 10 {
		t.Errorf("items[0] = %+v; want Kursi lipat ×10", res.Items[0])
	}
}

// Untuk semua kasus gagal: tidak ada penitipan, barang, log, atau foto yang tersimpan.
func TestCreateConsignment_Errors_NothingSaved(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		change   func(t *testing.T, r *dto.CreateConsignmentRequest)
		wantCode string
		wantKind errx.Kind
	}{
		{name: "nama barang dobel (beda huruf besar/kecil)", change: func(_ *testing.T, r *dto.CreateConsignmentRequest) {
			r.Items = append(r.Items, dto.ConsignmentItemRequest{Name: "KURSI LIPAT", Quantity: 1})
		}, wantCode: "DUPLICATE_CONSIGNMENT_ITEM", wantKind: errx.KindInvalid},
		{name: "tanggal mulai kemarin", change: func(_ *testing.T, r *dto.CreateConsignmentRequest) { r.StartDate = dateWIB(-1) },
			wantCode: "START_DATE_PAST", wantKind: errx.KindInvalid},
		{name: "selesai di hari yang sama", change: func(_ *testing.T, r *dto.CreateConsignmentRequest) { r.EndDate = r.StartDate },
			wantCode: "END_NOT_AFTER_START", wantKind: errx.KindInvalid},
		{name: "lebih dari 7 hari", change: func(_ *testing.T, r *dto.CreateConsignmentRequest) { r.EndDate = dateWIB(9) },
			wantCode: "CONSIGNMENT_TOO_LONG", wantKind: errx.KindInvalid},
		{name: "lembaga tidak ada", change: func(_ *testing.T, r *dto.CreateConsignmentRequest) { r.LembagaID = 999 },
			wantCode: "LEMBAGA_NOT_FOUND", wantKind: errx.KindNotFound},
		{name: "foto bukan gambar (nama .jpg)", change: func(t *testing.T, r *dto.CreateConsignmentRequest) {
			r.Photo = photoFile(t, "foto.jpg", []byte("ini teks, bukan gambar"))
		}, wantCode: "INVALID_CONSIGNMENT_PHOTO", wantKind: errx.KindInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newConsignmentFixture()
			req := validConsignment(t, f.lembaga.ID)
			tt.change(t, &req)

			_, err := f.svc.CreateConsignment(t.Context(), f.user.ID, req)
			assertDomainError(t, err, tt.wantCode, tt.wantKind)
			assertNothingSaved(t, f)
		})
	}
}

// Penyimpanan ke database gagal → foto yang sudah terunggah dihapus lagi (tidak ada file yatim).
func TestCreateConsignment_RepoFails_DeletesPhoto(t *testing.T) {
	t.Parallel()
	f := newConsignmentFixture()
	f.repo.createErr = errors.New("database mati")

	_, err := f.svc.CreateConsignment(t.Context(), f.user.ID, validConsignment(t, f.lembaga.ID))
	if err == nil || !strings.Contains(err.Error(), "database mati") {
		t.Fatalf("error = %v; want error database", err)
	}
	if len(f.storage.deleted) != 1 || !consignmentPhotoPattern.MatchString(f.storage.deleted[0]) {
		t.Errorf("foto dihapus = %v; want 1 foto consignments/...", f.storage.deleted)
	}
	assertNothingSaved(t, f)
}

// Unggah foto gagal → tidak ada yang disimpan ke database.
func TestCreateConsignment_StorageFails_NothingSaved(t *testing.T) {
	t.Parallel()
	f := newConsignmentFixture()
	f.storage.saveErr = errors.New("cloudinary tidak bisa dihubungi")

	_, err := f.svc.CreateConsignment(t.Context(), f.user.ID, validConsignment(t, f.lembaga.ID))
	if err == nil {
		t.Fatal("error = nil; want error storage")
	}
	assertNothingSaved(t, f)
}

func assertNothingSaved(t *testing.T, f *consignmentFixture) {
	t.Helper()
	if len(f.db.consignments) != 0 || len(f.db.items) != 0 || len(f.db.logs) != 0 || len(f.storage.files) != 0 {
		t.Errorf("tersimpan: %d penitipan, %d barang, %d log, %d foto; want semuanya 0",
			len(f.db.consignments), len(f.db.items), len(f.db.logs), len(f.storage.files))
	}
}

// ---------- daftar penitipan ----------

// listFixture: Raka punya 2 penitipan (yang kedua sudah APPROVED), Budi punya 1.
func newConsignmentListFixture(t *testing.T) (f *consignmentFixture, raka1, raka2, budi1 uint) {
	t.Helper()
	f = newConsignmentFixture()
	budi := f.db.addUser("Budi", enums.RoleUser)
	submit := func(userID uint, event string) uint {
		req := validConsignment(t, f.lembaga.ID)
		req.EventName = event
		res, err := f.svc.CreateConsignment(t.Context(), userID, req)
		mustNoErr(t, err)
		return res.ID
	}
	raka1 = submit(f.user.ID, "Pameran 1")
	raka2 = submit(f.user.ID, "Pameran 2")
	budi1 = submit(budi.ID, "Bazar Budi")
	f.db.consignments[raka2].Status = enums.ConsignmentApproved
	return f, raka1, raka2, budi1
}

func consignmentIDs(res []dto.ConsignmentResponse) []uint {
	ids := make([]uint, 0, len(res))
	for _, r := range res {
		ids = append(ids, r.ID)
	}
	return ids
}

// User hanya melihat penitipannya sendiri (terbaru dulu), lengkap dengan barang & URL foto.
func TestGetMyConsignments_OnlyOwn(t *testing.T) {
	t.Parallel()
	f, raka1, raka2, _ := newConsignmentListFixture(t)

	res, page, err := f.svc.GetMyConsignments(t.Context(), f.user.ID, dto.ConsignmentListQuery{Page: 1, Limit: 10})
	mustNoErr(t, err)
	if got, want := consignmentIDs(res), []uint{raka2, raka1}; !slices.Equal(got, want) {
		t.Fatalf("GetMyConsignments = id %v; want %v (milik sendiri, terbaru dulu)", got, want)
	}
	if page.Total != 2 {
		t.Errorf("meta total = %d; want 2", page.Total)
	}
	first := res[0]
	if first.Status != enums.ConsignmentApproved || len(first.Items) != 2 || first.User == nil || first.User.ID != f.user.ID ||
		first.PhotoURL == nil || !strings.HasPrefix(*first.PhotoURL, "https://cdn.test/consignments/") {
		t.Errorf("res[0] = %+v; want APPROVED, 2 barang, user Raka, photoUrl dari storage", first)
	}

	// User yang belum pernah menitipkan → daftar kosong [] (bukan null).
	none, _, err := f.svc.GetMyConsignments(t.Context(), 999, dto.ConsignmentListQuery{Page: 1, Limit: 10})
	mustNoErr(t, err)
	if none == nil || len(none) != 0 {
		t.Errorf("tanpa penitipan = %#v; want [] (slice kosong)", none)
	}
}

// CountConsignments menghitung SEMUA penitipan: semua status & semua penitip.
func TestCountConsignments(t *testing.T) {
	t.Parallel()
	empty := newConsignmentFixture()
	res, err := empty.svc.CountConsignments(t.Context())
	mustNoErr(t, err)
	if res.Total != 0 {
		t.Errorf("CountConsignments (kosong) = %d; want 0", res.Total)
	}

	f, _, _, _ := newConsignmentListFixture(t) // Raka 2 (PENDING + APPROVED), Budi 1
	res, err = f.svc.CountConsignments(t.Context())
	mustNoErr(t, err)
	if res.Total != 3 {
		t.Errorf("CountConsignments = %d; want 3 (semua status, semua user)", res.Total)
	}
}

// Admin melihat semua penitipan, bisa difilter status; status tidak dikenal → 400.
func TestGetConsignments_Admin(t *testing.T) {
	t.Parallel()
	f, raka1, raka2, budi1 := newConsignmentListFixture(t)

	tests := []struct {
		name   string
		status string
		want   []uint
	}{
		{name: "semua", want: []uint{budi1, raka2, raka1}},
		{name: "PENDING", status: enums.ConsignmentPending, want: []uint{budi1, raka1}},
		{name: "APPROVED", status: enums.ConsignmentApproved, want: []uint{raka2}},
		{name: "REJECTED (kosong)", status: enums.ConsignmentRejected, want: []uint{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, _, err := f.svc.GetConsignments(t.Context(), dto.ConsignmentListQuery{Page: 1, Limit: 10, Status: tt.status})
			mustNoErr(t, err)
			if got := consignmentIDs(res); !slices.Equal(got, tt.want) {
				t.Errorf("GetConsignments(status=%q) = id %v; want %v", tt.status, got, tt.want)
			}
		})
	}

	t.Run("pagination", func(t *testing.T) {
		res, page, err := f.svc.GetConsignments(t.Context(), dto.ConsignmentListQuery{Page: 2, Limit: 2})
		mustNoErr(t, err)
		if got := consignmentIDs(res); !slices.Equal(got, []uint{raka1}) || page.Total != 3 || page.TotalPages != 2 {
			t.Errorf("halaman 2 = id %v total %d pages %d; want [%d], 3, 2", got, page.Total, page.TotalPages, raka1)
		}
	})
	t.Run("status tidak dikenal → 400", func(t *testing.T) {
		_, _, err := f.svc.GetConsignments(t.Context(), dto.ConsignmentListQuery{Page: 1, Limit: 10, Status: "pending"})
		assertDomainError(t, err, "INVALID_CONSIGNMENT_STATUS", errx.KindInvalid)
	})
}
