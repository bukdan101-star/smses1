package middleware

import (
	"github.com/gofiber/fiber/v2"
)

// StaffOrAdminOnly - Bisa diakses staff atau admin
func StaffOrAdminOnly(c *fiber.Ctx) error {
	user := c.Locals("user")
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	claims, ok := user.(*map[string]interface{})
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid user claims",
		})
	}

	role, ok := (*claims)["role"].(string)
	if !ok || (role != "admin" && role != "staff") {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Staff or admin access required",
		})
	}

	return c.Next()
}
