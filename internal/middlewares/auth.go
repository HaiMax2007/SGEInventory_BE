package middlewares

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"

	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
)

// Protected membuat middleware yang memastikan request membawa token valid di header:
//
//	Authorization: Bearer <token>
//
// Jika valid, id & role user disimpan di c.Locals agar bisa dibaca handler.
// Error dikembalikan (return errx...) lalu diubah menjadi JSON oleh ErrorHandler.
func Protected(jwtSecret string) fiber.Handler {
	return func(c fiber.Ctx) error {
		tokenString, ok := strings.CutPrefix(c.Get("Authorization"), "Bearer ")
		if !ok || tokenString == "" {
			return errx.ErrTokenMissing
		}

		claims := &dto.JWTClaims{}
		_, err := jwt.ParseWithClaims(tokenString, claims,
			func(token *jwt.Token) (any, error) {
				return []byte(jwtSecret), nil
			},
			jwt.WithValidMethods([]string{"HS256"}), // tolak token dengan algoritma lain (mis. "none")
			jwt.WithExpirationRequired(),            // token wajib punya waktu kedaluwarsa
		)
		if err != nil {
			return errx.ErrTokenInvalid
		}

		c.Locals("userID", claims.UserID)
		c.Locals("role", claims.Role)
		return c.Next()
	}
}

// AdminOnly hanya meloloskan user dengan role ADMIN. Pasang setelah Protected.
func AdminOnly(c fiber.Ctx) error {
	if fiber.Locals[string](c, "role") != enums.RoleAdmin {
		return errx.ErrAdminOnly
	}
	return c.Next()
}

// UserOnly hanya meloloskan user dengan role USER (mis. mengajukan peminjaman). Pasang setelah Protected.
func UserOnly(c fiber.Ctx) error {
	if fiber.Locals[string](c, "role") != enums.RoleUser {
		return errx.ErrUserOnly
	}
	return c.Next()
}
