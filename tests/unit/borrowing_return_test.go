package unit

import (
	"bytes"
	"errors"
	"mime/multipart"
	"regexp"
	"slices"
	"testing"

	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
)

// Tes tahap 3: pengembalian (user unggah foto → admin ACC / tolak).

// Isi file contoh. http.DetectContentType mengenali jenis file dari "tanda tangan" di awal isinya.
var (
	pngContent  = append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 64)...)
	jpegContent = append([]byte("\xff\xd8\xff\xe0"), make([]byte, 64)...)
	webpContent = append([]byte("RIFF\x00\x00\x00\x00WEBPVP8 "), make([]byte, 64)...)
)

// photoPathPattern: path foto harus acak (32 karakter hex) di folder returns/.
var photoPathPattern = regexp.MustCompile(`^returns/[0-9a-f]{32}\.(jpg|png|webp)$`)

// photoFile membuat *multipart.FileHeader sungguhan (seperti hasil c.FormFile("photo") di handler)
// dengan nama file & isi tertentu.
func photoFile(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("photo", filename)
	mustNoErr(t, err)
	_, err = part.Write(content)
	mustNoErr(t, err)
	mustNoErr(t, w.Close())

	form, err := multipart.NewReader(&body, w.Boundary()).ReadForm(16 << 20)
	mustNoErr(t, err)
	t.Cleanup(func() { _ = form.RemoveAll() })
	return form.File["photo"][0]
}

// returnFixture: satu pengajuan milik f.user berisi Proyektor ×2 (total 5) dan Kabel ×1 (total 3),
// dengan stok yang sudah berkurang karena pengajuan ini.
type returnFixture struct {
	*borrowingFixture
	borrowing *entity.Borrowing
	proyektor *entity.Item
	kabel     *entity.Item
}

func newReturnFixture(t *testing.T, status string) *returnFixture {
	t.Helper()
	f := &returnFixture{borrowingFixture: newBorrowingFixture(t)}
	f.proyektor = f.db.addItem("Proyektor", 5, 3, f.lembaga.ID) // 2 sedang dipinjam
	f.kabel = f.db.addItem("Kabel HDMI", 3, 2, f.lembaga.ID)    // 1 sedang dipinjam
	f.borrowing = f.db.addBorrowing(f.user.ID, f.lembaga.ID, status,
		line{itemID: f.proyektor.ID, qty: 2}, line{itemID: f.kabel.ID, qty: 1})
	return f
}

// withPhoto memasang foto lama pada pengajuan (di database palsu dan di storage palsu).
func (f *returnFixture) withPhoto(path string) {
	f.db.borrowings[f.borrowing.ID].ReturnPhoto = &path
	f.storage.files[path] = pngContent
}

func (f *returnFixture) stored() *entity.Borrowing { return f.db.borrowings[f.borrowing.ID] }

func (f *returnFixture) stock() (proyektor, kabel int) {
	return f.db.items[f.proyektor.ID].AvailableQuantity, f.db.items[f.kabel.ID].AvailableQuantity
}

// ---------- user mengunggah foto ----------

