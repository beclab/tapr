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

func (a *appController) PatchFile(c *fiber.Ctx) error {
	uploadID := c.Params("uid")

	if !utils.PathExists(fileutils.UploadsDir) {
		if err := os.MkdirAll(fileutils.UploadsDir, os.ModePerm); err != nil {
			klog.Warningf("uploadID:%s, err:%v", uploadID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(
				models.NewResponse(1, "failed to create folder", nil))
		}
	}

	klog.Infof("uploadID:%s, c:%+v", uploadID, c)

	var patchInfo models.FilePatchInfo
	if err := c.BodyParser(&patchInfo); err != nil {
		klog.Warningf("uploadID:%s, err:%v", uploadID, err)
		//todo check info valid
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "param invalid", nil))
	}

	var err error
	patchInfo.File, err = c.FormFile("file")
	if err != nil || patchInfo.File == nil {
		klog.Warningf("uploadID:%s, Failed to parse file: %v\n", uploadID, err)
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "param invalid", nil))
	}

	klog.Infof("uploadID:%s, patchInfo:%+v", uploadID, patchInfo)

	// Get file information based on upload ID
	exist, info := a.server.fileInfoMgr.ExistFileInfo(uploadID)
	if !exist {
		klog.Warningf("uploadID %s not exist", uploadID)
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "Invalid upload ID", nil))
	}
	klog.Infof("uploadID:%s, info:%+v", uploadID, info)
	if uploadID != info.ID {
		klog.Warningf("uploadID:%s diff from info:%+v", uploadID, info)
	}

	if patchInfo.UploadOffset == 0 {
		//clear temp file and reset info
		fileutils.RemoveTempFileAndInfoFile(uploadID)
		if info.Offset != 0 {
			info.Offset = 0
			a.server.fileInfoMgr.UpdateInfo(uploadID, info)
		}
	}

	fileExist, fileLen := a.server.fileInfoMgr.CheckTempFile(uploadID)
	if fileExist {
		klog.Infof("uploadID %s temp file exist, info.Offset:%d, fileLen:%d", uploadID, info.Offset, fileLen)
		if info.Offset != fileLen {
			info.Offset = fileLen
			a.server.fileInfoMgr.UpdateInfo(uploadID, info)
		}
	}

	// Check if file size and offset match
	if patchInfo.UploadOffset != info.Offset {
		klog.Warningf("uploadID %s, patchInfo.UploadOffset:%d diff from info.Offset:%d, info:%v", uploadID, patchInfo.UploadOffset, info.Offset, info)
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "Invalid offset", nil))
	}

	fileHeader := patchInfo.File
	size := fileHeader.Size

	klog.Infof("fileHeader.Size:%d, patchInfo.UploadOffset:%d, info.Offset:%d, info.FileSize:%d",
		fileHeader.Size, patchInfo.UploadOffset, info.Offset, info.FileSize)
	if !a.server.checkSize(size) || size+info.Offset > info.FileSize {
		klog.Warningf("Unsupported file size uploadSize:%d", size)
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "Unsupported file size", nil))
	}

	// Write the file contents to the file at the specified path
	fileSize, err := fileutils.SaveFile(fileHeader, fileutils.GetTempFilePathById(uploadID))
	if err != nil {
		klog.Warningf("uploadID:%s, info:%+v, err:%v", uploadID, info, err)
		return c.Status(fiber.StatusInternalServerError).JSON(
			models.NewResponse(1, err.Error(), info))
	}

	info.Offset = fileSize
	a.server.fileInfoMgr.UpdateInfo(uploadID, info)

	// Update file information for debug
	err = fileutils.UpdateFileInfo(info)
	if err != nil {
		klog.Warningf("uploadID:%s, info:%+v, err:%v", uploadID, info, err)
		return c.Status(fiber.StatusInternalServerError).JSON(
			models.NewResponse(1, err.Error(), info))
	}

	// Check if the file has been written
	if fileSize == info.FileSize {
		// Move the file to the specified upload path
		err = fileutils.MoveFileByInfo(info)
		if err != nil {
			klog.Warningf("uploadID:%s, info:%+v, err:%v", uploadID, info, err)
			return c.Status(fiber.StatusInternalServerError).JSON(
				models.NewResponse(1, err.Error(), info))
		}
		a.server.fileInfoMgr.DelFileInfo(uploadID)

		klog.Infof("uploadID:%s File uploaded successfully info:%+v", uploadID, info)
		// Return successful response
		return c.Status(fiber.StatusOK).JSON(
			models.NewResponse(0, "File uploaded successfully", info))
	}

	klog.Infof("uploadID:%s File Continue uploading info:%+v", uploadID, info)

	return c.Status(fiber.StatusOK).JSON(
		models.NewResponse(0, "Continue uploading", info))
}

