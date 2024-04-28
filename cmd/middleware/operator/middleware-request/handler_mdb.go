package middlewarerequest

import (
	"errors"

	psmdbv1 "github.com/percona/percona-server-mongodb-operator/pkg/apis/psmdb/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"

	aprv1 "bytetrade.io/web3os/tapr/pkg/apis/apr/v1alpha1"
	"bytetrade.io/web3os/tapr/pkg/mongo"
	"bytetrade.io/web3os/tapr/pkg/workload/percona"
)

func (c *controller) createOrUpdateMDBRequest(req *aprv1.MiddlewareRequest) error {
	pwd, err := req.Spec.MongoDB.Password.GetVarValue(c.ctx, c.k8sClientSet, req.Namespace)
	if err != nil {
		return err
	}

	client, err := c.connectToCluster(req)
	if err != nil {
		return err
	}
	defer client.Close(c.ctx)

	return client.CreateOrUpdateUserWithDatabase(c.ctx, req.Spec.MongoDB.User, pwd, dbRealNames(req.Spec.AppNamespace, req.Spec.MongoDB.Databases))
}

func (c *controller) deleteMDBRequest(req *aprv1.MiddlewareRequest) error {
	client, err := c.connectToCluster(req)
	if err != nil {
		return err
	}
	defer client.Close(c.ctx)

	return client.DropUserAndDatabase(c.ctx, req.Spec.MongoDB.User, dbRealNames(req.Spec.AppNamespace, req.Spec.MongoDB.Databases))
}

func (c *controller) connectToCluster(req *aprv1.MiddlewareRequest) (*mongo.MongoClient, error) {
	host, err := c.getMongoClusterHost(req)
	if err != nil {
		return nil, err
	}

	user, pwd, err := c.getMongoClusterAdminUser(req)
	if err != nil {
		return nil, err
	}

	client := &mongo.MongoClient{
		User:     user,
		Password: pwd,
		Addr:     host + ":27017",
	}

	err = client.Connect(c.ctx)
	if err != nil {
		klog.Error("connect mongodb error, ", err, ", ", host)
		return nil, err
	}

	return client, nil
}

func (c *controller) getMongoClusterHost(req *aprv1.MiddlewareRequest) (string, error) {
	psmdb := psmdbv1.PerconaServerMongoDB{}
	resource, err := c.dynamicClient.Resource(percona.PSMDBClassGVR).Namespace(percona.PSMDB_NAMESPACE).Get(c.ctx, percona.PSMDB_NAME, metav1.GetOptions{})
	if err != nil {
		klog.Error("find user mongo cluster error, ", err, req)
		return "", err
	}

	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Object, &psmdb); err != nil {
		klog.Error("parse PerconaServerMongoDB error, ", err)
		return "", err
	}

	if psmdb.Status.Host == "" {
		return "", errors.New("cluster is not running")
	}

	return psmdb.Status.Host, nil
}

func (c *controller) getMongoClusterAdminUser(req *aprv1.MiddlewareRequest) (user, password string, err error) {
	secret, err := c.k8sClientSet.CoreV1().Secrets(percona.PSMDB_NAMESPACE).Get(c.ctx, percona.PSMDB_SECRET, metav1.GetOptions{})
	if err != nil {
		klog.Error("find mongo cluster admin user error, ", err)
		return
	}

	user = string(secret.Data[percona.PSMDB_ADMIN_KEY])
	password = string(secret.Data[percona.PSMDB_ADMIN_PASSWORD_KEY])

	return
}

func dbRealNames(namespace string, dbs []string) []string {
	var ret []string

	for _, db := range dbs {
		ret = append(ret, percona.GetDatabaseName(namespace, db))
	}

	return ret
}
