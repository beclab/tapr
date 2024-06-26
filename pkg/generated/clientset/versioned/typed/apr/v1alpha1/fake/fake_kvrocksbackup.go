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

// FakeKVRocksBackups implements KVRocksBackupInterface
type FakeKVRocksBackups struct {
	Fake *FakeAprV1alpha1
	ns   string
}

var kvrocksbackupsResource = v1alpha1.SchemeGroupVersion.WithResource("kvrocksbackups")

var kvrocksbackupsKind = v1alpha1.SchemeGroupVersion.WithKind("KVRocksBackup")

// Get takes name of the kVRocksBackup, and returns the corresponding kVRocksBackup object, and an error if there is any.
func (c *FakeKVRocksBackups) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.KVRocksBackup, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(kvrocksbackupsResource, c.ns, name), &v1alpha1.KVRocksBackup{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.KVRocksBackup), err
}

// List takes label and field selectors, and returns the list of KVRocksBackups that match those selectors.
func (c *FakeKVRocksBackups) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.KVRocksBackupList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(kvrocksbackupsResource, kvrocksbackupsKind, c.ns, opts), &v1alpha1.KVRocksBackupList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.KVRocksBackupList{ListMeta: obj.(*v1alpha1.KVRocksBackupList).ListMeta}
	for _, item := range obj.(*v1alpha1.KVRocksBackupList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested kVRocksBackups.
func (c *FakeKVRocksBackups) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(kvrocksbackupsResource, c.ns, opts))

}

// Create takes the representation of a kVRocksBackup and creates it.  Returns the server's representation of the kVRocksBackup, and an error, if there is any.
func (c *FakeKVRocksBackups) Create(ctx context.Context, kVRocksBackup *v1alpha1.KVRocksBackup, opts v1.CreateOptions) (result *v1alpha1.KVRocksBackup, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(kvrocksbackupsResource, c.ns, kVRocksBackup), &v1alpha1.KVRocksBackup{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.KVRocksBackup), err
}

// Update takes the representation of a kVRocksBackup and updates it. Returns the server's representation of the kVRocksBackup, and an error, if there is any.
func (c *FakeKVRocksBackups) Update(ctx context.Context, kVRocksBackup *v1alpha1.KVRocksBackup, opts v1.UpdateOptions) (result *v1alpha1.KVRocksBackup, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(kvrocksbackupsResource, c.ns, kVRocksBackup), &v1alpha1.KVRocksBackup{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.KVRocksBackup), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeKVRocksBackups) UpdateStatus(ctx context.Context, kVRocksBackup *v1alpha1.KVRocksBackup, opts v1.UpdateOptions) (*v1alpha1.KVRocksBackup, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(kvrocksbackupsResource, "status", c.ns, kVRocksBackup), &v1alpha1.KVRocksBackup{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.KVRocksBackup), err
}

// Delete takes name of the kVRocksBackup and deletes it. Returns an error if one occurs.
func (c *FakeKVRocksBackups) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(kvrocksbackupsResource, c.ns, name, opts), &v1alpha1.KVRocksBackup{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeKVRocksBackups) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(kvrocksbackupsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.KVRocksBackupList{})
	return err
}

// Patch applies the patch and returns the patched kVRocksBackup.
func (c *FakeKVRocksBackups) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.KVRocksBackup, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(kvrocksbackupsResource, c.ns, name, pt, data, subresources...), &v1alpha1.KVRocksBackup{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.KVRocksBackup), err
}
