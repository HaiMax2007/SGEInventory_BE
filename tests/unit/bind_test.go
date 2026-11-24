package unit

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/errx"
	"SGEInventory_BE/internal/middlewares"
)

// Tes ini memakai c.Bind().Body sungguhan dari Fiber (sama seperti di handler), supaya terbukti
// errx.InvalidBody tetap mengenali error JSON walaupun dibungkus fiber.BindError.

// bindTarget meniru bentuk DTO request: teks, angka, id (uint), pointer, dan daftar.
type bindTarget struct {
	Name          string `json:"name"`
	LembagaID     uint   `json:"lembagaId"`
	TotalQuantity int    `json:"totalQuantity"`
	Stock         *int   `json:"stock"`
	Active        bool   `json:"active"`
	Items         []struct {
		Quantity int `json:"quantity"`
	} `json:"items"`
}

func TestInvalidBody(t *testing.T) {
	t.Parallel()
	app := fiber.New(fiber.Config{ErrorHandler: middlewares.ErrorHandler})
	app.Post("/bind", func(c fiber.Ctx) error {
		var req bindTarget
		if err := c.Bind().Body(&req); err != nil {
			return errx.InvalidBody(err)
		}
		return c.SendString("ok")
	})

	const invalidBody = "Format data tidak valid, kirim body JSON yang benar"
	tests := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
		wantMessage string
	}{
		{name: "valid", body: `{"name":"Proyektor","totalQuantity":5}`, wantStatus: 200},
		{name: "angka dikirim teks", body: `{"totalQuantity":"lima"}`, wantStatus: 400,
			wantMessage: "Field totalQuantity harus berupa angka bulat"},
		{name: "angka desimal", body: `{"totalQuantity":1.5}`, wantStatus: 400,
			wantMessage: "Field totalQuantity harus berupa angka bulat"},
		{name: "pointer angka dikirim teks", body: `{"stock":"lima"}`, wantStatus: 400,
			wantMessage: "Field stock harus berupa angka bulat"},
		{name: "id negatif", body: `{"lembagaId":-1}`, wantStatus: 400,
			wantMessage: "Field lembagaId harus berupa angka bulat positif"},
		{name: "teks dikirim angka", body: `{"name":5}`, wantStatus: 400,
			wantMessage: "Field name harus berupa teks"},
		{name: "boolean dikirim teks", body: `{"active":"ya"}`, wantStatus: 400,
			wantMessage: "Field active harus berupa true/false"},
		{name: "daftar dikirim teks", body: `{"items":"proyektor"}`, wantStatus: 400,
			wantMessage: "Field items harus berupa daftar (array)"},
		{name: "field di dalam daftar", body: `{"items":[{"quantity":"dua"}]}`, wantStatus: 400,
			wantMessage: "Field items.0.quantity harus berupa angka bulat"}, // 0 = barang pertama
		{name: "JSON rusak", body: `{"name":`, wantStatus: 400, wantMessage: invalidBody},
		{name: "bukan objek", body: `[1,2]`, wantStatus: 400, wantMessage: invalidBody},
		{name: "tanpa Content-Type", contentType: "-", body: `{"name":"x"}`, wantStatus: 400, wantMessage: invalidBody},
		{name: "Content-Type teks", contentType: "text/plain", body: `{"name":"x"}`, wantStatus: 400, wantMessage: invalidBody},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, "/bind", strings.NewReader(tt.body))
			switch tt.contentType {
			case "":
				req.Header.Set("Content-Type", "application/json")
			case "-": // tidak dikirim
			default:
				req.Header.Set("Content-Type", tt.contentType)
			}
			resp, err := app.Test(req)
			mustNoErr(t, err)
			defer resp.Body.Close()
			raw, err := io.ReadAll(resp.Body)
			mustNoErr(t, err)

			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("POST %s = %d %s; want %d", tt.body, resp.StatusCode, raw, tt.wantStatus)
			}
			if tt.wantMessage == "" {
				return
			}
			var res dto.APIResponse
			mustNoErr(t, json.Unmarshal(raw, &res))
			if res.Message != tt.wantMessage {
				t.Errorf("POST %s message = %q; want %q", tt.body, res.Message, tt.wantMessage)
			}
			// Pesan asli Go (nama struct, "json: cannot unmarshal") tidak boleh bocor.
			if strings.Contains(string(raw), "bindTarget") || strings.Contains(string(raw), "unmarshal") {
				t.Errorf("POST %s membocorkan detail Go: %s", tt.body, raw)
			}
		})
	}
}