func TestSubmitReturn_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		from    string
		content []byte
		wantExt string
	}{
		{name: "dari APPROVED (PNG)", from: enums.BorrowingApproved, content: pngContent, wantExt: ".png"},
		{name: "dari RETURN_REJECTED (unggah ulang, JPG)", from: enums.BorrowingReturnRejected, content: jpegContent, wantExt: ".jpg"},
		{name: "WebP", from: enums.BorrowingApproved, content: webpContent, wantExt: ".webp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newReturnFixture(t, tt.from)
			// Nama file kiriman sengaja salah (.txt): ekstensi diambil dari ISI file, bukan dari namanya.
			req := dto.SubmitReturnRequest{Photo: photoFile(t, "foto.txt", tt.content)}

			res, err := f.svc.SubmitReturn(t.Context(), f.borrowing.ID, f.user.ID, req)
			mustNoErr(t, err)

			stored := f.stored()
			if stored.Status != enums.BorrowingReturnPending || stored.ReturnPhoto == nil {
				t.Fatalf("tersimpan status %q foto %v; want RETURN_PENDING dengan foto", stored.Status, stored.ReturnPhoto)
			}
			path := *stored.ReturnPhoto
			if !photoPathPattern.MatchString(path) || path[len(path)-len(tt.wantExt):] != tt.wantExt {
				t.Errorf("path foto = %q; want returns/<32 hex>%s", path, tt.wantExt)
			}
			if !bytes.Equal(f.storage.files[path], tt.content) {
				t.Errorf("isi file tersimpan tidak sama dengan yang diunggah")
			}
			if res.Status != enums.BorrowingReturnPending || res.ReturnPhotoURL == nil || *res.ReturnPhotoURL != "/uploads/"+path {
				t.Errorf("respons status %q url %v; want RETURN_PENDING dan /uploads/%s", res.Status, res.ReturnPhotoURL, path)
			}
			// Stok belum berubah: barang baru dianggap kembali setelah admin ACC.
			if p, k := f.stock(); p != 3 || k != 2 {
				t.Errorf("stok = %d, %d; want tetap 3, 2", p, k)
			}
			logs := f.db.logsFor(f.borrowing.ID)
			last := logs[len(logs)-1]
			if last.RequestActivity != enums.ActivityReturnSubmitted || last.UserID != f.user.ID {
				t.Errorf("log terakhir = %s oleh %d; want RETURN_SUBMITTED oleh peminjam", last.RequestActivity, last.UserID)
			}
			if len(res.Timeline) != len(logs) {
				t.Errorf("timeline %d baris; want %d (sama dengan log)", len(res.Timeline), len(logs))
			}
		})
	}
}

// Untuk semua kasus gagal: status & foto di database tidak berubah, tidak ada log baru,
// dan tidak ada file yang tertinggal di storage.
func TestSubmitReturn_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		status   string
		setup    func(t *testing.T, f *returnFixture) (id, userID uint, req dto.SubmitReturnRequest)
		wantCode string
		wantKind errx.Kind
	}{
		{name: "tanpa foto", status: enums.BorrowingApproved,
			setup: func(t *testing.T, f *returnFixture) (uint, uint, dto.SubmitReturnRequest) {
				return f.borrowing.ID, f.user.ID, dto.SubmitReturnRequest{}
			}, wantCode: "INVALID_RETURN_PHOTO", wantKind: errx.KindInvalid},
		{name: "file kosong", status: enums.BorrowingApproved,
			setup: func(t *testing.T, f *returnFixture) (uint, uint, dto.SubmitReturnRequest) {
				return f.borrowing.ID, f.user.ID, dto.SubmitReturnRequest{Photo: photoFile(t, "foto.png", nil)}
			}, wantCode: "INVALID_RETURN_PHOTO", wantKind: errx.KindInvalid},
		{name: "file teks berganti nama .jpg", status: enums.BorrowingApproved,
			setup: func(t *testing.T, f *returnFixture) (uint, uint, dto.SubmitReturnRequest) {
				return f.borrowing.ID, f.user.ID, dto.SubmitReturnRequest{Photo: photoFile(t, "foto.jpg", []byte("bukan gambar"))}
			}, wantCode: "INVALID_RETURN_PHOTO", wantKind: errx.KindInvalid},
		{name: "lebih dari 5 MB", status: enums.BorrowingApproved,
			setup: func(t *testing.T, f *returnFixture) (uint, uint, dto.SubmitReturnRequest) {
				big := append(slices.Clone(pngContent), make([]byte, 5<<20)...)
				return f.borrowing.ID, f.user.ID, dto.SubmitReturnRequest{Photo: photoFile(t, "foto.png", big)}
			}, wantCode: "INVALID_RETURN_PHOTO", wantKind: errx.KindInvalid},
		{name: "pengajuan orang lain", status: enums.BorrowingApproved,
			setup: func(t *testing.T, f *returnFixture) (uint, uint, dto.SubmitReturnRequest) {
				other := f.db.addUser("Budi", enums.RoleUser)
				return f.borrowing.ID, other.ID, dto.SubmitReturnRequest{Photo: photoFile(t, "foto.png", pngContent)}
			}, wantCode: "BORROWING_NOT_FOUND", wantKind: errx.KindNotFound},
		{name: "pengajuan tidak ada", status: enums.BorrowingApproved,
			setup: func(t *testing.T, f *returnFixture) (uint, uint, dto.SubmitReturnRequest) {
				return 999, f.user.ID, dto.SubmitReturnRequest{Photo: photoFile(t, "foto.png", pngContent)}
			}, wantCode: "BORROWING_NOT_FOUND", wantKind: errx.KindNotFound},
		{name: "masih PENDING", status: enums.BorrowingPending,
			setup: validReturnRequest, wantCode: "BORROWING_NOT_RETURNABLE", wantKind: errx.KindConflict},
		{name: "ditolak (REJECTED)", status: enums.BorrowingRejected,
			setup: validReturnRequest, wantCode: "BORROWING_NOT_RETURNABLE", wantKind: errx.KindConflict},
		{name: "sudah menunggu persetujuan", status: enums.BorrowingReturnPending,
			setup: validReturnRequest, wantCode: "BORROWING_NOT_RETURNABLE", wantKind: errx.KindConflict},
		{name: "sudah dikembalikan", status: enums.BorrowingReturned,
			setup: validReturnRequest, wantCode: "BORROWING_NOT_RETURNABLE", wantKind: errx.KindConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newReturnFixture(t, tt.status)
			id, userID, req := tt.setup(t, f)
			logsBefore := len(f.db.logs)

			_, err := f.svc.SubmitReturn(t.Context(), id, userID, req)
			assertDomainError(t, err, tt.wantCode, tt.wantKind)
			if s := f.stored(); s.Status != tt.status || s.ReturnPhoto != nil {
				t.Errorf("status %q foto %v; want tetap %q tanpa foto", s.Status, s.ReturnPhoto, tt.status)
			}
			if len(f.db.logs) != logsBefore || len(f.storage.files) != 0 {
				t.Errorf("log %d→%d, file tersimpan %d; want tidak ada yang tersimpan", logsBefore, len(f.db.logs), len(f.storage.files))
			}
		})
	}
}

