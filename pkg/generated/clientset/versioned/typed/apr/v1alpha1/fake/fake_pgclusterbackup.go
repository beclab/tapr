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

// FakePGClusterBackups implements PGClusterBackupInterface
type FakePGClusterBackups struct {
	Fake *FakeAprV1alpha1
	ns   string
}

var pgclusterbackupsResource = v1alpha1.SchemeGroupVersion.WithResource("pgclusterbackups")

var pgclusterbackupsKind = v1alpha1.SchemeGroupVersion.WithKind("PGClusterBackup")

// Get takes name of the pGClusterBackup, and returns the corresponding pGClusterBackup object, and an error if there is any.
func (c *FakePGClusterBackups) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.PGClusterBackup, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(pgclusterbackupsResource, c.ns, name), &v1alpha1.PGClusterBackup{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PGClusterBackup), err
}

// List takes label and field selectors, and returns the list of PGClusterBackups that match those selectors.
func (c *FakePGClusterBackups) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.PGClusterBackupList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(pgclusterbackupsResource, pgclusterbackupsKind, c.ns, opts), &v1alpha1.PGClusterBackupList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.PGClusterBackupList{ListMeta: obj.(*v1alpha1.PGClusterBackupList).ListMeta}
	for _, item := range obj.(*v1alpha1.PGClusterBackupList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested pGClusterBackups.
func (c *FakePGClusterBackups) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(pgclusterbackupsResource, c.ns, opts))

}

// Create takes the representation of a pGClusterBackup and creates it.  Returns the server's representation of the pGClusterBackup, and an error, if there is any.
func (c *FakePGClusterBackups) Create(ctx context.Context, pGClusterBackup *v1alpha1.PGClusterBackup, opts v1.CreateOptions) (result *v1alpha1.PGClusterBackup, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(pgclusterbackupsResource, c.ns, pGClusterBackup), &v1alpha1.PGClusterBackup{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PGClusterBackup), err
}

// Update takes the representation of a pGClusterBackup and updates it. Returns the server's representation of the pGClusterBackup, and an error, if there is any.
func (c *FakePGClusterBackups) Update(ctx context.Context, pGClusterBackup *v1alpha1.PGClusterBackup, opts v1.UpdateOptions) (result *v1alpha1.PGClusterBackup, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(pgclusterbackupsResource, c.ns, pGClusterBackup), &v1alpha1.PGClusterBackup{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PGClusterBackup), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakePGClusterBackups) UpdateStatus(ctx context.Context, pGClusterBackup *v1alpha1.PGClusterBackup, opts v1.UpdateOptions) (*v1alpha1.PGClusterBackup, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(pgclusterbackupsResource, "status", c.ns, pGClusterBackup), &v1alpha1.PGClusterBackup{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PGClusterBackup), err
}

// Delete takes name of the pGClusterBackup and deletes it. Returns an error if one occurs.
func (c *FakePGClusterBackups) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(pgclusterbackupsResource, c.ns, name, opts), &v1alpha1.PGClusterBackup{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakePGClusterBackups) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(pgclusterbackupsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.PGClusterBackupList{})
	return err
}

// Patch applies the patch and returns the patched pGClusterBackup.
func (c *FakePGClusterBackups) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.PGClusterBackup, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(pgclusterbackupsResource, c.ns, name, pt, data, subresources...), &v1alpha1.PGClusterBackup{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PGClusterBackup), err
}
