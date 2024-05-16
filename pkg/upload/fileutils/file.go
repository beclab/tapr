package fileutils

import (
	"bytetrade.io/web3os/tapr/pkg/upload/models"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"k8s.io/klog/v2"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// MoveFile moves file from src to dst.
// By default the rename filesystem system call is used. If src and dst point to different volumes
// the file copy is used as a fallback
func MoveFile(src, dst string) error {
	if os.Rename(src, dst) == nil {
		return nil
	}

	return moveFile(src, dst)
}

// moveFile copies a file from source to dest and returns
// an error if any.
func moveFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	err = os.Remove(src)
	if err != nil {
		return err
	}

	return nil
}

func HashFileByAlgo(filePath, algo string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hashcode := getHash(algo)
	if _, err := io.Copy(hashcode, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hashcode.Sum(nil)), nil
}

func HashFileHeaderByAlgo(fh *multipart.FileHeader, algo string) (string, error) {
	file, err := fh.Open()
	if err != nil {
		return "nil", err
	}

	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {
		}
	}(file)

	hashcode := getHash(algo)

	if _, err := io.Copy(hashcode, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hashcode.Sum(nil)), nil
}

func getHash(algo string) hash.Hash {
	switch algo {
	case "md5":
		return md5.New()
	case "sha256":
		return sha256.New()
	case "sha512":
		return sha512.New()
	case "sha1":
		return sha1.New()
	default:
		return sha256.New()
	}
}

func MergeChunkFile(dir string) (int64, error) {
	start := time.Now().UnixMicro()
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

	// create complete file
	completeFile, err := os.Create(fmt.Sprintf("%s/complete", dir))
	if err != nil {
		return 0, err
	}

	for _, file := range files {
		if file.IsDir() || strings.HasPrefix(file.Name(), ".") {
			continue
		}

		// read chunk file
		bytes, err := os.ReadFile(fmt.Sprintf("%s/%s", dir, file.Name()))
		if err != nil {
			return 0, err
		}

		// write data to complete file
		_, err = completeFile.Write(bytes)
		if err != nil {
			return 0, err
		}
	}

	end := time.Now().UnixMicro()
	timeSend := end - start
	return timeSend, nil
}

func GetTempFilePathById(id string) string {
	return filepath.Join(UploadsDir, id)
}

func SaveFile(fileHeader *multipart.FileHeader, filePath string) (int64, error) {
	// Open source file
	file, err := fileHeader.Open()
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// Create target file
	dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return 0, err
	}
	defer dstFile.Close()

	// Write the contents of the source file to the target file
	_, err = io.Copy(dstFile, file)
	if err != nil {
		return 0, err
	}

	// Get new file size
	fileInfo, err := dstFile.Stat()
	if err != nil {
		return 0, err
	}
	fileSize := fileInfo.Size()

	return fileSize, nil
}

func UpdateFileInfo(fileInfo models.FileInfo) error {
	// Construct file information path
	infoPath := filepath.Join(UploadsDir, fileInfo.ID+".info")

	// Convert file information to JSON string
	infoJSON, err := json.Marshal(fileInfo)
	if err != nil {
		return err
	}

	// Write file information
	err = ioutil.WriteFile(infoPath, infoJSON, 0644)
	if err != nil {
		return err
	}

	return nil
}

func RemoveTempFileAndInfoFile(uid string) {
	removeTempFile(uid)
	removeInfoFile(uid)
}

func removeTempFile(uid string) {
	filePath := filepath.Join(UploadsDir, uid)
	err := os.Remove(filePath)
	if err != nil {
		klog.Warningf("remove %s err:%v", filePath, err)
	}

}

func MoveFileByInfo(fileInfo models.FileInfo) error {
	// Construct file path
	filePath := filepath.Join(UploadsDir, fileInfo.ID)

	// Construct target path
	destinationPath := fileInfo.FullPath

	// Move files to target path
	err := MoveFile(filePath, destinationPath)
	if err != nil {
		return err
	}

	// Remove info file
	removeInfoFile(fileInfo.ID)

	return nil
}

func removeInfoFile(uid string) {
	infoPath := filepath.Join(UploadsDir, uid+".info")
	err := os.Remove(infoPath)
	if err != nil {
		klog.Warningf("remove %s err:%v", infoPath, err)
	}
}
