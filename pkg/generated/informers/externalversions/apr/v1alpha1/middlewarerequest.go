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

// MiddlewareRequestInformer provides access to a shared informer and lister for
// MiddlewareRequests.
type MiddlewareRequestInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.MiddlewareRequestLister
}

type middlewareRequestInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewMiddlewareRequestInformer constructs a new informer for MiddlewareRequest type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewMiddlewareRequestInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredMiddlewareRequestInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredMiddlewareRequestInformer constructs a new informer for MiddlewareRequest type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredMiddlewareRequestInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AprV1alpha1().MiddlewareRequests(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AprV1alpha1().MiddlewareRequests(namespace).Watch(context.TODO(), options)
			},
		},
		&aprv1alpha1.MiddlewareRequest{},
		resyncPeriod,
		indexers,
	)
}

func (f *middlewareRequestInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredMiddlewareRequestInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *middlewareRequestInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&aprv1alpha1.MiddlewareRequest{}, f.defaultInformer)
}

func (f *middlewareRequestInformer) Lister() v1alpha1.MiddlewareRequestLister {
	return v1alpha1.NewMiddlewareRequestLister(f.Informer().GetIndexer())
}
