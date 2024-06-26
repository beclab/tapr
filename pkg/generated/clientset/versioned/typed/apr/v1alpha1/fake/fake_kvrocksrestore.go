// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "bytetrade.io/web3os/tapr/pkg/apis/apr/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeKVRocksRestores implements KVRocksRestoreInterface
type FakeKVRocksRestores struct {
	Fake *FakeAprV1alpha1
	ns   string
}

var kvrocksrestoresResource = v1alpha1.SchemeGroupVersion.WithResource("kvrocksrestores")

var kvrocksrestoresKind = v1alpha1.SchemeGroupVersion.WithKind("KVRocksRestore")

// Get takes name of the kVRocksRestore, and returns the corresponding kVRocksRestore object, and an error if there is any.
func (c *FakeKVRocksRestores) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.KVRocksRestore, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(kvrocksrestoresResource, c.ns, name), &v1alpha1.KVRocksRestore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.KVRocksRestore), err
}

// List takes label and field selectors, and returns the list of KVRocksRestores that match those selectors.
func (c *FakeKVRocksRestores) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.KVRocksRestoreList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(kvrocksrestoresResource, kvrocksrestoresKind, c.ns, opts), &v1alpha1.KVRocksRestoreList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.KVRocksRestoreList{ListMeta: obj.(*v1alpha1.KVRocksRestoreList).ListMeta}
	for _, item := range obj.(*v1alpha1.KVRocksRestoreList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested kVRocksRestores.
func (c *FakeKVRocksRestores) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(kvrocksrestoresResource, c.ns, opts))

}

// Create takes the representation of a kVRocksRestore and creates it.  Returns the server's representation of the kVRocksRestore, and an error, if there is any.
func (c *FakeKVRocksRestores) Create(ctx context.Context, kVRocksRestore *v1alpha1.KVRocksRestore, opts v1.CreateOptions) (result *v1alpha1.KVRocksRestore, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(kvrocksrestoresResource, c.ns, kVRocksRestore), &v1alpha1.KVRocksRestore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.KVRocksRestore), err
}

// Update takes the representation of a kVRocksRestore and updates it. Returns the server's representation of the kVRocksRestore, and an error, if there is any.
func (c *FakeKVRocksRestores) Update(ctx context.Context, kVRocksRestore *v1alpha1.KVRocksRestore, opts v1.UpdateOptions) (result *v1alpha1.KVRocksRestore, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(kvrocksrestoresResource, c.ns, kVRocksRestore), &v1alpha1.KVRocksRestore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.KVRocksRestore), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeKVRocksRestores) UpdateStatus(ctx context.Context, kVRocksRestore *v1alpha1.KVRocksRestore, opts v1.UpdateOptions) (*v1alpha1.KVRocksRestore, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(kvrocksrestoresResource, "status", c.ns, kVRocksRestore), &v1alpha1.KVRocksRestore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.KVRocksRestore), err
}

// Delete takes name of the kVRocksRestore and deletes it. Returns an error if one occurs.
func (c *FakeKVRocksRestores) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(kvrocksrestoresResource, c.ns, name, opts), &v1alpha1.KVRocksRestore{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeKVRocksRestores) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(kvrocksrestoresResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.KVRocksRestoreList{})
	return err
}

// Patch applies the patch and returns the patched kVRocksRestore.
func (c *FakeKVRocksRestores) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.KVRocksRestore, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(kvrocksrestoresResource, c.ns, name, pt, data, subresources...), &v1alpha1.KVRocksRestore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.KVRocksRestore), err
}
