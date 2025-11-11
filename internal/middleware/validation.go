package middleware

import (
	"event-management-backend/internal/utils"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

var validate = validator.New()

func ValidateBody(dest interface{}) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := c.BodyParser(dest); err != nil {
			return utils.Error(c, "Invalid request body", fiber.StatusBadRequest)
		}

		if err := validate.Struct(dest); err != nil {
			validationErrors := err.(validator.ValidationErrors)
			firstError := validationErrors[0]

			var errorMessage string
			switch firstError.Tag() {
			case "required":
				errorMessage = firstError.Field() + " is required"
			case "email":
				errorMessage = "Invalid email format"
			case "min":
				errorMessage = firstError.Field() + " is too short"
			case "max":
				errorMessage = firstError.Field() + " is too long"
			case "uuid":
				errorMessage = "Invalid UUID format"
			case "gtfield":
				errorMessage = firstError.Field() + " must be greater than " + firstError.Param()
			default:
				errorMessage = "Validation failed for " + firstError.Field()
			}

			return utils.Error(c, errorMessage, fiber.StatusBadRequest)
		}

		c.Locals("validatedBody", dest)
		return c.Next()
	}
}
