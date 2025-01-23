package app

import (
	"bytetrade.io/web3os/tapr/pkg/upload/fileutils"
	"bytetrade.io/web3os/tapr/pkg/upload/models"
	"bytetrade.io/web3os/tapr/pkg/upload/uid"
	"bytetrade.io/web3os/tapr/pkg/utils"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	CacheRequestPrefix = "/AppData"
	CachePathPrefix    = "/appcache"
	ExternalPathPrefix = "/data/External/"
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

	var uploadsDir = CachePathPrefix + "/" + cachePvc + "/files/uploadstemp"
	//var uploadsDir = ""
	//if val, ok := fileutils.UploadsDirs4[bflName]; ok {
	//	uploadsDir = val
	//} else {
	//	uploadsDir = CachePathPrefix + "/" + cachePvc + "/files/uploadstemp"
	//	fileutils.UploadsDirs4[bflName] = uploadsDir
	//}

	return bflName, userPvc, cachePvc, uploadsDir, nil
}

func extractPart(s string) string {
	if !strings.HasPrefix(s, ExternalPathPrefix) {
		return ""
	}

	s = s[len(ExternalPathPrefix):]

	index := strings.Index(s, "/")

	if index == -1 {
		return s
	} else {
		return s[:index]
	}
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

	// change temp file location
	extracted := extractPart(path)
	if extracted != "" {
		uploadsDir = ExternalPathPrefix + extracted + "/.uploadstemp"
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

	timestamp := time.Now().UnixNano()
	timestampStr := strconv.FormatInt(timestamp, 10)
	makeIDString := timestampStr + "_" + path
	uploadID := uid.MakeUid(makeIDString)
	IDCache.Add(uploadID, path, timestamp)

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

	uploadID := c.Query("upload_id", "")
	if uploadID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "upload_id invalid", nil))
	}
	uploadCache := IDCache.Get(uploadID)
	if uploadCache == nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "upload_id doesn't exist", nil))
	}

	responseData := make(map[string]interface{})
	responseData["uploadedBytes"] = 0

	// change temp file location
	extracted := extractPart(parentDir)
	if extracted != "" {
		uploadsDir = ExternalPathPrefix + extracted + "/.uploadstemp"
	}

	if !utils.PathExists(uploadsDir) {
		return c.JSON(responseData)
	}
	klog.Infof("c:%+v", c)

	//fullPath := AddVersionSuffix(filepath.Join(parentDir, fileName))
	fullPath := filepath.Join(parentDir, fileName)

	if uploadCache.filePath != fullPath {
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "upload_id and filepath don't match", nil))
	}

	dirPath := filepath.Dir(fullPath)

	//dstName := filepath.Base(fullPath)
	//tmpName := dstName + ".uploading"

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
	//innerIdentifier := uid.MakeUid(fullPath)
	innerIdentifier := uploadID
	tmpName := innerIdentifier
	fileutils.UploadsFiles4[innerIdentifier] = filepath.Join(uploadsDir, tmpName) // innerIdentifier)
	exist, info := a.server.fileInfoMgr.ExistFileInfo(innerIdentifier)
	//fileExist, fileLen := a.server.fileInfoMgr.CheckTempFile4(innerIdentifier, uploadsDir)
	fileExist, fileLen := a.server.fileInfoMgr.CheckTempFile4(tmpName, uploadsDir)
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
			a.server.fileInfoMgr.DelFileInfo4(innerIdentifier, tmpName, uploadsDir)
			//IDCache.Delete(uploadID)
		}
	}
	return c.JSON(responseData)
}

func checkMem() {
	v, err := mem.VirtualMemory()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Total: %v, Used: %v, Free: %v\n", v.Total, v.Used, v.Free)
}

func checkCpu() {
	percentages, err := cpu.Percent(time.Second, false)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	for _, percentage := range percentages {
		fmt.Printf("CPU Usage: %.2f%%\n", percentage)
	}
}

const (
	maxReasonableSpace = 1000 * 1e12 // 1000T
)

