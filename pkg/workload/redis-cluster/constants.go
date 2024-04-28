package rediscluster

import "k8s.io/apimachinery/pkg/runtime/schema"

const (
	RedisClusterService     = "redis-cluster-proxy"
	RedisclusterProxyConfig = "predixy-configs"
	RedisProxy              = "redis-cluster-proxy"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: "redis.kun", Version: "v1alpha1"}

	RedisClusterClassGVR = schema.GroupVersionResource{
		Group:    SchemeGroupVersion.Group,
		Version:  SchemeGroupVersion.Version,
		Resource: "distributedredisclusters",
	}

	RedisClusterBackupGVR = schema.GroupVersionResource{
		Group:    SchemeGroupVersion.Group,
		Version:  SchemeGroupVersion.Version,
		Resource: "redisclusterbackups",
	}
)

const (
	REDISCLUSTER_NAMESPACE = "os-system"
	REDISCLUSTER_NAME      = "redis-cluster"
)
