package unit

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"

	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
	"SGEInventory_BE/internal/middlewares"
)

const testSecret = "rahasia-untuk-unit-test-minimal-32-karakter"

// signToken membuat JWT seperti AuthService.Login. exp nol = token tanpa waktu kedaluwarsa.
func signToken(t *testing.T, secret string, userID uint, role string, exp time.Time) string {
	t.Helper()
	claims := dto.JWTClaims{UserID: userID, Role: role}
	if !exp.IsZero() {
		claims.ExpiresAt = jwt.NewNumericDate(exp)
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	mustNoErr(t, err)
	return token
}

// newMiddlewareApp membuat app Fiber kecil: tiap route membalas "userID:role" dari c.Locals.
func newMiddlewareApp() *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: middlewares.ErrorHandler})
	whoAmI := func(c fiber.Ctx) error {
		return c.SendString(fmt.Sprintf("%d:%s", fiber.Locals[uint](c, "userID"), fiber.Locals[string](c, "role")))
	}
	protected := middlewares.Protected(testSecret)
	app.Get("/login", protected, whoAmI)
	app.Get("/admin", protected, middlewares.AdminOnly, whoAmI)
	app.Get("/user", protected, middlewares.UserOnly, whoAmI)
	return app
}

// doRequest mengirim request palsu ke app (tanpa membuka port) dan mengembalikan status + body.
func doRequest(t *testing.T, app *fiber.App, path, authHeader string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	resp, err := app.Test(req)
	mustNoErr(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	mustNoErr(t, err)
	return resp.StatusCode, string(body)
}

func TestProtected(t *testing.T) {
	t.Parallel()
	app := newMiddlewareApp()
	future := time.Now().Add(time.Hour)

	noneToken, err := jwt.NewWithClaims(jwt.SigningMethodNone,
		dto.JWTClaims{UserID: 1, Role: enums.RoleAdmin, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(future)}},
	).SignedString(jwt.UnsafeAllowNoneSignatureType)
	mustNoErr(t, err)

	tests := []struct {
		name       string
		header     string
		wantStatus int
		wantBody   string
	}{
		{name: "tanpa header", header: "", wantStatus: 401, wantBody: "Token tidak ada"},
		{name: "bukan Bearer", header: "Basic abc", wantStatus: 401, wantBody: "Token tidak ada"},
		{name: "Bearer kosong", header: "Bearer ", wantStatus: 401, wantBody: "Token tidak ada"},
		{name: "token acak", header: "Bearer abc.def.ghi", wantStatus: 401, wantBody: "Token tidak valid"},
		{name: "secret berbeda", header: "Bearer " + signToken(t, "secret-lain", 1, enums.RoleUser, future), wantStatus: 401, wantBody: "Token tidak valid"},
		{name: "sudah kedaluwarsa", header: "Bearer " + signToken(t, testSecret, 1, enums.RoleUser, time.Now().Add(-time.Minute)), wantStatus: 401, wantBody: "Token tidak valid"},
		{name: "tanpa waktu kedaluwarsa", header: "Bearer " + signToken(t, testSecret, 1, enums.RoleUser, time.Time{}), wantStatus: 401, wantBody: "Token tidak valid"},
		{name: "algoritma none ditolak", header: "Bearer " + noneToken, wantStatus: 401, wantBody: "Token tidak valid"},
		{name: "valid: id & role tersimpan di Locals", header: "Bearer " + signToken(t, testSecret, 7, enums.RoleUser, future), wantStatus: 200, wantBody: "7:USER"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			status, body := doRequest(t, app, "/login", tt.header)
			if status != tt.wantStatus || !strings.Contains(body, tt.wantBody) {
				t.Errorf("GET /login dengan header %q = %d %s; want %d berisi %q", tt.header, status, body, tt.wantStatus, tt.wantBody)
			}
		})
	}
}

func TestAdminOnlyAndUserOnly(t *testing.T) {
	t.Parallel()
	app := newMiddlewareApp()
	future := time.Now().Add(time.Hour)
	adminToken := "Bearer " + signToken(t, testSecret, 1, enums.RoleAdmin, future)
	userToken := "Bearer " + signToken(t, testSecret, 2, enums.RoleUser, future)

	tests := []struct {
		path, header string
		wantStatus   int
	}{
		{path: "/admin", header: adminToken, wantStatus: 200},
		{path: "/admin", header: userToken, wantStatus: 403},
		{path: "/user", header: userToken, wantStatus: 200},
		{path: "/user", header: adminToken, wantStatus: 403},
		{path: "/admin", header: "", wantStatus: 401}, // Protected dijalankan lebih dulu
	}
	for _, tt := range tests {
		status, body := doRequest(t, app, tt.path, tt.header)
		if status != tt.wantStatus {
			t.Errorf("GET %s (token %q...) = %d %s; want %d", tt.path, short(tt.header), status, body, tt.wantStatus)
		}
	}
}

func TestErrorHandler(t *testing.T) {
	t.Parallel()
	app := fiber.New(fiber.Config{ErrorHandler: middlewares.ErrorHandler})
	routes := map[string]error{
		"/invalid":    errx.ErrInvalidID,
		"/validation": errx.Validation(errors.New("detail validasi")),
		"/auth":       errx.ErrTokenMissing,
		"/forbidden":  errx.ErrAdminOnly,
		"/notfound":   errx.ErrItemNotFound,
		"/conflict":   fmt.Errorf("repo: %w", errx.ErrEmailTaken), // terbungkus tetap dikenali
		"/internal":   errors.New("pq: password authentication failed for user db_admin"),
	}
	for path, err := range routes {
		app.Get(path, func(fiber.Ctx) error { return err })
	}

	tests := []struct {
		path        string
		wantStatus  int
		wantMessage string
		wantError   string
	}{
		{path: "/invalid", wantStatus: 400, wantMessage: "ID tidak valid"},
		{path: "/validation", wantStatus: 400, wantMessage: "Data tidak valid", wantError: "detail validasi"},
		{path: "/auth", wantStatus: 401, wantMessage: "Token tidak ada"},
		{path: "/forbidden", wantStatus: 403, wantMessage: "Hanya admin yang boleh mengakses"},
		{path: "/notfound", wantStatus: 404, wantMessage: "Barang tidak ditemukan"},
		{path: "/conflict", wantStatus: 409, wantMessage: "Email sudah terdaftar"},
		// error tak dikenal: detail (mis. info database) TIDAK boleh bocor ke client
		{path: "/internal", wantStatus: 500, wantMessage: "Terjadi kesalahan pada server"},
		{path: "/tidak-ada", wantStatus: 404}, // route tidak terdaftar → *fiber.Error
	}
	for _, tt := range tests {
		status, body := doRequest(t, app, tt.path, "")
		var res dto.APIResponse
		if err := json.Unmarshal([]byte(body), &res); err != nil {
			t.Fatalf("GET %s: body bukan JSON: %s", tt.path, body)
		}
		if status != tt.wantStatus {
			t.Errorf("GET %s status = %d; want %d", tt.path, status, tt.wantStatus)
		}
		if tt.wantMessage != "" && res.Message != tt.wantMessage {
			t.Errorf("GET %s message = %q; want %q", tt.path, res.Message, tt.wantMessage)
		}
		if res.Error != tt.wantError {
			t.Errorf("GET %s error = %q; want %q", tt.path, res.Error, tt.wantError)
		}
	}
}

func short(s string) string { return s[:min(len(s), 15)] }
