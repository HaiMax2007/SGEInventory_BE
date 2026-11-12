package middlewares

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"

	"SGEInventory_BE/models"
)

// Protected memastikan request membawa token yang valid di header:
//
//	Authorization: Bearer <token>
//
// Jika valid, id & role user disimpan di c.Locals agar bisa dibaca controller.
func Protected(c fiber.Ctx) error {
	tokenString, ok := strings.CutPrefix(c.Get("Authorization"), "Bearer ")
	if !ok || tokenString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Token tidak ada"})
	}

	claims := &models.JWTClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims,
		func(token *jwt.Token) (any, error) {
			return []byte(os.Getenv("JWT_SECRET")), nil
		},
		jwt.WithValidMethods([]string{"HS256"}), // tolak token dengan algoritma lain (mis. "none")
		jwt.WithExpirationRequired(),            // token wajib punya waktu kedaluwarsa
	)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Token tidak valid atau sudah kedaluwarsa"})
	}

	c.Locals("userID", claims.UserID)
	c.Locals("role", claims.Role)
	return c.Next()
}

// AdminOnly hanya meloloskan user dengan role ADMIN. Pasang setelah Protected.
func AdminOnly(c fiber.Ctx) error {
	if fiber.Locals[string](c, "role") != models.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Hanya admin yang boleh mengakses"})
	}
	return c.Next()
}
