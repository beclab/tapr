package zinc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"bytetrade.io/web3os/tapr/pkg/constants"
	"github.com/emicklei/go-restful"
	"github.com/go-resty/resty/v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

func FindAdminUser(ctx context.Context, client *kubernetes.Clientset) (user, pwd string, err error) {
	var server *appsv1.StatefulSet

RETRY:
	server, err = client.AppsV1().StatefulSets(constants.SystemNamespace).Get(ctx, ZincServerName, metav1.GetOptions{})
	if err != nil {
		klog.Error("find zinc search server error, ", err)
		time.Sleep(5 * time.Second)
		goto RETRY
	}

	var secret *corev1.Secret
	for _, c := range server.Spec.Template.Spec.Containers {
		if c.Name == "zinc-server" {
			user = "admin" // default admin user
			for _, e := range c.Env {
				switch e.Name {
				case "ZINC_FIRST_ADMIN_USER":
					user = e.Value
				case "ZINC_FIRST_ADMIN_PASSWORD":
					if e.ValueFrom != nil && e.ValueFrom.SecretKeyRef != nil {
						secret, err = client.CoreV1().Secrets(constants.SystemNamespace).Get(ctx,
							e.ValueFrom.SecretKeyRef.Name, metav1.GetOptions{})

						if err != nil {
							klog.Error("find zinc admin user password secret error, ", err, ", ", e.ValueFrom.SecretKeyRef.Name)
							return
						}

						if p, ok := secret.Data[e.ValueFrom.SecretKeyRef.Key]; !ok {
							err = errors.New("zinc admin user secret without password")
							return
						} else {
							pwd = string(p)
						}

						return
					}
				} // end  switch
			} // end env loop
		} // end find container
	} // end container loop

	klog.Error("zinc admin user not found")
	err = errors.New("not found")
	return
}

func FindIndexConfig(ctx context.Context, client *kubernetes.Clientset, namespace, config, key string) (schema string, err error) {
	var configMap *corev1.ConfigMap

	configMap, err = client.CoreV1().ConfigMaps(namespace).Get(ctx, config, metav1.GetOptions{})
	if err != nil {
		klog.Error("find index config map error, ", err, ", ", namespace, ",", config, ",", key)
		return
	}

	schema, ok := configMap.Data[key]
	if !ok {
		klog.Error("the key not found in the config map, ", namespace, ",", config, ",", key)
		return
	}

	return
}

func CreateOrUpdateIndex(admin, pwd, namespace, index, schema string) error {
	host := ZincServerService + "." + constants.SystemNamespace
	endpoint := fmt.Sprintf("http://%s/api/index", host)

	client := resty.New().SetTimeout(2 * time.Second)

	mapping := make(map[string]interface{})
	err := json.Unmarshal([]byte(schema), &mapping)
	if err != nil {
		klog.Error("parse index schema error, ", err, ", ", schema)
		return err
	}

	indexNew := &IndexSimple{
		Name:        GetIndexName(namespace, index),
		StorageType: "disk",
		Mappings:    mapping,
	}
	klog.Info("send put to ", endpoint, " to create or update index")

	resp, err := client.R().SetBasicAuth(admin, pwd).
		SetHeader(restful.HEADER_ContentType, restful.MIME_JSON).
		SetBody(indexNew).
		Put(endpoint)

	if err != nil {
		klog.Error("create or update user index error, ", err, ",", index)
		return err
	}

	if resp.StatusCode() >= 400 {
		klog.Error("create or update user index response err, ", resp.StatusCode(), ",", index)
		return err
	}

	klog.Info("create or update index success, ", string(resp.Body()))

	return nil
}

func DeleteIndex(admin, pwd, namespace, index string) error {
	if namespace == "" || index == "" {
		return errors.New("namespace or index is empty")
	}

	host := ZincServerService + "." + constants.SystemNamespace
	endpoint := fmt.Sprintf("http://%s/api/index/%s", host, GetIndexName(namespace, index))

	client := resty.New().SetTimeout(2 * time.Second)
	klog.Info("send delete to ", endpoint, " to create or update index")

	resp, err := client.R().SetBasicAuth(admin, pwd).
		SetHeader(restful.HEADER_ContentType, restful.MIME_JSON).
		Delete(endpoint)

	if err != nil {
		klog.Error("delete user index error, ", err, ",", index)
		return err
	}

	if resp.StatusCode() >= 400 {
		klog.Error("delete user index response err, ", resp.StatusCode(), ",", index)
		return err
	}

	klog.Info("delete index success, ", string(resp.Body()))

	return nil

}

func GetIndexName(namespace, index string) string {
	return fmt.Sprintf("%s_%s", namespace, index)
}

func CreateOrUpdateUser(admin, adminPwd, user, pwd string) error {
	host := ZincServerService + "." + constants.SystemNamespace
	endpoint := fmt.Sprintf("http://%s/api/user", host)

	client := resty.New().SetTimeout(2 * time.Second)
	klog.Info("send post to ", endpoint, " to create or update user")

	userNew := &User{
		ID:       user,
		Name:     user,
		Password: pwd,
		Role:     RoleUser.ID,
	}

	resp, err := client.R().SetBasicAuth(admin, adminPwd).
		SetHeader(restful.HEADER_ContentType, restful.MIME_JSON).
		SetBody(userNew).
		Post(endpoint)

	if err != nil {
		klog.Error("create or update user error, ", err, ",", user)
		return err
	}

	if resp.StatusCode() >= 400 {
		klog.Error("create or update user response err, ", resp.StatusCode(), ",", user)
		return err
	}

	klog.Info("create or update user success, ", string(resp.Body()))

	return nil

}

func DeleteUser(admin, adminPwd, user string) error {
	if user == "" {
		return errors.New("user is empty")
	}

	host := ZincServerService + "." + constants.SystemNamespace
	endpoint := fmt.Sprintf("http://%s/api/user/%s", host, user)

	client := resty.New().SetTimeout(2 * time.Second)
	klog.Info("send delete to ", endpoint, " to delete user")

	resp, err := client.R().SetBasicAuth(admin, adminPwd).
		SetHeader(restful.HEADER_ContentType, restful.MIME_JSON).
		Delete(endpoint)

	if err != nil {
		klog.Error("delete user error, ", err, ",", user)
		return err
	}

	if resp.StatusCode() >= 400 {
		klog.Error("delete user response err, ", resp.StatusCode(), ",", user)
		return err
	}

	klog.Info("delete user success, ", string(resp.Body()))

	return nil

}

func InitRole(admin, adminPwd string) error {
	host := ZincServerService + "." + constants.SystemNamespace
	endpoint := fmt.Sprintf("http://%s/api/role", host)

RETRY:
	restyClient := resty.New().SetTimeout(2 * time.Second)
	klog.Info("send post to ", endpoint, " to create or update role")

	resp, err := restyClient.R().SetBasicAuth(admin, adminPwd).
		SetHeader(restful.HEADER_ContentType, restful.MIME_JSON).
		SetBody(RoleUser).
		Post(endpoint)

	if err != nil {
		klog.Error("create or update role error, ", err, ",", RoleUser)
		time.Sleep(2 * time.Second)
		goto RETRY
	}

	if resp.StatusCode() >= 400 {
		klog.Error("create or update role response err, ", resp.StatusCode(), ",", RoleUser)
		return err
	}

	klog.Info("create or update role success, ", string(resp.Body()))

	return nil

}
