package middlewarerequest

import (
	"context"
	"crypto/tls"
	"fmt"
	"k8s.io/klog/v2"
	"net/http"
	"strings"

	aprv1 "bytetrade.io/web3os/tapr/pkg/apis/apr/v1alpha1"
	wes "bytetrade.io/web3os/tapr/pkg/workload/elasticsearch"

	elastic "github.com/elastic/go-elasticsearch/v8"
	esapi "github.com/elastic/go-elasticsearch/v8/esapi"
)

const elasticNamespace = "elasticsearch-middleware"

func (c *controller) createOrUpdateElasticsearchRequest(req *aprv1.MiddlewareRequest) error {
	adminUser, adminPassword, err := wes.FindElasticsearchAdminUser(c.ctx, c.k8sClientSet, elasticNamespace)
	if err != nil {
		klog.Errorf("failed to get elastic admin user %v", err)
		return err
	}

	endpoint := c.getElasticsearchEndpoint()
	klog.Infof("req.Spec.Elasticsearch %#v", req.Spec.Elasticsearch)
	klog.Infof("req.Spec.Elasticsearch.Password %#v", req.Spec.Elasticsearch.Password)

	userPassword, err := req.Spec.Elasticsearch.Password.GetVarValue(c.ctx, c.k8sClientSet, req.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get user password %v", err)
	}

	es, err := newESClient(endpoint, adminUser, adminPassword)
	if err != nil {
		return fmt.Errorf("failed to new esclient %v", err)
	}

	err = esPutUser(es, req.Spec.Elasticsearch.User, userPassword)
	if err != nil {
		return fmt.Errorf("failed to put user %s %v", req.Spec.Elasticsearch.User, err)
	}

	// Create indices and grant permissions via role
	var indices []string
	for _, idx := range req.Spec.Elasticsearch.Indexes {
		name := wes.GetIndexName(req.Spec.AppNamespace, idx.Name)
		indices = append(indices, name)
		err = esCreateOrUpdateIndex(es, name)
		if err != nil {
			return fmt.Errorf("failed to create index %s %v", name, err)
		}
	}
	roleName := fmt.Sprintf("role-%s", req.Spec.Elasticsearch.User)
	err = esPutRole(es, roleName, indices)
	if err != nil {
		return err
	}
	err = esPutUserRole(es, req.Spec.Elasticsearch.User, roleName)
	if err != nil {
		return err
	}
	return nil
}

func (c *controller) deleteElasticsearchRequest(req *aprv1.MiddlewareRequest) error {
	adminUser, adminPassword, err := wes.FindElasticsearchAdminUser(c.ctx, c.k8sClientSet, elasticNamespace)
	if err != nil {
		klog.Errorf("failed to find admin user %v", err)
		return err
	}
	endpoint := c.getElasticsearchEndpoint()
	es, err := newESClient(endpoint, adminUser, adminPassword)
	if err != nil {
		return fmt.Errorf("failed to new esclient %v", err)
	}
	roleName := fmt.Sprintf("role-%s", req.Spec.Elasticsearch.User)
	err = esDeleteUser(es, req.Spec.Elasticsearch.User)
	if err != nil {
		return fmt.Errorf("failed to delete user %s %v", req.Spec.Elasticsearch.User, err)
	}
	err = esDeleteRole(es, roleName)
	if err != nil {
		return fmt.Errorf("failed to delete role %s %v", roleName, err)
	}
	for _, idx := range req.Spec.Elasticsearch.Indexes {
		err = esDeleteIndex(es, wes.GetIndexName(req.Spec.AppNamespace, idx.Name))
		if err != nil {
			return fmt.Errorf("failed to delete index %v", err)
		}
	}
	return nil
}

func (c *controller) getElasticsearchEndpoint() string {
	return fmt.Sprintf("https://elasticsearch-master-http.%s:9200", elasticNamespace)
}

func newESClient(endpoint, username, password string) (*elastic.Client, error) {
	cfg := elastic.Config{
		Addresses: []string{endpoint},
		Username:  username,
		Password:  password,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	return elastic.NewClient(cfg)
}

func esCreateOrUpdateIndex(es *elastic.Client, index string) error {
	exists, err := checkIndexIfExists(es, index)
	if err != nil {
		return err
	}
	if exists {
		klog.Errorf("index %s already exists", index)
		return nil
	}
	req := esapi.IndicesCreateRequest{Index: index}
	res, err := req.Do(context.Background(), es)
	if err != nil {
		return err
	}
	if res.IsError() {
		return fmt.Errorf("create index %s failed: %s", index, res.String())
	}
	return nil
}

func checkIndexIfExists(es *elastic.Client, index string) (bool, error) {
	req := esapi.IndicesExistsRequest{Index: []string{index}}
	res, err := req.Do(context.Background(), es)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		return true, nil
	}
	if res.IsError() && res.StatusCode != http.StatusNotFound {
		return false, fmt.Errorf("check index %s exists failed: %s", index, res.String())
	}
	return false, nil
}

func esDeleteIndex(es *elastic.Client, index string) error {
	req := esapi.IndicesDeleteRequest{Index: []string{index}}
	res, err := req.Do(context.Background(), es)
	if err != nil {
		return err
	}
	if res.StatusCode == http.StatusNotFound {
		return nil
	}
	if res.IsError() {
		return fmt.Errorf("delete index %s failed: %s", index, res.String())
	}
	return nil
}

func esPutRole(es *elastic.Client, role string, indices []string) error {
	body := fmt.Sprintf(`{"indices":[{"names":%q,"privileges":["all"]}]}`, indices)
	req := esapi.SecurityPutRoleRequest{Name: role, Body: strings.NewReader(body)}
	res, err := req.Do(context.Background(), es)
	if err != nil {
		return err
	}
	if res.IsError() {
		return fmt.Errorf("put role failed: %s", res.String())
	}
	return nil
}

func esDeleteRole(es *elastic.Client, role string) error {
	req := esapi.SecurityDeleteRoleRequest{Name: role}
	res, err := req.Do(context.Background(), es)
	if err != nil {
		return err
	}
	if res.StatusCode == http.StatusNotFound {
		return nil
	}
	if res.IsError() {
		return fmt.Errorf("delete role %s failed: %s", role, res.String())
	}
	return nil
}

func esPutUser(es *elastic.Client, user, password string) error {
	body := fmt.Sprintf(`{"password":"%s","roles":[]}`, password)
	req := esapi.SecurityPutUserRequest{Username: user, Body: strings.NewReader(body)}
	res, err := req.Do(context.Background(), es)
	if err != nil {
		return err
	}
	if res.IsError() {
		return fmt.Errorf("put user failed: %s", res.String())
	}
	return nil
}

func esDeleteUser(es *elastic.Client, user string) error {
	req := esapi.SecurityDeleteUserRequest{Username: user}
	res, err := req.Do(context.Background(), es)
	if err != nil {
		return fmt.Errorf("failed to send delete uer req %v", err)
	}
	if res.StatusCode == http.StatusNotFound {
		return nil
	}
	if res.IsError() {
		return fmt.Errorf("delete user failed: %s", res.String())
	}
	return nil
}

func esPutUserRole(es *elastic.Client, user, role string) error {
	body := fmt.Sprintf(`{"roles":[%q]}`, role)
	req := esapi.SecurityPutUserRequest{Username: user, Body: strings.NewReader(body)}
	res, err := req.Do(context.Background(), es)
	if err != nil {
		return err
	}
	if res.IsError() {
		return fmt.Errorf("assign role failed: %s", res.String())
	}
	return nil
}
