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
	"time"
)

const (
	CacheRequestPrefix = "/AppData"
	CachePathPrefix    = "/appcache"
)

func getPVC(c *fiber.Ctx) (string, string, string, string, error) {
	bflName := c.Get("X-Bfl-User")
	klog.Info("BFL_NAME: ", bflName)

	userPvc, err := PVCs.getUserPVCOrCache(bflName) // appdata.GetAnnotation(p.mainCtx, p.k8sClient, "userspace_pvc", bflName)
	if err != nil {
		klog.Info(err)
		return bflName, "", "", "", err
	} else {
		klog.Info("user-space pvc: ", userPvc)
	}

	cachePvc, err := PVCs.getCachePVCOrCache(bflName) // appdata.GetAnnotation(p.mainCtx, p.k8sClient, "appcache_pvc", bflName)
	if err != nil {
		klog.Info(err)
		return bflName, "", "", "", err
	} else {
		klog.Info("appcache pvc: ", cachePvc)
	}

	var uploadsDir = ""
	if val, ok := fileutils.UploadsDirs4[bflName]; ok {
		uploadsDir = val
	} else {
		uploadsDir = CachePathPrefix + "/" + cachePvc + "/files/uploadstemp"
		fileutils.UploadsDirs4[bflName] = uploadsDir
	}

	return bflName, userPvc, cachePvc, uploadsDir, nil
}

func (a *appController) UploadLink(c *fiber.Ctx) error {
	_, userPvc, cachePvc, uploadsDir, err := getPVC(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "bfl header missing or invalid", nil))
	}

	path := c.Query("p", "")
	if path == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "missing path query parameter", nil))
	}

	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}
	if strings.HasPrefix(path, CacheRequestPrefix) {
		path = CachePathPrefix + strings.TrimPrefix(path, CacheRequestPrefix)
		path = rewriteUrl(path, cachePvc, CachePathPrefix)
	} else {
		path = rewriteUrl(path, userPvc, "")
	}

	if !utils.PathExists(uploadsDir) {
		if err := os.MkdirAll(uploadsDir, os.ModePerm); err != nil {
			klog.Warning("err:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(
				models.NewResponse(1, "failed to create folder", nil))
		}
	}
	klog.Infof("c:%+v", c)

	if !utils.CheckDirExist(path) {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			klog.Warning("err:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(
				models.NewResponse(1, "failed to create folder", nil))
		}
	}

	uploadID := uid.MakeUid(path)

	uploadLink := fmt.Sprintf("/upload/upload-link/%s", uploadID)

	return c.SendString(uploadLink)
}

func (a *appController) UploadedBytes(c *fiber.Ctx) error {
	_, userPvc, cachePvc, uploadsDir, err := getPVC(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "bfl header missing or invalid", nil))
	}

	parentDir := c.Query("parent_dir", "")
	if parentDir == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "missing parent_dir query parameter", nil))
	}

	if !strings.HasSuffix(parentDir, "/") {
		parentDir = parentDir + "/"
	}
	if strings.HasPrefix(parentDir, CacheRequestPrefix) {
		parentDir = CachePathPrefix + strings.TrimPrefix(parentDir, CacheRequestPrefix)
		parentDir = rewriteUrl(parentDir, cachePvc, CachePathPrefix)
	} else {
		parentDir = rewriteUrl(parentDir, userPvc, "")
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

	if !utils.PathExists(uploadsDir) {
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
	fileExist, fileLen := a.server.fileInfoMgr.CheckTempFile4(innerIdentifier, uploadsDir)
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
			a.server.fileInfoMgr.DelFileInfo4(innerIdentifier, uploadsDir)
		}
	}
	return c.JSON(responseData)
}

