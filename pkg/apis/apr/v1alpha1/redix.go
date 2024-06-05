package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +kubebuilder:printcolumn:name="type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced, shortName={rdxc}, categories={all}
// RedixCluster is the Schema for the Redis-Compatible Cluster
type RedixCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RedixClusterSpec   `json:"spec,omitempty"`
	Status RedixClusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RedixClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []RedixCluster `json:"items"`
}

type RedixClusterSpec struct {
	Type    RedixType    `json:"type"`
	KVRocks *KVRocksSpec `json:"kvrocks,omitempty"`
}

type RedixClusterStatus struct {
	// the state of the application: draft, submitted, passed, rejected, suspended, active
	State      string       `json:"state"`
	UpdateTime *metav1.Time `json:"updateTime,omitempty"`
	StatusTime *metav1.Time `json:"statusTime,omitempty"`
}

type KVRocksSpec struct {
	Password      PasswordVar       `json:"password,omitempty"`
	Owner         string            `json:"owner"`
	BackupStorage *string           `json:"backupStorage,omitempty"`
	Config        map[string]string `json:"config"`
}

type RedixType string

const (
	RedisCluster   RedixType = "redis-cluster"
	RedisServer    RedixType = "redis-server"
	KVRocks        RedixType = "kvrocks"
	KVRocksCluster RedixType = "kvrocks-cluster"
)