func checkDiskSpace(filePath string, newContentSize int64) (bool, int64, int64, int64, error) {
	reservedSpaceStr := os.Getenv("RESERVED_SPACE") // env is MB, default is 10000MB
	if reservedSpaceStr == "" {
		reservedSpaceStr = "10000"
	}
	reservedSpace, err := strconv.ParseInt(reservedSpaceStr, 10, 64)
	if err != nil {
		return false, 0, 0, 0, fmt.Errorf("failed to parse reserved space: %w", err)
	}
	reservedSpace *= 1024 * 1024

	var rootStat, dataStat syscall.Statfs_t

	err = syscall.Statfs("/", &rootStat)
	if err != nil {
		return false, 0, 0, 0, fmt.Errorf("failed to get root file system stats: %w", err)
	}
	rootAvailableSpace := int64(rootStat.Bavail * uint64(rootStat.Bsize))

	err = syscall.Statfs(filePath, &dataStat)
	if err != nil {
		return false, 0, 0, 0, fmt.Errorf("failed to get /data file system stats: %w", err)
	}
	dataAvailableSpace := int64(dataStat.Bavail * uint64(dataStat.Bsize))

	availableSpace := int64(0)
	if dataAvailableSpace >= maxReasonableSpace {
		availableSpace = rootAvailableSpace - reservedSpace
	} else {
		availableSpace = dataAvailableSpace - reservedSpace
	}

	requiredSpace := newContentSize

	if availableSpace >= requiredSpace {
		return true, requiredSpace, availableSpace, reservedSpace, nil
	}

	return false, requiredSpace, availableSpace, reservedSpace, nil
}

func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	var result string
	var value float64

	if bytes >= GB {
		value = float64(bytes) / GB
		result = fmt.Sprintf("%.4fG", value)
	} else if bytes >= MB {
		value = float64(bytes) / MB
		result = fmt.Sprintf("%.4fM", value)
	} else if bytes >= KB {
		value = float64(bytes) / KB
		result = fmt.Sprintf("%.4fK", value)
	} else {
		result = strconv.FormatInt(bytes, 10) + "B"
	}

	return result
}

