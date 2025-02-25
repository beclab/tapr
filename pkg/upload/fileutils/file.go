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
	"path"
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

func ioCopyFileWithBuffer(sourcePath, targetPath string, bufferSize int) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	dir := filepath.Dir(targetPath)
	baseName := filepath.Base(targetPath)

	tempFileName := fmt.Sprintf(".uploading_%s", baseName)
	tempFilePath := filepath.Join(dir, tempFileName)

	targetFile, err := os.OpenFile(tempFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer targetFile.Close()

	buf := make([]byte, bufferSize)
	for {
		n, err := sourceFile.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}
		if _, err := targetFile.Write(buf[:n]); err != nil {
			return err
		}
	}

	if err := targetFile.Sync(); err != nil {
		return err
	}
	return os.Rename(tempFilePath, targetPath)
}

// moveFile copies a file from source to dest and returns
// an error if any.
func moveFile(src, dst string) error {
	//srcFile, err := os.Open(src)
	//if err != nil {
	//	return err
	//}
	//defer srcFile.Close()
	//
	//dstFile, err := os.Create(dst)
	//if err != nil {
	//	return err
	//}
	//defer dstFile.Close()
	//
	//_, err = io.Copy(dstFile, srcFile)
	//if err != nil {
	//	return err
	//}

	err := ioCopyFileWithBuffer(src, dst, 8*1024*1024)
	if err != nil {
		fmt.Println(err)
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

func GetTempFilePathById4(id string, uploadsDir string) string {
	return filepath.Join(uploadsDir, id)
}

func SaveFile4(fileHeader *multipart.FileHeader, filePath string, newFile bool, offset int64) (int64, error) {
	startTime := time.Now()
	fmt.Printf("--- Function SaveFile4 started at: %s\n", startTime)

	defer func() {
		endTime := time.Now()
		fmt.Printf("--- Function SaveFile4 ended at: %s\n", endTime)
	}()

	// Open source file
	file, err := fileHeader.Open()
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// Determine file open flags based on newFile parameter
	var flags int
	if newFile {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	} else {
		flags = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	}

	// Create target file with appropriate flags
	dstFile, err := os.OpenFile(filePath, flags, 0644)
	if err != nil {
		return 0, err
	}
	defer dstFile.Close()

	// Seek to the specified offset if not creating a new file
	if !newFile {
		_, err = dstFile.Seek(offset, io.SeekStart)
		if err != nil {
			return 0, err
		}
	}

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

//func SaveFile4(fileHeader *multipart.FileHeader, filePath string, newFile bool) (int64, error) {
//	// Open source file
//	file, err := fileHeader.Open()
//	if err != nil {
//		return 0, err
//	}
//	defer file.Close()
//
//	// Determine file open flags based on newFile parameter
//	var flags int
//	if newFile {
//		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
//	} else {
//		flags = os.O_WRONLY | os.O_CREATE | os.O_APPEND
//	}
//
//	// Create target file with appropriate flags
//	dstFile, err := os.OpenFile(filePath, flags, 0644)
//	if err != nil {
//		return 0, err
//	}
//	defer dstFile.Close()
//
//	// Write the contents of the source file to the target file
//	_, err = io.Copy(dstFile, file)
//	if err != nil {
//		return 0, err
//	}
//
//	// Get new file size
//	fileInfo, err := dstFile.Stat()
//	if err != nil {
//		return 0, err
//	}
//	fileSize := fileInfo.Size()
//
//	return fileSize, nil
//}

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

func ParseContentRange(ranges string) (int64, int64, bool) {
	start := strings.Index(ranges, "bytes")
	end := strings.Index(ranges, "-")
	slash := strings.Index(ranges, "/")

	if start < 0 || end < 0 || slash < 0 {
		return -1, -1, false
	}

	startStr := strings.TrimLeft(ranges[start+len("bytes"):end], " ")
	firstByte, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		return -1, -1, false
	}

	lastByte, err := strconv.ParseInt(ranges[end+1:slash], 10, 64)
	if err != nil {
		return -1, -1, false
	}

	fileSize, err := strconv.ParseInt(ranges[slash+1:], 10, 64)
	if err != nil {
		return -1, -1, false
	}

	if firstByte > lastByte || lastByte >= fileSize {
		return -1, -1, false
	}

	//fsm.rstart = firstByte
	//fsm.rend = lastByte
	//fsm.fsize = fileSize

	return firstByte, lastByte, true
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

func UpdateFileInfo4(fileInfo models.FileInfo, uploadsDir string) error {
	// Construct file information path
	//infoPath := filepath.Join(uploadsDir, filepath.Base(fileInfo.FullPath)+".uploading.info")
	infoPath := filepath.Join(uploadsDir, fileInfo.ID+".info")

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

func RemoveTempFileAndInfoFile4(uid string, uploadsDir string) {
	removeTempFile4(uid, uploadsDir)
	removeInfoFile4(uid, uploadsDir)
}

func removeTempFile4(uid string, uploadsDir string) {
	filePath := filepath.Join(uploadsDir, uid)
	err := os.Remove(filePath)
	if err != nil {
		klog.Warningf("remove %s err:%v", filePath, err)
	}

}

func ClearTempFileContent(uid string, uploadsDir string) {
	filePath := filepath.Join(uploadsDir, uid)

	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		klog.Warningf("failed to open file %s: %v", filePath, err)
		return
	}
	defer file.Close()

	err = file.Truncate(0)
	if err != nil {
		klog.Warningf("failed to truncate file %s: %v", filePath, err)
	}
}

func AddVersionSuffix(source string) string {
	counter := 1
	dir, name := path.Split(source)
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)

	for {
		if _, err := os.Stat(source); err == nil {
			renamed := fmt.Sprintf("%s(%d)%s", base, counter, ext)
			source = path.Join(dir, renamed)
			counter++
		} else if os.IsNotExist(err) {
			break
		} else {
			fmt.Println("Error checking file:", err)
			break
		}
	}

	return source
}

func RenameFileByInfo4(fileInfo models.FileInfo, uploadsDir string) error {
	// Construct the current file path
	//filePath := fileInfo.FullPath + ".uploading"
	filePath := filepath.Join(uploadsDir, fileInfo.ID)

	// Construct the target path
	destinationPath := AddVersionSuffix(fileInfo.FullPath)

	// Perform the move operation by renaming the file
	err := os.Rename(filePath, destinationPath)
	if err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	// Optionally, you might want to log success or perform additional operations here

	return nil
}

func MoveFileByInfo4(fileInfo models.FileInfo, uploadsDir string) error {
	// Construct file path
	filePath := filepath.Join(uploadsDir, fileInfo.ID)

	// Construct target path
	destinationPath := AddVersionSuffix(fileInfo.FullPath)

	// Move files to target path
	err := MoveFile(filePath, destinationPath)
	if err != nil {
		return err
	}

	// Remove info file
	removeInfoFile4(fileInfo.ID, uploadsDir)

	return nil
}

func removeInfoFile4(uid string, uploadsDir string) {
	infoPath := filepath.Join(uploadsDir, uid+".info")
	err := os.Remove(infoPath)
	if err != nil {
		klog.Warningf("remove %s err:%v", infoPath, err)
	}
}
