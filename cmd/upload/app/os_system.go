package app

import (
	"context"
	"errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"strings"
	"sync"
)

var PVCs *PVCCache = nil

func minWithNegativeOne(a, b int, aName, bName string) (int, string) {
	if a == -1 && b == -1 {
		return -1, ""
	}

	if a == -1 {
		return b, bName
	}
	if b == -1 {
		return a, aName
	}

	if a < b {
		return a, aName
	} else {
		return b, bName
	}
}

func rewriteUrl(path string, pvc string, prefix string) string {
	if prefix == "" {
		homeIndex := strings.Index(path, "/Home")
		applicationIndex := strings.Index(path, "/Application")
		splitIndex, splitName := minWithNegativeOne(homeIndex, applicationIndex, "/Home", "/Application")
		if splitIndex != -1 {
			firstHalf := path[:splitIndex]
			secondHalf := path[splitIndex:]
			klog.Info("firstHalf=", firstHalf)
			klog.Info("secondHalf=", secondHalf)

			if strings.HasSuffix(firstHalf, pvc) {
				return path
			}
			if splitName == "/Home" {
				return firstHalf + "/" + pvc + secondHalf
			} else {
				secondHalf = strings.TrimPrefix(path[splitIndex:], splitName)
				return firstHalf + "/" + pvc + "/Data" + secondHalf
			}
		}
	} else {
		pathSuffix := strings.TrimPrefix(path, prefix)
		if strings.HasPrefix(pathSuffix, "/"+pvc) {
			return path
		}
		return prefix + "/" + pvc + pathSuffix
	}
	return path
}

func GetAnnotation(ctx context.Context, client *kubernetes.Clientset, key string, bflName string) (string, error) {
	if bflName == "" {
		klog.Error("get Annotation error, bfl-name is empty")
		return "", errors.New("bfl-name is emtpty")
	}

	namespace := "user-space-" + bflName

	bfl, err := client.AppsV1().StatefulSets(namespace).Get(ctx, "bfl", metav1.GetOptions{})
	if err != nil {
		klog.Error("find user's bfl error, ", err, ", ", namespace)
		return "", err
	}

	klog.Infof("bfl.Annotations: %+v", bfl.Annotations)
	return bfl.Annotations[key], nil
}

type PVCCache struct {
	server      *Server
	userPvcMap  map[string]string
	cachePvcMap map[string]string
	mu          sync.Mutex
}

func NewPVCCache(server *Server) *PVCCache {
	return &PVCCache{
		server:      server,
		userPvcMap:  make(map[string]string),
		cachePvcMap: make(map[string]string),
	}
}

func (p *PVCCache) getUserPVCOrCache(bflName string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if val, ok := p.userPvcMap[bflName]; ok {
		return val, nil
	}

	userPvc, err := GetAnnotation(p.server.context, p.server.k8sClient, "userspace_pvc", bflName)
	if err != nil {
		return "", err
	}
	p.userPvcMap[bflName] = userPvc
	return userPvc, nil
}

func (p *PVCCache) getCachePVCOrCache(bflName string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if val, ok := p.cachePvcMap[bflName]; ok {
		return val, nil
	}

	cachePvc, err := GetAnnotation(p.server.context, p.server.k8sClient, "appcache_pvc", bflName)
	if err != nil {
		return "", err
	}
	p.cachePvcMap[bflName] = cachePvc
	return cachePvc, nil
}
