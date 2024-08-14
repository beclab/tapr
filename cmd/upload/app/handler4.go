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
	uploadLink := fmt.Sprintf("/upload/upload-link/%s", uploadID)

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
	//uploadID := uid.MakeUid(fullPath)
	//resumableIdentifier := uid.GenerateUniqueIdentifier(fileName)
	innerIdentifier := uid.MakeUid(fullPath)
	exist, info := a.server.fileInfoMgr.ExistFileInfo(innerIdentifier)
	fileExist, fileLen := a.server.fileInfoMgr.CheckTempFile(innerIdentifier)
	if exist {
		if fileExist {
			if info.Offset != fileLen {
				info.Offset = fileLen
				a.server.fileInfoMgr.UpdateInfo(innerIdentifier, info)
			}
			klog.Infof("innerIdentifier:%s, info.Offset:%d", innerIdentifier, info.Offset)
			responseData["uploadedBytes"] = info.Offset
		} else if info.Offset == 0 {
			klog.Warningf("innerIdentifier:%s, info.Offset:%d", innerIdentifier, info.Offset)
		} else {
			a.server.fileInfoMgr.DelFileInfo(innerIdentifier)
		}
	}
	return c.JSON(responseData)
}

func (a *appController) UploadChunks(c *fiber.Ctx) error {
	responseData := make(map[string]interface{})
	responseData["success"] = true

	uploadID := c.Params("uid")

	if !utils.PathExists(fileutils.UploadsDir) {
		if err := os.MkdirAll(fileutils.UploadsDir, os.ModePerm); err != nil {
			klog.Warningf("uploadID:%s, err:%v", uploadID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(
				models.NewResponse(1, "failed to create folder", nil))
		}
	}

	klog.Infof("uploadID:%s, c:%+v", uploadID, c)

	var resumableInfo models.ResumableInfo
	if err := c.BodyParser(&resumableInfo); err != nil {
		klog.Warningf("uploadID:%s, err:%v", uploadID, err)
		//todo check info valid
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "param invalid", nil))
	}

	if uploadID != uid.MakeUid(resumableInfo.ParentDir) {
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "invalid upload link", nil))
	}

	var err error
	resumableInfo.File, err = c.FormFile("file")
	if err != nil || resumableInfo.File == nil {
		klog.Warningf("uploadID:%s, Failed to parse file: %v\n", uploadID, err)
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "param invalid", nil))
	}

	klog.Infof("uploadID:%s, patchInfo:%+v", uploadID, resumableInfo)

	// Get file information based on upload ID
	fullPath := filepath.Join(resumableInfo.ParentDir, resumableInfo.ResumableRelativePath)
	//resumableIdentifier := resumableInfo.ResumableIdentifier
	innerIdentifier := uid.MakeUid(fullPath)
	exist, info := a.server.fileInfoMgr.ExistFileInfo(innerIdentifier)
	if !exist {
		klog.Warningf("innerIdentifier %s not exist", innerIdentifier)
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "Invalid innerIdentifier", nil))
	}
	klog.Infof("innerIdentifier:%s, info:%+v", innerIdentifier, info)
	if innerIdentifier != info.ID {
		klog.Warningf("innerIdentifier:%s diff from info:%+v", innerIdentifier, info)
	}

	if resumableInfo.ResumableChunkNumber == 1 {
		//clear temp file and reset info
		fileutils.RemoveTempFileAndInfoFile(innerIdentifier)
		if info.Offset != 0 {
			info.Offset = 0
			a.server.fileInfoMgr.UpdateInfo(innerIdentifier, info)
		}

		//do creation when the first chunk
		if !utils.CheckDirExist(resumableInfo.ParentDir) {
			klog.Warningf("Parent dir %s is not exist or is not a dir", resumableInfo.ParentDir)
			return c.Status(fiber.StatusBadRequest).JSON(
				models.NewResponse(1, "Parent dir is not exist or is not a dir", nil))
		}

		//fullPath := filepath.Join(resumableInfo.ParentDir, resumableInfo.ResumableRelativePath)

		dirPath := filepath.Dir(fullPath)

		if !utils.CheckDirExist(dirPath) {
			if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
				klog.Warning("err:", err)
				return c.Status(fiber.StatusInternalServerError).JSON(
					models.NewResponse(1, "failed to create folder", nil))
			}
		}

		if resumableInfo.ResumableRelativePath == "" {
			return c.Status(fiber.StatusBadRequest).JSON(
				models.NewResponse(1, "file_relative_path invalid", nil))
		}

		if strings.HasSuffix(resumableInfo.ResumableRelativePath, "/") {
			klog.Warningf("full path %s is a dir", fullPath)
			return c.Status(fiber.StatusBadRequest).JSON(
				models.NewResponse(1, fmt.Sprintf("full path %s is a dir", fullPath), nil))
		}

		// Make support judgment after parsing the file type
		if !a.server.checkType(resumableInfo.ResumableType) {
			klog.Warningf("unsupported filetype:%s", resumableInfo.ResumableType)
			return c.Status(fiber.StatusBadRequest).JSON(
				models.NewResponse(1, "Unsupported file type", nil))
		}

		if !a.server.checkSize(resumableInfo.ResumableTotalSize) {
			klog.Warningf("Unsupported file size uploadSize:%d", resumableInfo.ResumableTotalSize)
			return c.Status(fiber.StatusBadRequest).JSON(
				models.NewResponse(1, "Unsupported file size", nil))
		}

		//Generate unique Upload-ID
		//uploadID := uid.MakeUid(uploadInfo.FullPath)
		oExist, oInfo := a.server.fileInfoMgr.ExistFileInfo(innerIdentifier)
		oFileExist, oFileLen := a.server.fileInfoMgr.CheckTempFile(innerIdentifier)
		if oExist {
			if oFileExist {
				if oInfo.Offset != oFileLen {
					oInfo.Offset = oFileLen
					a.server.fileInfoMgr.UpdateInfo(innerIdentifier, oInfo)
				}
				klog.Infof("innerIdentifier:%s, info.Offset:%d", innerIdentifier, oInfo.Offset)
				//return c.Status(fiber.StatusOK).JSON(
				//	models.NewResponse(0, "success", info))
				return c.JSON(responseData)
			} else if oInfo.Offset == 0 {
				klog.Warningf("innerIdentifier:%s, info.Offset:%d", innerIdentifier, oInfo.Offset)
				//return c.Status(fiber.StatusOK).JSON(
				//	models.NewResponse(0, "success", info))
				return c.JSON(responseData)
			} else {
				a.server.fileInfoMgr.DelFileInfo(innerIdentifier)
			}
		}

		fileInfo := models.FileInfo{
			ID:     innerIdentifier,
			Offset: 0,
			FileMetaData: models.FileMetaData{
				FileRelativePath: resumableInfo.ResumableRelativePath,
				FileType:         resumableInfo.ResumableType,
				FileSize:         resumableInfo.ResumableTotalSize,
				StoragePath:      resumableInfo.ParentDir,
				FullPath:         fullPath,
			},
		}

		if oFileExist {
			fileInfo.Offset = oFileLen
		}

		err = a.server.fileInfoMgr.AddFileInfo(innerIdentifier, fileInfo)
		if err != nil {
			klog.Warningf("innerIdentifier:%s, err:%v", innerIdentifier, err)
			return c.Status(fiber.StatusInternalServerError).JSON(
				models.NewResponse(1, "Error save file info", nil))
		}

		klog.Infof("innerIdentifier:%s, fileInfo:%+v", innerIdentifier, fileInfo)
		//return c.Status(fiber.StatusOK).JSON(
		//	models.NewResponse(0, "success", fileInfo))
		// can't return here
	}

	fileExist, fileLen := a.server.fileInfoMgr.CheckTempFile(innerIdentifier)
	if fileExist {
		klog.Infof("innerIdentifier %s temp file exist, info.Offset:%d, fileLen:%d", uploadID, info.Offset, fileLen)
		if info.Offset != fileLen {
			info.Offset = fileLen
			a.server.fileInfoMgr.UpdateInfo(uploadID, info)
		}
	}

	// Check if file size and offset match
	//if patchInfo.UploadOffset != info.Offset {
	//	klog.Warningf("uploadID %s, patchInfo.UploadOffset:%d diff from info.Offset:%d, info:%v", uploadID, patchInfo.UploadOffset, info.Offset, info)
	//	return c.Status(fiber.StatusBadRequest).JSON(
	//		models.NewResponse(1, "Invalid offset", nil))
	//}

	fileHeader := resumableInfo.File
	size := fileHeader.Size

	klog.Infof("fileHeader.Size:%d, info.Offset:%d, info.FileSize:%d",
		fileHeader.Size, info.Offset, info.FileSize)
	if !a.server.checkSize(size) || size+info.Offset > info.FileSize {
		klog.Warningf("Unsupported file size uploadSize:%d", size)
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "Unsupported file size", nil))
	}

	// Write the file contents to the file at the specified path
	fileSize, err := fileutils.SaveFile(fileHeader, fileutils.GetTempFilePathById(innerIdentifier))
	if err != nil {
		klog.Warningf("innerIdentifier:%s, info:%+v, err:%v", innerIdentifier, info, err)
		return c.Status(fiber.StatusInternalServerError).JSON(
			models.NewResponse(1, err.Error(), info))
	}

	info.Offset = fileSize
	a.server.fileInfoMgr.UpdateInfo(innerIdentifier, info)

	// Update file information for debug
	err = fileutils.UpdateFileInfo(info)
	if err != nil {
		klog.Warningf("innerIdentifier:%s, info:%+v, err:%v", innerIdentifier, info, err)
		return c.Status(fiber.StatusInternalServerError).JSON(
			models.NewResponse(1, err.Error(), info))
	}

	// Check if the file has been written
	if fileSize == info.FileSize {
		// Move the file to the specified upload path
		err = fileutils.MoveFileByInfo(info)
		if err != nil {
			klog.Warningf("innerIdentifier:%s, info:%+v, err:%v", innerIdentifier, info, err)
			return c.Status(fiber.StatusInternalServerError).JSON(
				models.NewResponse(1, err.Error(), info))
		}
		a.server.fileInfoMgr.DelFileInfo(innerIdentifier)

		klog.Infof("innerIdentifier:%s File uploaded successfully info:%+v", innerIdentifier, info)
		// Return successful response

		finishData := []map[string]interface{}{
			{
				"name": resumableInfo.ResumableFilename,
				"id":   uid.MakeUid(info.FullPath),
				"size": info.FileSize,
			},
		}
		return c.JSON(finishData)
		//return c.Status(fiber.StatusOK).JSON(
		//	models.NewResponse(0, "File uploaded successfully", info))
	}

	klog.Infof("innerIdentifier:%s File Continue uploading info:%+v", innerIdentifier, info)

	//return c.Status(fiber.StatusOK).JSON(
	//	models.NewResponse(0, "Continue uploading", info))
	return c.JSON(responseData)
}
