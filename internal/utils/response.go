package utils

import "github.com/gofiber/fiber/v2"

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

type Meta struct {
	Page      int   `json:"page,omitempty"`
	PageSize  int   `json:"page_size,omitempty"`
	Total     int64 `json:"total,omitempty"`
	TotalPage int   `json:"total_page,omitempty"`
}

func Success(c *fiber.Ctx, data interface{}, message string, statusCode ...int) error {
	code := fiber.StatusOK
	if len(statusCode) > 0 {
		code = statusCode[0]
	}

	resp := Response{
		Success: true,
		Message: message,
		Data:    data,
	}

	return c.Status(code).JSON(resp)
}

func SuccessWithMeta(c *fiber.Ctx, data interface{}, meta *Meta, message string) error {
	resp := Response{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

func Error(c *fiber.Ctx, message string, statusCode ...int) error {
	code := fiber.StatusBadRequest
	if len(statusCode) > 0 {
		code = statusCode[0]
	}

	resp := Response{
		Success: false,
		Error:   message,
	}

	return c.Status(code).JSON(resp)
}