func (a *appController) UploadFile(c *fiber.Ctx) error {
	if !utils.PathExists(fileutils.UploadsDir) {
		if err := os.MkdirAll(fileutils.UploadsDir, os.ModePerm); err != nil {
			klog.Warning("err:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(
				models.NewResponse(1, "failed to create folder", nil))
		}
	}
	klog.Infof("c:%+v", c)

	var uploadInfo models.FileMetaData
	if err := c.BodyParser(&uploadInfo); err != nil {
		klog.Warning("err:", err)
		//todo check info valid
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "param invalid", nil))
	}

	if !utils.CheckDirExist(uploadInfo.StoragePath) {
		klog.Warningf("Storage path %s is not exist or is not a dir", uploadInfo.StoragePath)
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "Storage path is not exist or is not a dir", nil))
	}

	uploadInfo.FullPath = filepath.Join(uploadInfo.StoragePath, uploadInfo.FileRelativePath)

	dirPath := filepath.Dir(uploadInfo.FullPath)

	if !utils.CheckDirExist(dirPath) {
		if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
			klog.Warning("err:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(
				models.NewResponse(1, "failed to create folder", nil))
		}
	}

	if uploadInfo.FileRelativePath == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "file_relative_path invalid", nil))
	}

	if strings.HasSuffix(uploadInfo.FileRelativePath, "/") { // upload dir, check exist or create it
		if !utils.CheckDirExist(uploadInfo.FullPath) {
			if err := os.MkdirAll(uploadInfo.FullPath, os.ModePerm); err != nil {
				klog.Warning("err:", err)
				return c.Status(fiber.StatusInternalServerError).JSON(
					models.NewResponse(1, "failed to create folder", nil))
			}
		}
		return c.Status(fiber.StatusOK).JSON(
			models.NewResponse(0, "success", nil))
	} else {
		if utils.CheckDirExist(uploadInfo.FullPath) {
			klog.Warningf("full path %s is a dir", uploadInfo.FullPath)
			return c.Status(fiber.StatusBadRequest).JSON(
				models.NewResponse(1, fmt.Sprintf("full path %s is a dir", uploadInfo.FullPath), nil))
		}
	}

	// Make support judgment after parsing the file type
	if !a.server.checkType(uploadInfo.FileType) {
		klog.Warningf("unsupported filetype:%s", uploadInfo.FileType)
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "Unsupported file type", nil))
	}

	if !a.server.checkSize(uploadInfo.FileSize) {
		klog.Warningf("Unsupported file size uploadSize:%d", uploadInfo.FileSize)
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "Unsupported file size", nil))
	}

	//Generate unique Upload-ID
	uploadID := uid.MakeUid(uploadInfo.FullPath)
	exist, info := a.server.fileInfoMgr.ExistFileInfo(uploadID)
	if exist {
		fileExist, fileLen := a.server.fileInfoMgr.CheckTempFile(uploadID)
		if fileExist {
			if info.Offset != fileLen {
				info.Offset = fileLen
				a.server.fileInfoMgr.UpdateInfo(uploadID, info)
			}
			klog.Infof("uploadID:%s, info.Offset:%d", uploadID, info.Offset)
			return c.Status(fiber.StatusOK).JSON(
				models.NewResponse(0, "success", info))
		} else if info.Offset == 0 {
			klog.Warningf("uploadID:%s, info.Offset:%d", uploadID, info.Offset)
			return c.Status(fiber.StatusOK).JSON(
				models.NewResponse(0, "success", info))
		} else {
			a.server.fileInfoMgr.DelFileInfo(uploadID)
		}
	}

	fileInfo := models.FileInfo{
		ID:           uploadID,
		Offset:       0,
		FileMetaData: uploadInfo,
	}

	err := a.server.fileInfoMgr.AddFileInfo(uploadID, fileInfo)
	if err != nil {
		klog.Warningf("uploadID:%s, err:%v", uploadID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(
			models.NewResponse(1, "Error save file info", nil))
	}

	// Update file information todo save to file
	//err := fileutils.UpdateFileInfo(fileInfo)
	//if err != nil {
	//	klog.Warning("err:", err)
	//	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
	//		"message": "Error updating file info",
	//	})
	//}

	klog.Infof("uploadID:%s, fileInfo:%+v", uploadID, fileInfo)
	return c.Status(fiber.StatusOK).JSON(
		models.NewResponse(0, "success", fileInfo))
}
