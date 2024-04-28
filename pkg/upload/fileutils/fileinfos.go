package fileutils

import (
	"bytetrade.io/web3os/tapr/pkg/upload/models"
	"bytetrade.io/web3os/tapr/pkg/utils"
	"fmt"
	"github.com/robfig/cron/v3"
	"k8s.io/klog/v2"
	"path/filepath"
	"time"

	//"runtime/debug"
	"sync"
)

type FileInfoMgr struct {
	//todo use storage like boltdb/redis
	InfoMap map[string]*models.FileInfo
	mu      sync.RWMutex
}

func NewFileInfoMgr() *FileInfoMgr {
	return &FileInfoMgr{InfoMap: make(map[string]*models.FileInfo)}
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
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, v := range m.InfoMap {
		if time.Since(v.LastUpdateTime) > expireTime {
			delete(m.InfoMap, k)
		}
	}
}

func (m *FileInfoMgr) AddFileInfo(id string, info *models.FileInfo) error {
	exist, _ := m.ExistFileInfo(id)
	if exist {
		return fmt.Errorf("id %s already exist", id)
	}

	info.LastUpdateTime = time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.InfoMap[id] = info

	return nil
}

func (m *FileInfoMgr) UpdateInfo(id string, info *models.FileInfo) {
	info.LastUpdateTime = time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.InfoMap[id] = info
}

func (m *FileInfoMgr) DelFileInfo(id string) {
	exist, _ := m.ExistFileInfo(id)
	if !exist {
		klog.Warningf("id %s not exist in map", id)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.InfoMap, id)
}

func (m *FileInfoMgr) ExistFileInfo(id string) (bool, *models.FileInfo) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if info, exist := m.InfoMap[id]; exist {
		//debug.PrintStack()
		klog.Infof("id %s exist in map, info:%+v", id, info)
		return exist, info
	}

	return false, nil
}

func (m *FileInfoMgr) CheckTempFile(id string) (bool, int64) {
	return utils.PathExistsAndGetLen(filepath.Join(UploadsDir, id))
}
