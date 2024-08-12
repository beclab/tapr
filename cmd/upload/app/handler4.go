package app

import (
	"bytetrade.io/web3os/tapr/pkg/upload/fileutils"
	"bytetrade.io/web3os/tapr/pkg/upload/models"
	"bytetrade.io/web3os/tapr/pkg/upload/uid"
	"bytetrade.io/web3os/tapr/pkg/utils"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
	"strings"
)

// UploadLink 处理上传链接的 GET 请求
func (a *appController) UploadLink(c *fiber.Ctx) error {
	// 从查询参数中获取 path
	path := c.Query("path", "")
	if path == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "missing path query parameter", nil))
	}

	// 检查上传目录是否存在，如果不存在则创建
	if !utils.PathExists(fileutils.UploadsDir) {
		if err := os.MkdirAll(fileutils.UploadsDir, os.ModePerm); err != nil {
			klog.Warning("err:", err)
			// 如果创建目录失败，返回内部服务器错误
			return c.Status(fiber.StatusInternalServerError).JSON(
				models.NewResponse(1, "failed to create folder", nil))
		}
	}
	klog.Infof("c:%+v", c) // 记录当前请求的上下文信息

	// 检查文件的目录路径是否存在，不存在则创建
	if !utils.CheckDirExist(path) {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			klog.Warning("err:", err)
			// 如果创建目录失败，返回内部服务器错误
			return c.Status(fiber.StatusInternalServerError).JSON(
				models.NewResponse(1, "failed to create folder", nil))
		}
	}

	// 生成唯一的上传ID
	uploadID := uid.MakeUid(path)

	// 拼接响应字符串
	uploadLink := fmt.Sprintf("/upload/upload_link/%s", uploadID)

	// 返回生成的链接
	return c.SendString(uploadLink)
}

func (a *appController) UploadedBytes(c *fiber.Ctx) error {
	parentDir := c.Query("parent_dir", "")
	if parentDir == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "missing parent_dir query parameter", nil))
	}

	if !utils.CheckDirExist(parentDir) {
		klog.Warningf("Storage path %s is not exist or is not a dir", parentDir)
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "Storage path is not exist or is not a dir", nil))
	}

	fileName := c.Query("file_name", "")
	if fileName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "file_relative_path invalid", nil))
	}

	responseData := make(map[string]interface{})
	responseData["uploadedBytes"] = 0

	if !utils.PathExists(fileutils.UploadsDir) {
		return c.JSON(responseData)
	}
	klog.Infof("c:%+v", c)

	fullPath := filepath.Join(parentDir, fileName)

	dirPath := filepath.Dir(fullPath)

	if !utils.CheckDirExist(dirPath) {
		return c.JSON(responseData)
	}

	if strings.HasSuffix(fileName, "/") {
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, fmt.Sprintf("full path %s is a dir", fullPath), nil))
	}

	//Generate unique Upload-ID
	uploadID := uid.MakeUid(fullPath)
	exist, info := a.server.fileInfoMgr.ExistFileInfo(uploadID)
	fileExist, fileLen := a.server.fileInfoMgr.CheckTempFile(uploadID)
	if exist {
		if fileExist {
			if info.Offset != fileLen {
				info.Offset = fileLen
				a.server.fileInfoMgr.UpdateInfo(uploadID, info)
			}
			klog.Infof("uploadID:%s, info.Offset:%d", uploadID, info.Offset)
			responseData["uploadedBytes"] = info.Offset
		} else if info.Offset == 0 {
			klog.Warningf("uploadID:%s, info.Offset:%d", uploadID, info.Offset)
		} else {
			a.server.fileInfoMgr.DelFileInfo(uploadID)
		}
	}
	return c.JSON(responseData)
}
