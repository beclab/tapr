package app

import aprv1 "bytetrade.io/web3os/tapr/pkg/apis/apr/v1alpha1"

type MiddlewareReq struct {
	App          string               `json:"app"`
	AppNamespace string               `json:"appNamespace"`
	Namespace    string               `json:"namespace"`
	Middleware   aprv1.MiddlewareType `json:"middleware"`
}

type Database struct {
	Name        string `json:"name"`
	Distributed bool   `json:"distributed,omitempty"`
}

type MetaInfo struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

type MiddlewareRequestInfo struct {
	MetaInfo
	App       MetaInfo             `json:"app"`
	UserName  string               `json:"username,omitempty"`
	Password  string               `json:"password"`
	Type      aprv1.MiddlewareType `json:"type"`
	Databases []Database           `json:"databases,omitempty"`
}

type MiddlewareRequestResp struct {
	MiddlewareRequestInfo
	Host      string            `json:"host"`
	Port      int32             `json:"port"`
	Indexes   map[string]string `json:"indexes"`
	Databases map[string]string `json:"databases"`
	Subjects  map[string]string `json:"subjects"`
	Refs      map[string]string `json:"refs"`
}

type Proxy struct {
	Endpoint string `json:"endpoint"`
	Size     int32  `json:"size"`
}

type MiddlewareClusterResp struct {
	MetaInfo
	Nodes      int32  `json:"nodes"`
	AdminUser  string `json:"adminUser"`
	Password   string `json:"password"`
	Mongos     Proxy  `json:"mongos,omitempty"`
	RedisProxy Proxy  `json:"redisProxy,omitempty"`
}

type ClusterScaleReq struct {
	MetaInfo
	Middleware aprv1.MiddlewareType `json:"middleware"`
	Nodes      int32                `json:"nodes"`
}

type ClusterChangePwdReq struct {
	MetaInfo
	Middleware aprv1.MiddlewareType `json:"middleware"`

	User     string `json:"user,omitempty"`
	Password string `json:"password"`
}
