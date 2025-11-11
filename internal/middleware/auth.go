package middleware

import (
	"event-management-backend/internal/config"
	"event-management-backend/internal/utils"

	"github.com/gofiber/fiber/v2"
	jwtware "github.com/gofiber/jwt/v3"
	"github.com/golang-jwt/jwt/v4"
)

func JWTMiddleware(cfg *config.Config) fiber.Handler {
	return jwtware.New(jwtware.Config{
		SigningKey:   []byte(cfg.JWTSecret),
		ContextKey:   "user",
		ErrorHandler: jwtError,
		SuccessHandler: func(c *fiber.Ctx) error {
			user := c.Locals("user").(*jwt.Token)
			claims := user.Claims.(jwt.MapClaims)
			c.Locals("user_id", claims["user_id"])
			c.Locals("user_role", claims["role"])
			return c.Next()
		},
	})
}

func jwtError(c *fiber.Ctx, err error) error {
	return utils.Error(c, "Unauthorized", fiber.StatusUnauthorized)
}

func AdminOnly(c *fiber.Ctx) error {
	userRole, ok := c.Locals("user_role").(string)
	if !ok || userRole != "admin" {
		return utils.Error(c, "Admin access required", fiber.StatusForbidden)
	}
	return c.Next()
}

func OrganizerOrAdmin(c *fiber.Ctx) error {
	userRole, ok := c.Locals("user_role").(string)
	if !ok || (userRole != "admin" && userRole != "organizer") {
		return utils.Error(c, "Access denied", fiber.StatusForbidden)
	}
	return c.Next()
}

func StaffOrAbove(c *fiber.Ctx) error {
	userRole, ok := c.Locals("user_role").(string)
	if !ok {
		return utils.Error(c, "Access denied", fiber.StatusForbidden)
	}

	allowedRoles := map[string]bool{
		"admin":     true,
		"organizer": true,
		"staff":     true,
	}

	if !allowedRoles[userRole] {
		return utils.Error(c, "Access denied", fiber.StatusForbidden)
	}
	return c.Next()
}

func GetUserIDFromContext(c *fiber.Ctx) (string, error) {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return "", fiber.NewError(fiber.StatusUnauthorized, "User not authenticated")
	}
	return userID, nil
}
