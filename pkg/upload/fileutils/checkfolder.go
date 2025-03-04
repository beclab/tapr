package fileutils

import (
	"fmt"
	"github.com/robfig/cron/v3"
	//"io/ioutil"
	"k8s.io/klog/v2"
	"os"
	//"path/filepath"
	"time"
)

const (
	expireTime = time.Duration(24) * time.Hour
)

func Init() {
	////cronDeleteOldfolders(UploadsDir)
	//for _, uploadsDir := range UploadsDirs4 {
	//	cronDeleteOldfolders(uploadsDir)
	//}
	////checkTempDir(UploadsDir)
	for _, uploadsFile := range UploadsFiles4 {
		cronDeleteOldFile(uploadsFile)
	}
}

//func checkTempDir(dirPath string) {
//	os.RemoveAll(dirPath)
//	err := os.MkdirAll(dirPath, os.ModePerm)
//	if err != nil {
//		klog.Warning("MkdirAll err:", err)
//		return
//	}
//	klog.Infof("MkdirAll %s success", dirPath)
//}

//func cronDeleteOldfolders(dir string) {
//	c := cron.New()
//
//	_, err := c.AddFunc("30 * * * *", func() {
//		subErr := DeleteOldSubfolders(dir)
//		if subErr != nil {
//			klog.Warningf("DeleteOldSubfolders %s, err:%v", dir, subErr)
//		}
//	})
//	if err != nil {
//		klog.Warningf("AddFunc DeleteOldSubfolders err:%v", err)
//	}
//
//	c.Start()
//}

func cronDeleteOldFile(filePath string) {
	c := cron.New()

	_, err := c.AddFunc("30 * * * *", func() {
		subErr := DeleteIfFileExpired(filePath)
		if subErr != nil {
			klog.Warningf("DeleteOldFile %s, err:%v", filePath, subErr)
		}
	})
	if err != nil {
		klog.Warningf("AddFunc DeleteOldSubfolders err:%v", err)
	}

	c.Start()
}

//func DeleteOldSubfolders(parentDir string) error {
//	// Get all subfolders under the parent directory
//	subfolders, err := ioutil.ReadDir(parentDir)
//	if err != nil {
//		return fmt.Errorf("failed to read subfolders: %s", err.Error())
//	}
//
//	// Iterate over each subfolder
//	for _, subfolder := range subfolders {
//		if !subfolder.IsDir() {
//			modTime := subfolder.ModTime()
//			if time.Since(modTime) < expireTime {
//				subfolderPath := filepath.Join(parentDir, subfolder.Name())
//				err := os.RemoveAll(subfolderPath)
//				if err != nil {
//					return fmt.Errorf("failed to delete subfile: %s", err.Error())
//				}
//				klog.Infof("Deleted subfile: %s\n", subfolderPath)
//			}
//			continue
//		}
//
//		subfolderPath := filepath.Join(parentDir, subfolder.Name())
//
//		// Get all files in the subfolder
//		files, err := ioutil.ReadDir(subfolderPath)
//		if err != nil {
//			return fmt.Errorf("failed to read files in subfolder: %s", err.Error())
//		}
//
//		// Check if all files in the subfolder are older than 24 hours
//		allFilesOld := true
//		for _, file := range files {
//			filePath := filepath.Join(subfolderPath, file.Name())
//			modTime := file.ModTime()
//			if time.Since(modTime) < expireTime {
//				allFilesOld = false
//				break
//			}
//			klog.Infof("File %s modified %v ago\n", filePath, time.Since(modTime))
//		}
//
//		// Delete the subfolder if all files are older than 24 hours
//		if allFilesOld {
//			err := os.RemoveAll(subfolderPath)
//			if err != nil {
//				return fmt.Errorf("failed to delete subfolder: %s", err.Error())
//			}
//			klog.Infof("Deleted subfolder: %s\n", subfolderPath)
//		}
//	}
//
//	return nil
//}

func DeleteIfFileExpired(filePath string) error {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %s", err.Error())
	}

	if !fileInfo.IsDir() {
		modTime := fileInfo.ModTime()
		if time.Since(modTime) >= expireTime {
			err := os.Remove(filePath)
			if err != nil {
				return fmt.Errorf("failed to delete file: %s", err.Error())
			}
			klog.Infof("Deleted file: %s\n", filePath)
		} else {
			klog.Infof("File %s is not expired (modified %v ago)\n", filePath, time.Since(modTime))
		}
	} else {
		return fmt.Errorf("provided path is a directory, not a file: %s", filePath)
	}

	return nil
}