func validReturnRequest(t *testing.T, f *returnFixture) (uint, uint, dto.SubmitReturnRequest) {
	return f.borrowing.ID, f.user.ID, dto.SubmitReturnRequest{Photo: photoFile(t, "foto.png", pngContent)}
}

// Dua unggahan bersamaan: unggahan lain sudah mengubah status setelah cek cepat. Status dicek ulang setelah
// baris dikunci, transaksi dibatalkan, dan file yang terlanjur disimpan dihapus lagi.
func TestSubmitReturn_StatusChangedAfterCheck_DeletesSavedFile(t *testing.T) {
	t.Parallel()
	f := newReturnFixture(t, enums.BorrowingApproved)
	f.db.beforeLockBorrowing = func() { f.db.borrowings[f.borrowing.ID].Status = enums.BorrowingReturnPending }

	_, err := f.svc.SubmitReturn(t.Context(), f.borrowing.ID, f.user.ID, dto.SubmitReturnRequest{Photo: photoFile(t, "foto.png", pngContent)})
	assertDomainError(t, err, "BORROWING_NOT_RETURNABLE", errx.KindConflict)
	if len(f.storage.files) != 0 || len(f.storage.deleted) != 1 {
		t.Errorf("file tersisa %d, dihapus %v; want file yang baru disimpan ikut dihapus", len(f.storage.files), f.storage.deleted)
	}
	if f.stored().ReturnPhoto != nil {
		t.Errorf("foto tersimpan di database; want tidak (transaksi dibatalkan)")
	}
}

func TestSubmitReturn_StorageFails_NothingSaved(t *testing.T) {
	t.Parallel()
	f := newReturnFixture(t, enums.BorrowingApproved)
	f.storage.saveErr = errors.New("disk penuh")

	_, err := f.svc.SubmitReturn(t.Context(), f.borrowing.ID, f.user.ID, dto.SubmitReturnRequest{Photo: photoFile(t, "foto.png", pngContent)})
	if err == nil {
		t.Fatal("error = nil; want error penyimpanan")
	}
	if _, ok := errx.As(err); ok {
		t.Errorf("error = DomainError; want error biasa (500, bukan kesalahan user)")
	}
	if s := f.stored(); s.Status != enums.BorrowingApproved || s.ReturnPhoto != nil {
		t.Errorf("status %q foto %v; want tetap APPROVED tanpa foto", s.Status, s.ReturnPhoto)
	}
}

// ---------- admin menyetujui pengembalian ----------