func (a *appController) UploadChunks(c *fiber.Ctx) error {
	_, userPvc, cachePvc, uploadsDir, err := getPVC(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "bfl header missing or invalid", nil))
	}

	responseData := make(map[string]interface{})
	responseData["success"] = true

	uploadID := c.Params("uid")

	if !utils.PathExists(uploadsDir) {
		if err := os.MkdirAll(uploadsDir, os.ModePerm); err != nil {
			klog.Warningf("uploadID:%s, err:%v", uploadID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(
				models.NewResponse(1, "failed to create folder", nil))
		}
	}

	klog.Infof("uploadID:%s, c:%+v", uploadID, c)

	var resumableInfo models.ResumableInfo
	if err = c.BodyParser(&resumableInfo); err != nil {
		klog.Warningf("uploadID:%s, err:%v", uploadID, err)
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "param invalid", nil))
	}

	parentDir := resumableInfo.ParentDir
	if !strings.HasSuffix(parentDir, "/") {
		parentDir = parentDir + "/"
	}
	if strings.HasPrefix(parentDir, CacheRequestPrefix) {
		parentDir = CachePathPrefix + strings.TrimPrefix(parentDir, CacheRequestPrefix)
		parentDir = rewriteUrl(parentDir, cachePvc, CachePathPrefix)
	} else {
		parentDir = rewriteUrl(parentDir, userPvc, "")
	}
	if uploadID != uid.MakeUid(parentDir) {
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "invalid upload link", nil))
	}

	resumableInfo.File, err = c.FormFile("file")
	if err != nil || resumableInfo.File == nil {
		klog.Warningf("uploadID:%s, Failed to parse file: %v\n", uploadID, err)
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "param invalid", nil))
	}

	klog.Infof("uploadID:%s, patchInfo:%+v", uploadID, resumableInfo)

	// Get file information based on upload ID
	fullPath := filepath.Join(parentDir, resumableInfo.ResumableRelativePath)
	//resumableIdentifier := resumableInfo.ResumableIdentifier
	innerIdentifier := uid.MakeUid(fullPath)
	exist, info := a.server.fileInfoMgr.ExistFileInfo(innerIdentifier)
	if !exist {
		klog.Warningf("innerIdentifier %s not exist", innerIdentifier)
		//return c.Status(fiber.StatusBadRequest).JSON(
		//	models.NewResponse(1, "Invalid innerIdentifier", nil))
	}
	klog.Infof("innerIdentifier:%s, info:%+v", innerIdentifier, info)
	if innerIdentifier != info.ID {
		klog.Warningf("innerIdentifier:%s diff from info:%+v", innerIdentifier, info)
	}

	if !exist || innerIdentifier != info.ID {
		//clear temp file and reset info
		fileutils.RemoveTempFileAndInfoFile4(innerIdentifier, uploadsDir)
		if info.Offset != 0 {
			info.Offset = 0
			a.server.fileInfoMgr.UpdateInfo(innerIdentifier, info)
		}

		//do creation when the first chunk
		if !utils.CheckDirExist(parentDir) {
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

		info.FileSize = resumableInfo.ResumableTotalSize
		a.server.fileInfoMgr.UpdateInfo(innerIdentifier, info)

		//Generate unique Upload-ID
		//uploadID := uid.MakeUid(uploadInfo.FullPath)
		oExist, oInfo := a.server.fileInfoMgr.ExistFileInfo(innerIdentifier)
		oFileExist, oFileLen := a.server.fileInfoMgr.CheckTempFile4(innerIdentifier, uploadsDir)
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
				a.server.fileInfoMgr.DelFileInfo4(innerIdentifier, uploadsDir)
			}
		}

		fileInfo := models.FileInfo{
			ID:     innerIdentifier,
			Offset: 0,
			FileMetaData: models.FileMetaData{
				FileRelativePath: resumableInfo.ResumableRelativePath,
				FileType:         resumableInfo.ResumableType,
				FileSize:         resumableInfo.ResumableTotalSize,
				StoragePath:      parentDir, //resumableInfo.ParentDir,
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
		info = fileInfo
	}

	fileExist, fileLen := a.server.fileInfoMgr.CheckTempFile4(innerIdentifier, uploadsDir)
	if fileExist {
		klog.Infof("innerIdentifier %s temp file exist, info.Offset:%d, fileLen:%d", innerIdentifier, info.Offset, fileLen)
		if info.Offset != fileLen {
			info.Offset = fileLen
			a.server.fileInfoMgr.UpdateInfo(innerIdentifier, info)
		}
	}

	// Check if file size and offset match
	// not functional when resumable.js
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

	ranges := c.Get("Content-Range")
	var offset int64
	var parsed bool
	if ranges != "" {
		offset, parsed = fileutils.ParseContentRange(ranges)
		if !parsed {
			return c.Status(fiber.StatusBadRequest).JSON(
				models.NewResponse(1, "Invalid content range", nil))
		}
	}

	const maxRetries = 100
	for retry := 0; retry < maxRetries; retry++ {
		if info.Offset == offset {
			fileSize, err := fileutils.SaveFile(fileHeader, fileutils.GetTempFilePathById4(innerIdentifier, uploadsDir))
			if err != nil {
				klog.Warningf("innerIdentifier:%s, info:%+v, err:%v", innerIdentifier, info, err)
				return c.Status(fiber.StatusInternalServerError).JSON(
					models.NewResponse(1, err.Error(), info))
			}
			info.Offset = fileSize
			a.server.fileInfoMgr.UpdateInfo(innerIdentifier, info)
			break
		}

		time.Sleep(500 * time.Millisecond)

		klog.Infof("Waiting for info.Offset to match offset (%d != %d), retry %d/%d", info.Offset, offset, retry+1, maxRetries)

		if retry < maxRetries-1 {
			exist, info = a.server.fileInfoMgr.ExistFileInfo(innerIdentifier)
			if !exist {
				klog.Warningf("innerIdentifier %s not exist", innerIdentifier)
				return c.Status(fiber.StatusBadRequest).JSON(
					models.NewResponse(1, "Invalid innerIdentifier", nil))
			}
			continue
		}

		return c.Status(fiber.StatusInternalServerError).JSON(
			models.NewResponse(1, "Failed to match offset after multiple retries", info))
	}

	// Update file information for debug
	err = fileutils.UpdateFileInfo4(info, uploadsDir)
	if err != nil {
		klog.Warningf("innerIdentifier:%s, info:%+v, err:%v", innerIdentifier, info, err)
		return c.Status(fiber.StatusInternalServerError).JSON(
			models.NewResponse(1, err.Error(), info))
	}

	// Check if the file has been written
	if info.Offset == info.FileSize {
		// Move the file to the specified upload path
		err = fileutils.MoveFileByInfo4(info, uploadsDir)
		if err != nil {
			klog.Warningf("innerIdentifier:%s, info:%+v, err:%v", innerIdentifier, info, err)
			return c.Status(fiber.StatusInternalServerError).JSON(
				models.NewResponse(1, err.Error(), info))
		}
		a.server.fileInfoMgr.DelFileInfo4(innerIdentifier, uploadsDir)

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