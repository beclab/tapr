package app

import (
	"bytetrade.io/web3os/tapr/pkg/upload/fileutils"
	"bytetrade.io/web3os/tapr/pkg/upload/models"
	"bytetrade.io/web3os/tapr/pkg/utils"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"k8s.io/klog/v2"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	UploadDir = "/data/Home/Documents"
	//UploadDir   = "/Users/yangtao/work/src/data"
)

// UploadSmallFile @Summary Upload Small File
// @Description Uploads a small file to the specified storage path
// @Accept multipart/form-data
// @Param file formData file "File to upload"
// @Param storage_path formData string "Storage path for the file"
// @Success 200
// @Failure 400/500...
// @Router /tapr/upload/small [post]
func (a *appController) UploadSmallFile(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		klog.Warning("err:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	if file.Size > a.server.limitedSize {
		klog.Warningf("file.Size:%d exceeds the limit", file.Size)
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
			"message": "File size exceeds the limit. Please use the chunked upload API.",
		})
	}

	storageRelativePath := c.FormValue("storage_path")

	storagePath := UploadDir
	if storageRelativePath != "" {
		storagePath = path.Join(UploadDir, storageRelativePath)
	}
	if !utils.CheckDirExist(storagePath) {
		klog.Warningf("Storage path %s is not exist", storagePath)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Storage path is not exist",
		})
	}

	tempDir, err := os.MkdirTemp("", "upload")
	if err != nil {
		klog.Warning("err:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	fileTempPath := filepath.Join(tempDir, file.Filename)
	err = c.SaveFile(file, fileTempPath)
	if err != nil {
		klog.Warning("err:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": err.Error(),
		})
	}
	defer os.RemoveAll(tempDir)

	dstFile := filepath.Join(storagePath, file.Filename)
	err = fileutils.MoveFile(fileTempPath, dstFile)
	if err != nil {
		klog.Warning("err:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	//calc ETag and set header
	fileETag, err := fileutils.HashFileByAlgo(dstFile, "sha1")
	if err != nil {
		klog.Warning("err:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": err.Error(),
		})
	}
	c.Set("ETag", fileETag)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"code":    0,
		"message": "success",
	})
}

func (a *appController) UploadChunk(c *fiber.Ctx) error {
	// parse param
	var chunkFileUp models.UploadChunkFile
	if err := c.BodyParser(&chunkFileUp); err != nil {
		klog.Warning("err:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "param invalid",
		})
	}

	var err error
	chunkFileUp.ChunkFile, err = c.FormFile("chunk_file")
	if err != nil || chunkFileUp.ChunkFile == nil {
		klog.Warningf("Failed to parse chunk_file: %v\n", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "param invalid",
		})
	}

	klog.Infof("chunkFileUp:%+v", chunkFileUp)

	// param info
	chunkIndex := chunkFileUp.ChunkIndex
	chunkHash := chunkFileUp.ChunkHash
	chunkSum := chunkFileUp.ChunkSum
	fileHash := chunkFileUp.FileHash

	// get file header information
	fileHeader := chunkFileUp.ChunkFile

	// file information
	//contentType := fileHeader.Header.Get("Content-Type")
	filename := fileHeader.Filename
	size := fileHeader.Size

	if size > int64(a.server.limitedSize) {
		klog.Warningf("file.Size:%d exceeds the limit", size)
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
			"message": "File size exceeds the limit. Please use the chunked upload API.",
		})
	}

	// get the hash value of a chunked file
	sha1Hash, err := fileutils.HashFileHeaderByAlgo(fileHeader, "sha1")
	if err != nil {
		klog.Warning("err:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "ile is incomplete",
		})
	}

	klog.Info("sha1Hash:", sha1Hash)

	// if the hashes do not match, the file is incomplete
	if chunkHash != sha1Hash {
		klog.Warningf("chunkHash:%s, sha1Hash:%s not same", chunkHash, sha1Hash)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "file is incomplete",
		})
	}

	// folder path where chunked files are saved
	dir := fmt.Sprintf("public/file/%s", fileHash)
	// the complete file path of a single chunked file. The chunked file is named index-hash.
	dst := fmt.Sprintf("public/file/%s/%s", fileHash, chunkIndex+"-"+chunkHash)

	// determine whether the folder exists, create the folder if it does not exist
	if !utils.PathExists(dir) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			klog.Warning("err:", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "failed to create folder",
			})
		}
	}

	// determine whether the chunked file already exists. If it exists, it will directly return success.
	if utils.PathExists(dst) {
		klog.Warningf("dst:%s already exist", dst)
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message": "this chunked file has already been uploaded",
		})
	}

	// save file
	if err := c.SaveFile(fileHeader, dst); err != nil {
		klog.Warning("err:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "failed to save file",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		//"content_type": contentType,
		"filename":    filename,
		"size":        size,
		"chunk_index": chunkIndex,
		"chunk_sum":   chunkSum,
		"hash_sha1":   sha1Hash,
	})
}

