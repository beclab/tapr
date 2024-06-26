// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	aprv1alpha1 "bytetrade.io/web3os/tapr/pkg/apis/apr/v1alpha1"
	versioned "bytetrade.io/web3os/tapr/pkg/generated/clientset/versioned"
	internalinterfaces "bytetrade.io/web3os/tapr/pkg/generated/informers/externalversions/internalinterfaces"
	v1alpha1 "bytetrade.io/web3os/tapr/pkg/generated/listers/apr/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// PGClusterInformer provides access to a shared informer and lister for
// PGClusters.
type PGClusterInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.PGClusterLister
}

type pGClusterInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewPGClusterInformer constructs a new informer for PGCluster type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewPGClusterInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredPGClusterInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredPGClusterInformer constructs a new informer for PGCluster type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredPGClusterInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AprV1alpha1().PGClusters(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AprV1alpha1().PGClusters(namespace).Watch(context.TODO(), options)
			},
		},
		&aprv1alpha1.PGCluster{},
		resyncPeriod,
		indexers,
	)
}

func (f *pGClusterInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredPGClusterInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *pGClusterInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&aprv1alpha1.PGCluster{}, f.defaultInformer)
}

func (f *pGClusterInformer) Lister() v1alpha1.PGClusterLister {
	return v1alpha1.NewPGClusterLister(f.Informer().GetIndexer())
}