func TestApproveReturn_RestocksItems(t *testing.T) {
	t.Parallel()
	f := newReturnFixture(t, enums.BorrowingReturnPending)
	f.withPhoto("returns/bukti.png")

	res, err := f.svc.ApproveReturn(t.Context(), f.borrowing.ID, f.admin.ID)
	mustNoErr(t, err)

	if p, k := f.stock(); p != 5 || k != 3 {
		t.Errorf("stok = %d, %d; want kembali penuh 5, 3", p, k)
	}
	stored := f.stored()
	if stored.Status != enums.BorrowingReturned || stored.ReturnPhoto == nil || f.storage.files["returns/bukti.png"] == nil {
		t.Errorf("status %q foto %v; want RETURNED dan foto tetap disimpan sebagai bukti", stored.Status, stored.ReturnPhoto)
	}
	if res.Status != enums.BorrowingReturned || res.ReturnPhotoURL == nil {
		t.Errorf("respons status %q url %v; want RETURNED dengan url foto", res.Status, res.ReturnPhotoURL)
	}
	logs := f.db.logsFor(f.borrowing.ID)
	if last := logs[len(logs)-1]; last.RequestActivity != enums.ActivityReturnApproved || last.UserID != f.admin.ID {
		t.Errorf("log terakhir = %s oleh %d; want RETURN_APPROVED oleh admin", last.RequestActivity, last.UserID)
	}
	// Urutan kunci sama dengan ACC peminjaman (barang dulu, baru pengajuan), dalam satu transaksi.
	if want := []string{"lock items", "lock borrowing"}; !slices.Equal(f.db.calls, want) || f.tx.started != 1 {
		t.Errorf("urutan kunci %v (transaksi %d); want %v dalam 1 transaksi", f.db.calls, f.tx.started, want)
	}
}

func TestApproveReturn_Errors_NothingChanged(t *testing.T) {
	t.Parallel()
	for _, status := range []string{enums.BorrowingPending, enums.BorrowingApproved, enums.BorrowingRejected, enums.BorrowingReturnRejected, enums.BorrowingReturned} {
		t.Run("status "+status, func(t *testing.T) {
			t.Parallel()
			f := newReturnFixture(t, status)
			_, err := f.svc.ApproveReturn(t.Context(), f.borrowing.ID, f.admin.ID)
			assertDomainError(t, err, "RETURN_NOT_PENDING", errx.KindConflict)
			assertReturnUnchanged(t, f, status)
		})
	}

	t.Run("pengajuan tidak ada", func(t *testing.T) {
		t.Parallel()
		f := newReturnFixture(t, enums.BorrowingReturnPending)
		_, err := f.svc.ApproveReturn(t.Context(), 999, f.admin.ID)
		assertDomainError(t, err, "BORROWING_NOT_FOUND", errx.KindNotFound)
	})

	// Admin lain sudah memproses di antara cek cepat dan penguncian: status dicek ulang setelah terkunci.
	t.Run("sudah diproses admin lain", func(t *testing.T) {
		t.Parallel()
		f := newReturnFixture(t, enums.BorrowingReturnPending)
		f.db.afterLockItems = func() { f.db.borrowings[f.borrowing.ID].Status = enums.BorrowingReturned }
		_, err := f.svc.ApproveReturn(t.Context(), f.borrowing.ID, f.admin.ID)
		assertDomainError(t, err, "RETURN_NOT_PENDING", errx.KindConflict)
		if p, k := f.stock(); p != 3 || k != 2 {
			t.Errorf("stok = %d, %d; want tetap 3, 2 (stok tidak ditambah dua kali)", p, k)
		}
	})

	// Stok tersedia sudah penuh padahal barang tercatat dipinjam: data tidak wajar → 500 dan semua dibatalkan.
	t.Run("stok tidak wajar", func(t *testing.T) {
		t.Parallel()
		f := newReturnFixture(t, enums.BorrowingReturnPending)
		f.db.items[f.kabel.ID].AvailableQuantity = 3 // kabel penuh, tapi pengajuan ini meminjam 1
		_, err := f.svc.ApproveReturn(t.Context(), f.borrowing.ID, f.admin.ID)
		if _, ok := errx.As(err); err == nil || ok {
			t.Fatalf("error = %v; want error biasa (500)", err)
		}
		if p, _ := f.stock(); p != 3 || f.stored().Status != enums.BorrowingReturnPending {
			t.Errorf("stok proyektor %d status %q; want tetap 3 dan RETURN_PENDING (rollback)", p, f.stored().Status)
		}
	})
}

