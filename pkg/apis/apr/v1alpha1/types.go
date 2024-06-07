package v1alpha1

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +kubebuilder:printcolumn:name="replicas",type=number,JSONPath=`.spec.replicas`
// +kubebuilder:printcolumn:name="admin",type=string,JSONPath=`.spec.adminUser`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced, shortName={pgc}, categories={all}
// PGCluster is the Schema for the PostgreSQL Cluster
type PGCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PGClusterSpec   `json:"spec,omitempty"`
	Status PGClusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PGClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []PGCluster `json:"items"`
}

type PGClusterStatus struct {
	// the state of the application: draft, submitted, passed, rejected, suspended, active
	State      string       `json:"state"`
	UpdateTime *metav1.Time `json:"updateTime,omitempty"`
	StatusTime *metav1.Time `json:"statusTime,omitempty"`
}

type PGClusterSpec struct {
	// +kubebuilder:validation:Minimum=1
	Replicas int32 `json:"replicas"`

	// +kubebuilder:validation:Pattern=`^([a-zA-Z0-9_]*)$`
	AdminUser  string `json:"adminUser,omitempty"`
	CitusImage string `json:"citusImage,omitempty"`

	Password      PasswordVar `json:"password,omitempty"`
	Owner         string      `json:"owner"`
	BackupStorage string      `json:"backupStorage,omitempty"`
}

type PasswordVar struct {
	// Defaults to "".
	// +optional
	Value string `json:"value,omitempty" protobuf:"bytes,2,opt,name=value"`

	// Source for the environment variable's value. Cannot be used if value is not empty.
	// +optional
	ValueFrom *PasswordVarSource `json:"valueFrom,omitempty" protobuf:"bytes,3,opt,name=valueFrom"`
}

type PasswordVarSource struct {
	// Selects a key of a secret in the pod's namespace
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef,omitempty" protobuf:"bytes,4,opt,name=secretKeyRef"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +kubebuilder:printcolumn:name="middleware",type=number,JSONPath=`.spec.middleware`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced, shortName={mr}, categories={all}
// MiddlewareRequest is the Schema for the application Middleware Request
type MiddlewareRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MiddlewareSpec   `json:"spec,omitempty"`
	Status MiddlewareStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MiddlewareRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []MiddlewareRequest `json:"items"`
}

type MiddlewareStatus struct {
	// the state of the application: draft, submitted, passed, rejected, suspended, active
	State      string       `json:"state"`
	UpdateTime *metav1.Time `json:"updateTime,omitempty"`
	StatusTime *metav1.Time `json:"statusTime,omitempty"`
}

type MiddlewareSpec struct {
	App          string         `json:"app"`
	AppNamespace string         `json:"appNamespace"`
	Middleware   MiddlewareType `json:"middleware"`

	// +optional
	Redis Redis `json:"redis,omitempty"`

	// +optional
	MongoDB MongoDB `json:"mongodb,omitempty"`

	// +optional
	PostgreSQL PostgreSQL `json:"postgreSQL,omitempty"`

	// +optional
	Zinc Zinc `json:"zinc,omitempty"`
}

type Redis struct {
	Password  PasswordVar `json:"password,omitempty"`
	Namespace string      `json:"namespace"`
}

type MongoDB struct {
	Password  PasswordVar     `json:"password,omitempty"`
	Databases []MongoDatabase `json:"databases"`
	User      string          `json:"user"`
}

type PostgreSQL struct {
	Password  PasswordVar     `json:"password,omitempty"`
	Databases []CitusDatabase `json:"databases"`

	// +kubebuilder:validation:Pattern=`^([a-zA-Z0-9_]*)$`
	User string `json:"user"`
}

type Zinc struct {
	User     string             `json:"user"`
	Password PasswordVar        `json:"password,omitempty"`
	Indexes  []*ZincIndexConfig `json:"indexes"`
}

type CitusDatabase struct {
	Name       string   `json:"name"`
	Extensions []string `json:"extensions,omitempty"`
	Scripts    []string `json:"scripts,omitempty"`
	// +optional
	Distributed *bool `json:"distributed"`
}

type MongoDatabase struct {
	Name    string   `json:"name"`
	Scripts []string `json:"scripts,omitempty"`
}

type ZincIndexConfig struct {
	corev1.ConfigMapKeySelector `json:",inline"`
	Namespace                   string `json:"namespace"`
}

type MiddlewareType string

const (
	TypePostgreSQL MiddlewareType = "postgres"
	TypeMongoDB    MiddlewareType = "mongodb"
	TypeRedis      MiddlewareType = "redis"
	TypeZinc       MiddlewareType = "zinc"
)

func (c *CitusDatabase) IsDistributed() bool { return c.Distributed != nil && *c.Distributed }

func (p *PasswordVar) GetVarValue(ctx context.Context, client *kubernetes.Clientset, namespace string) (string, error) {
	if p.Value != "" {
		return p.Value, nil
	}

	if p.ValueFrom == nil {
		return "", errors.New("password is not defined")
	}

	secret, err := client.CoreV1().Secrets(namespace).Get(ctx, p.ValueFrom.SecretKeyRef.Name, metav1.GetOptions{})
	if err != nil {
		klog.Error("get password secret ref error, ", err, ", ", p.ValueFrom.SecretKeyRef.Name)
		return "", err
	}

	return string(secret.Data[p.ValueFrom.SecretKeyRef.Key]), nil
}
