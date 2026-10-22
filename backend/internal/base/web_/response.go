package web_

import (
	"handfree-work/octo-backup/internal/base/error_"

	"github.com/gofiber/fiber/v3"
)

// ListResponse 表示不分页列表响应的数据结构。
type ListResponse[T any] struct {
	Records []T `json:"records"`
}

// PageResponse 表示统一分页响应的数据结构。
type PageResponse[T any] struct {
	Offset  int64 `json:"offset"`
	Limit   int64 `json:"limit"`
	Records []T   `json:"records"`
	Total   int64 `json:"total"`
}

// Success returns the standard business response envelope.
func Success(c fiber.Ctx, data any) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"code": 0, "message": "成功", "data": data})
}

// Error returns a generic business error while keeping the HTTP status at 200.
func Error(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"code": 1, "message": message, "data": fiber.Map{}})
}

// BusinessError returns a coded business error while keeping the HTTP status at 200.
func BusinessError(c fiber.Ctx, code int, err error) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"code": code, "message": err.Error(), "data": fiber.Map{}})
}

// ErrorHandler 将业务 handler 返回的错误统一转换为标准响应。
func ErrorHandler(c fiber.Ctx, err error) error {
	apiErr := error_.NewApiError(err)
	return BusinessError(c, apiErr.Code, apiErr)
}

// HandleJSON 统一处理 JSON 请求体、业务调用和成功响应。
// 业务函数只需返回数据或错误，错误交给 Fiber 的 ErrorHandler 统一转换。
func HandleJSON[T any, R any](action func(fiber.Ctx, *T) (R, error)) fiber.Handler {
	return func(c fiber.Ctx) error {
		var input T
		if err := c.Bind().Body(&input); err != nil {
			return error_.NewRequestError("请求体格式错误")
		}
		data, err := action(c, &input)
		if err != nil {
			return err
		}
		return Success(c, data)
	}
}

// Handle 统一处理无请求体业务、成功响应和错误返回。
func Handle[R any](action func(fiber.Ctx) (R, error)) fiber.Handler {
	return func(c fiber.Ctx) error {
		data, err := action(c)
		if err != nil {
			return err
		}
		return Success(c, data)
	}
}