// ---------- admin menolak pengembalian ----------

func TestRejectReturn_DeletesPhotoAndAllowsResubmit(t *testing.T) {
	t.Parallel()
	f := newReturnFixture(t, enums.BorrowingReturnPending)
	f.withPhoto("returns/buram.png")

	res, err := f.svc.RejectReturn(t.Context(), f.borrowing.ID, f.admin.ID)
	mustNoErr(t, err)

	if s := f.stored(); s.Status != enums.BorrowingReturnRejected || s.ReturnPhoto != nil {
		t.Errorf("status %q foto %v; want RETURN_REJECTED tanpa foto", s.Status, s.ReturnPhoto)
	}
	if _, ok := f.storage.files["returns/buram.png"]; ok {
		t.Error("file foto lama masih ada; want dihapus")
	}
	if res.ReturnPhotoURL != nil {
		t.Errorf("respons url = %v; want null", *res.ReturnPhotoURL)
	}
	if p, k := f.stock(); p != 3 || k != 2 {
		t.Errorf("stok = %d, %d; want tetap 3, 2 (barang belum kembali)", p, k)
	}

	// User mengunggah foto baru → kembali menunggu persetujuan.
	_, err = f.svc.SubmitReturn(t.Context(), f.borrowing.ID, f.user.ID, dto.SubmitReturnRequest{Photo: photoFile(t, "baru.jpg", jpegContent)})
	mustNoErr(t, err)
	if s := f.stored(); s.Status != enums.BorrowingReturnPending || s.ReturnPhoto == nil || *s.ReturnPhoto == "returns/buram.png" {
		t.Errorf("status %q foto %v; want RETURN_PENDING dengan foto baru", s.Status, s.ReturnPhoto)
	}

	want := []string{enums.ActivitySubmitted, enums.ActivityReturnRejected, enums.ActivityReturnSubmitted}
	if got := activities(f.db.logsFor(f.borrowing.ID)); !slices.Equal(got, want) {
		t.Errorf("timeline = %v; want %v", got, want)
	}
}

// Gagal menghapus file tidak menggagalkan penolakan: database sudah benar, file yatim tidak berbahaya.
func TestRejectReturn_DeleteFileFails_StillRejected(t *testing.T) {
	t.Parallel()
	f := newReturnFixture(t, enums.BorrowingReturnPending)
	f.withPhoto("returns/buram.png")
	f.storage.deleteErr = errors.New("izin ditolak")

	_, err := f.svc.RejectReturn(t.Context(), f.borrowing.ID, f.admin.ID)
	mustNoErr(t, err)
	if f.stored().Status != enums.BorrowingReturnRejected {
		t.Errorf("status = %q; want RETURN_REJECTED", f.stored().Status)
	}
}

func TestRejectReturn_Errors_PhotoKept(t *testing.T) {
	t.Parallel()
	for _, status := range []string{enums.BorrowingPending, enums.BorrowingApproved, enums.BorrowingReturnRejected, enums.BorrowingReturned} {
		t.Run("status "+status, func(t *testing.T) {
			t.Parallel()
			f := newReturnFixture(t, status)
			f.withPhoto("returns/bukti.png")
			_, err := f.svc.RejectReturn(t.Context(), f.borrowing.ID, f.admin.ID)
			assertDomainError(t, err, "RETURN_NOT_PENDING", errx.KindConflict)
			if len(f.storage.deleted) != 0 || f.stored().ReturnPhoto == nil {
				t.Errorf("foto dihapus %v; want foto tetap ada", f.storage.deleted)
			}
		})
	}
}

// assertReturnUnchanged memastikan status, stok, dan log tidak berubah (hanya log SUBMITTED dari fixture).
func assertReturnUnchanged(t *testing.T, f *returnFixture, wantStatus string) {
	t.Helper()
	if s := f.stored().Status; s != wantStatus {
		t.Errorf("status = %q; want tetap %q", s, wantStatus)
	}
	if p, k := f.stock(); p != 3 || k != 2 {
		t.Errorf("stok = %d, %d; want tetap 3, 2", p, k)
	}
	if n := len(f.db.logsFor(f.borrowing.ID)); n != 1 {
		t.Errorf("jumlah log = %d; want 1 (tidak ada log baru)", n)
	}
}
