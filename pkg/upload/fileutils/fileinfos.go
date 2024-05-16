package fileutils

import (
	"bytetrade.io/web3os/tapr/pkg/upload/models"
	"bytetrade.io/web3os/tapr/pkg/utils"
	"fmt"
	"github.com/robfig/cron/v3"
	"k8s.io/klog/v2"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"
)

// todo use storage like boltdb/redis
var (
	InfoSyncMap sync.Map
)

type FileInfoMgr struct {
}

func NewFileInfoMgr() *FileInfoMgr {
	return &FileInfoMgr{}
}

func (m *FileInfoMgr) Init() {
	m.cronDeleteOldInfo()
}

func (m *FileInfoMgr) cronDeleteOldInfo() {
	c := cron.New()

	_, err := c.AddFunc("30 * * * *", func() {
		m.DeleteOldInfos()
	})
	if err != nil {
		klog.Warningf("AddFunc DeleteOldInfos err:%v", err)
	}

	c.Start()
}

func (m *FileInfoMgr) DeleteOldInfos() {
	InfoSyncMap.Range(func(key, value interface{}) bool {
		v := value.(models.FileInfo)
		klog.Infof("Key: %v, Value: %v\n", key, v)
		if time.Since(v.LastUpdateTime) > expireTime {
			klog.Infof("id %s expire del in map, stack:%s", key, debug.Stack())
			InfoSyncMap.Delete(key)
			RemoveTempFileAndInfoFile(key.(string))
		}
		return true
	})
}

func (m *FileInfoMgr) AddFileInfo(id string, info models.FileInfo) error {
	if id != info.ID {
		klog.Errorf("id:%s diff from v:%v", id, info)
		return fmt.Errorf("id:%s diff from v:%v", id, info)
	}

	info.LastUpdateTime = time.Now()
	InfoSyncMap.Store(id, info)

	return nil
}

func debugMap() {
	InfoSyncMap.Range(func(key, value interface{}) bool {
		v := value.(models.FileInfo)
		klog.Infof("Key: %v, Value: %v\n", key, v)
		if key != v.ID {
			klog.Errorf("k:%s different from v:%v stack:%s", key, v, debug.Stack())
		}
		return true
	})
}

func (m *FileInfoMgr) UpdateInfo(id string, info models.FileInfo) {
	if id != info.ID {
		klog.Errorf("id:%s diff from v:%v", id, info)
		return
	}

	info.LastUpdateTime = time.Now()
	InfoSyncMap.Store(id, info)
}

func (m *FileInfoMgr) DelFileInfo(id string) {
	InfoSyncMap.Delete(id)
	RemoveTempFileAndInfoFile(id)
}

func (m *FileInfoMgr) ExistFileInfo(id string) (bool, models.FileInfo) {
	value, ok := InfoSyncMap.Load(id)
	if ok {
		return ok, value.(models.FileInfo)
	}

	return false, models.FileInfo{}
}

func (m *FileInfoMgr) CheckTempFile(id string) (bool, int64) {
	return utils.PathExistsAndGetLen(filepath.Join(UploadsDir, id))
}
