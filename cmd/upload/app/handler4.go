package app

import (
	"bytetrade.io/web3os/tapr/pkg/upload/models"
	"bytetrade.io/web3os/tapr/pkg/upload/uid"
	"fmt"
	"github.com/gofiber/fiber/v2"
)

// UploadLink 处理上传链接的 GET 请求
func (a *appController) UploadLink(c *fiber.Ctx) error {
	// 从查询参数中获取 path
	fullPath := c.Query("path", "")
	if fullPath == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "missing path query parameter", nil))
	}

	// 假设 uid.MakeUid 是用于生成唯一 ID 的函数
	// 注意：这里直接使用 fullPath 作为生成 UploadID 的基础，实际中可能需要更复杂的逻辑
	uploadID := uid.MakeUid(fullPath)

	// 拼接响应字符串
	uploadLink := fmt.Sprintf("/upload/upload_link/%s", uploadID)

	// 返回生成的链接
	return c.SendString(uploadLink)
}