func (a *appController) UploadChunks(c *fiber.Ctx) error {
	fmt.Println("*********Checking Chunk-relative Mem and CPU***************")
	checkMem()
	checkCpu()

	_, userPvc, cachePvc, uploadsDir, err := getPVC(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "bfl header missing or invalid", nil))
	}

	responseData := make(map[string]interface{})
	responseData["success"] = true

	uploadID := c.Params("uid")
	if uploadID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "upload_id invalid", nil))
	}
	uploadCache := IDCache.Get(uploadID)
	if uploadCache == nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "upload_id doesn't exist", nil))
	}

	//if !utils.PathExists(uploadsDir) {
	//	if err := os.MkdirAll(uploadsDir, os.ModePerm); err != nil {
	//		klog.Warningf("uploadID:%s, err:%v", uploadID, err)
	//		return c.Status(fiber.StatusInternalServerError).JSON(
	//			models.NewResponse(1, "failed to create folder", nil))
	//	}
	//}

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
	//if uploadID != uid.MakeUid(parentDir) {
	//	return c.Status(fiber.StatusBadRequest).JSON(
	//		models.NewResponse(1, "invalid upload link", nil))
	//}

	// change temp file location
	extracted := extractPart(parentDir)
	if extracted != "" {
		uploadsDir = ExternalPathPrefix + extracted + "/.uploadstemp"
	}
	if !utils.PathExists(uploadsDir) {
		if err := os.MkdirAll(uploadsDir, os.ModePerm); err != nil {
			klog.Warningf("uploadID:%s, err:%v", uploadID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(
				models.NewResponse(1, "failed to create folder", nil))
		}
	}

	resumableInfo.File, err = c.FormFile("file")
	if err != nil || resumableInfo.File == nil {
		klog.Warningf("uploadID:%s, Failed to parse file: %v\n", uploadID, err)
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "param invalid", nil))
	}

	klog.Infof("uploadID:%s, patchInfo:%+v", uploadID, resumableInfo)

	// Get file information based on upload ID
	//fullPath := AddVersionSuffix(filepath.Join(parentDir, resumableInfo.ResumableRelativePath))
	fullPath := filepath.Join(parentDir, resumableInfo.ResumableRelativePath)
	if uploadCache.filePath != fullPath {
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "invalid upload link", nil))
	}

	//dstName := filepath.Base(fullPath)
	//tmpName := dstName + ".uploading"
	//resumableIdentifier := resumableInfo.ResumableIdentifier
	//innerIdentifier := uid.MakeUid(fullPath)
	innerIdentifier := uploadID
	tmpName := innerIdentifier
	fileutils.UploadsFiles4[innerIdentifier] = filepath.Join(uploadsDir, tmpName) // innerIdentifier)
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

	//if resumableInfo.ResumableChunkNumber == 1 {
	//	fmt.Println("*********Checking Whole File Disk Space***************")
	//	spaceOk, needs, avails, reserved, err := checkDiskSpace(uploadsDir, resumableInfo.ResumableTotalSize)
	//	if err != nil {
	//		fileutils.RemoveTempFileAndInfoFile4(tmpName, uploadsDir)
	//		return c.Status(fiber.StatusInternalServerError).JSON(
	//			models.NewResponse(1, "Disk space check error", nil))
	//	}
	//	needsStr := formatBytes(needs)
	//	availsStr := formatBytes(avails)
	//	reservedStr := formatBytes(reserved)
	//	if spaceOk {
	//		spaceMessage := fmt.Sprintf("Sufficient disk space available. This file requires: %s, while %s is already available (with an additional %s reserved for the system).",
	//			needsStr, availsStr, reservedStr)
	//		fmt.Println(spaceMessage)
	//	} else {
	//		fileutils.RemoveTempFileAndInfoFile4(tmpName, uploadsDir)
	//		errorMessage := fmt.Sprintf("Insufficient disk space available. This file requires: %s, but only %s is available (with an additional %s reserved for the system).",
	//			needsStr, availsStr, reservedStr)
	//		return c.Status(fiber.StatusBadRequest).JSON(
	//			models.NewResponse(1, errorMessage, nil))
	//	}
	//}

	if !exist || innerIdentifier != info.ID {
		//clear temp file and reset info
		//fileutils.RemoveTempFileAndInfoFile4(innerIdentifier, uploadsDir)
		fileutils.RemoveTempFileAndInfoFile4(tmpName, uploadsDir)
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
			if info.Offset == info.FileSize {
				klog.Warningf("All file chunks have been uploaded, skip upload")
				finishData := []map[string]interface{}{
					{
						"name": resumableInfo.ResumableFilename,
						"id":   uploadID, // uid.MakeUid(info.FullPath),
						"size": info.FileSize,
					},
				}
				return c.JSON(finishData)
			}
			klog.Warningf("Unsupported file size uploadSize:%d", resumableInfo.ResumableTotalSize)
			return c.Status(fiber.StatusBadRequest).JSON(
				models.NewResponse(1, "Unsupported file size", nil))
		}

		info.FileSize = resumableInfo.ResumableTotalSize
		a.server.fileInfoMgr.UpdateInfo(innerIdentifier, info)

		//Generate unique Upload-ID
		//uploadID := uid.MakeUid(uploadInfo.FullPath)
		oExist, oInfo := a.server.fileInfoMgr.ExistFileInfo(innerIdentifier)
		//oFileExist, oFileLen := a.server.fileInfoMgr.CheckTempFile4(innerIdentifier, uploadsDir)
		oFileExist, oFileLen := a.server.fileInfoMgr.CheckTempFile4(tmpName, uploadsDir)
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
				a.server.fileInfoMgr.DelFileInfo4(innerIdentifier, tmpName, uploadsDir)
				//IDCache.Delete(uploadID)
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

	//fileExist, fileLen := a.server.fileInfoMgr.CheckTempFile4(innerIdentifier, uploadsDir)
	fileExist, fileLen := a.server.fileInfoMgr.CheckTempFile4(tmpName, uploadsDir)
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

	fmt.Println("*********Checking Chunk-relative Disk Space***************")
	spaceOk, needs, avails, reserved, err := checkDiskSpace(uploadsDir, info.FileSize-info.Offset)
	if err != nil {
		fileutils.RemoveTempFileAndInfoFile4(tmpName, uploadsDir)
		return c.Status(fiber.StatusInternalServerError).JSON(
			models.NewResponse(1, "Disk space check error", nil))
	}
	needsStr := formatBytes(needs)
	availsStr := formatBytes(avails)
	reservedStr := formatBytes(reserved)
	if spaceOk {
		spaceMessage := fmt.Sprintf("Sufficient disk space available. This file still requires: %s, while %s is already available (with an additional %s reserved for the system).",
			needsStr, availsStr, reservedStr)
		fmt.Println(spaceMessage)
	} else {
		fileutils.RemoveTempFileAndInfoFile4(tmpName, uploadsDir)
		errorMessage := fmt.Sprintf("Insufficient disk space available. This file still requires: %s, but only %s is available (with an additional %s reserved for the system).",
			needsStr, availsStr, reservedStr)
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, errorMessage, nil))
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

	var newFile bool = false
	if info.Offset != offset && offset == 0 {
		fmt.Println("Retransfering innerIdentifier:", innerIdentifier, ", uploadsDir:", uploadsDir, ", info.Offset:", info.Offset)
		//fileutils.ClearTempFileContent(innerIdentifier, uploadsDir)
		newFile = true
		info.Offset = offset
		a.server.fileInfoMgr.UpdateInfo(innerIdentifier, info)
	}

	klog.Infof("fileHeader.Size:%d, info.Offset:%d, info.FileSize:%d",
		fileHeader.Size, info.Offset, info.FileSize)
	if !a.server.checkSize(size) || size+info.Offset > info.FileSize {
		if info.Offset == info.FileSize {
			klog.Warningf("All file chunks have been uploaded, skip upload")
			finishData := []map[string]interface{}{
				{
					"name": resumableInfo.ResumableFilename,
					"id":   uploadID, //uid.MakeUid(info.FullPath),
					"size": info.FileSize,
				},
			}
			return c.JSON(finishData)
		}
		klog.Warningf("Unsupported file size uploadSize:%d", size)
		return c.Status(fiber.StatusBadRequest).JSON(
			models.NewResponse(1, "Unsupported file size", nil))
	}

	const maxRetries = 100
	for retry := 0; retry < maxRetries; retry++ {
		if info.Offset-offset > 0 {
			klog.Warningf("This file chunks have already been uploaded, skip upload")
			return c.JSON(responseData)
		}

		if info.Offset == offset {
			//fileSize, err := fileutils.SaveFile4(fileHeader, fileutils.GetTempFilePathById4(innerIdentifier, uploadsDir), newFile)
			fileSize, err := fileutils.SaveFile4(fileHeader, filepath.Join(uploadsDir, tmpName), newFile)
			if err != nil {
				klog.Warningf("innerIdentifier:%s, info:%+v, err:%v", innerIdentifier, info, err)
				return c.Status(fiber.StatusInternalServerError).JSON(
					models.NewResponse(1, err.Error(), info))
			}
			info.Offset = fileSize
			a.server.fileInfoMgr.UpdateInfo(innerIdentifier, info)
			break
		}

		time.Sleep(1000 * time.Millisecond)

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
		//err = fileutils.RenameFileByInfo4(info, uploadsDir)
		if err != nil {
			klog.Warningf("innerIdentifier:%s, info:%+v, err:%v", innerIdentifier, info, err)
			return c.Status(fiber.StatusInternalServerError).JSON(
				models.NewResponse(1, err.Error(), info))
		}
		a.server.fileInfoMgr.DelFileInfo4(innerIdentifier, tmpName, uploadsDir)

		klog.Infof("innerIdentifier:%s File uploaded successfully info:%+v", innerIdentifier, info)
		// Return successful response

		finishData := []map[string]interface{}{
			{
				"name": resumableInfo.ResumableFilename,
				"id":   uploadID, //uid.MakeUid(info.FullPath),
				"size": info.FileSize,
			},
		}

		// only delete cache when finished, other status by timed cleaning
		IDCache.Delete(uploadID)
		return c.JSON(finishData)
		//return c.Status(fiber.StatusOK).JSON(
		//	models.NewResponse(0, "File uploaded successfully", info))
	}

	klog.Infof("innerIdentifier:%s File Continue uploading info:%+v", innerIdentifier, info)

	//return c.Status(fiber.StatusOK).JSON(
	//	models.NewResponse(0, "Continue uploading", info))
	return c.JSON(responseData)
}
