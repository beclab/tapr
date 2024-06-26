// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "bytetrade.io/web3os/tapr/pkg/apis/apr/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// RedixClusterLister helps list RedixClusters.
// All objects returned here must be treated as read-only.
type RedixClusterLister interface {
	// List lists all RedixClusters in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.RedixCluster, err error)
	// RedixClusters returns an object that can list and get RedixClusters.
	RedixClusters(namespace string) RedixClusterNamespaceLister
	RedixClusterListerExpansion
}

// redixClusterLister implements the RedixClusterLister interface.
type redixClusterLister struct {
	indexer cache.Indexer
}

// NewRedixClusterLister returns a new RedixClusterLister.
func NewRedixClusterLister(indexer cache.Indexer) RedixClusterLister {
	return &redixClusterLister{indexer: indexer}
}

// List lists all RedixClusters in the indexer.
func (s *redixClusterLister) List(selector labels.Selector) (ret []*v1alpha1.RedixCluster, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.RedixCluster))
	})
	return ret, err
}

// RedixClusters returns an object that can list and get RedixClusters.
func (s *redixClusterLister) RedixClusters(namespace string) RedixClusterNamespaceLister {
	return redixClusterNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// RedixClusterNamespaceLister helps list and get RedixClusters.
// All objects returned here must be treated as read-only.
type RedixClusterNamespaceLister interface {
	// List lists all RedixClusters in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.RedixCluster, err error)
	// Get retrieves the RedixCluster from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.RedixCluster, error)
	RedixClusterNamespaceListerExpansion
}

// redixClusterNamespaceLister implements the RedixClusterNamespaceLister
// interface.
type redixClusterNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all RedixClusters in the indexer for a given namespace.
func (s redixClusterNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.RedixCluster, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.RedixCluster))
	})
	return ret, err
}

// Get retrieves the RedixCluster from the indexer for a given namespace and name.
func (s redixClusterNamespaceLister) Get(name string) (*v1alpha1.RedixCluster, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("redixcluster"), name)
	}
	return obj.(*v1alpha1.RedixCluster), nil
}