func (a *appController) MergeChunk(c *fiber.Ctx) error {
	// get parameters
	var mergeChunk models.MergeChunkFile
	if err := c.BodyParser(&mergeChunk); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "param invalid",
		})
	}

	storagePath := UploadDir
	if mergeChunk.StoragePath != "" {
		storagePath = path.Join(UploadDir, mergeChunk.StoragePath)
	}
	if !utils.CheckDirExist(storagePath) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Storage relative path is not exist",
		})
	}

	// parameter information
	fileHash := mergeChunk.FileHash
	chunkSum := mergeChunk.ChunkSum

	// folder path where chunked files are saved
	dir := fmt.Sprintf("public/file/%s", fileHash)
	completeFile := fmt.Sprintf("%s/complete", dir)

	// determine whether a complete file exists
	exists := utils.PathExists(completeFile)
	if exists {
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message": "the complete file already exists",
		})
	}

	// read all chunked files in a folder
	files, err := os.ReadDir(dir)
	if err != nil {
		klog.Info("err", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "failed to read folder",
		})
	}

	// determine whether all chunks are complete
	if chunkSum != len(files) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "file chunks are incomplete",
		})
	}

	// merge files, the complete file is hash/complete
	timeSpend, err := fileutils.MergeChunkFile(dir)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "failed to merge files",
		})
	}

	dstFile := filepath.Join(storagePath, mergeChunk.FileName)
	err = fileutils.MoveFile(completeFile, dstFile)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": err.Error(),
		})
	}
	defer os.RemoveAll(dir)

	//calc ETag and set header
	fileETag, err := fileutils.HashFileByAlgo(dstFile, "sha1")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": err.Error(),
		})
	}
	c.Set("ETag", fileETag)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success":    true,
		"time_spend": timeSpend,
	})
}

func (a *appController) ChunkState(c *fiber.Ctx) error {
	hash := c.Query("hash")
	// folder path
	dir := fmt.Sprintf("public/file/%s", hash)

	// determine whether the folder exists
	dirExists := utils.PathExists(dir)
	// folder does not exist
	if !dirExists {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "no such file",
		})
	}

	// full file path
	completeFile := fmt.Sprintf("%s/complete", dir)
	// determine whether a complete file exists
	exists := utils.PathExists(completeFile)

	// complete file exists and uploaded successfully
	if exists {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "upload successful",
		})
	}

	// read files in a folder sorted by file name index
	files, _ := os.ReadDir(dir)
	sort.Slice(files, func(i, j int) bool {
		// get file index
		filename := files[i].Name()
		index := strings.Split(filename, "-")[0]

		indexInt, _ := strconv.Atoi(index)
		nextInt, _ := strconv.Atoi(strings.Split(files[j].Name(), "-")[0])
		return indexInt < nextInt
	})

	var indexes []int
	// traverse chunked files
	for _, file := range files {
		filename := file.Name()
		index := strings.Split(filename, "-")[0]
		indexInt, _ := strconv.Atoi(index)
		indexes = append(indexes, indexInt)
	}

	if len(indexes) == 0 {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "no fragment files yet",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"indexes": indexes,
		"message": "get the uploaded chunk index",
	})
}
