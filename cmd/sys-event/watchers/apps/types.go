package apps

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	GVR = schema.GroupVersionResource{
		Group: "app.bytetrade.io", Version: "v1alpha1", Resource: "applications",
	}
)

type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSpec   `json:"spec,omitempty"`
	Status ApplicationStatus `json:"status,omitempty"`
}

type ApplicationSpec struct {
	// the entrance of the application
	Index string `json:"index,omitempty"`

	// description from app's description or frontend
	Description string `json:"description,omitempty"`

	// The url of the icon
	Icon string `json:"icon,omitempty"`

	// the name of the application
	Name string `json:"name"`

	// the unique id of the application
	// for sys application appid equal name otherwise appid equal md5(name)[:8]
	Appid string `json:"appid"`

	IsSysApp bool `json:"isSysApp"`

	// the namespace of the application
	Namespace string `json:"namespace,omitempty"`

	// the deployment of the application
	DeploymentName string `json:"deployment,omitempty"`

	// the owner of the application
	Owner string `json:"owner,omitempty"`

	// the service address of the application
	ServiceAddr string `json:"service,omitempty"`

	// the extend settings of the application
	Settings map[string]string `json:"settings,omitempty"`
}

// ApplicationStatus defines the observed state of Application
type ApplicationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// the state of the application: draft, submitted, passed, rejected, suspended, active
	State      string       `json:"state,omitempty"`
	UpdateTime *metav1.Time `json:"updateTime"`
	StatusTime *metav1.Time `json:"statusTime"`
}
