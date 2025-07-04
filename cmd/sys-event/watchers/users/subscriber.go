package users

import (
	"context"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"

	"bytetrade.io/web3os/tapr/cmd/sys-event/watchers"
	aprv1 "bytetrade.io/web3os/tapr/pkg/apis/apr/v1alpha1"
	"bytetrade.io/web3os/tapr/pkg/kubesphere"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type task interface {
	doAdd(context.Context, *kubesphere.User) error
	doDelete(context.Context, *kubesphere.User) error
	doUpdate(context.Context, *kubesphere.User) error
}

var _ task = &Notify{}

// ////////////////////////////////
// notification task
type Notify struct {
	notification *watchers.Notification
}

// doAdd implements task.
func (n *Notify) doAdd(ctx context.Context, user *kubesphere.User) error {
	admin, err := n.notification.AdminUser(ctx)
	if err != nil {
		return err
	}

	if n.notification != nil {
		return n.notification.Send(ctx, admin, "user "+user.Name+" is created", &watchers.EventPayload{
			Type: string(aprv1.UserCreate),
			Data: map[string]interface{}{
				"user": user.Name,
			},
		})
	}

	return nil
}

// doDelete implements task.
func (n *Notify) doDelete(ctx context.Context, user *kubesphere.User) error {
	admin, err := n.notification.AdminUser(ctx)
	if err != nil {
		return err
	}

	if n.notification != nil {
		return n.notification.Send(ctx, admin, "user "+user.Name+" is deleted", &watchers.EventPayload{
			Type: string(aprv1.UserDelete),
			Data: map[string]interface{}{
				"user": user.Name,
			},
		})
	}

	return nil
}

// doUpdate implements task.
func (n *Notify) doUpdate(ctx context.Context, user *kubesphere.User) error { return nil }

// ////////////////////////////////
// update coredns task
var _ task = &UserDomain{}

type UserDomain struct {
	client kubernetes.Interface
}

// doAdd implements task.
func (u *UserDomain) doAdd(context.Context, *kubesphere.User) error { return nil }

// doDelete implements task.
func (u *UserDomain) doDelete(ctx context.Context, user *kubesphere.User) error {
	return u.updateCorefile(ctx, user, func(data, zone, _ string) (string, error) {
		return watchers.RemoveTemplateFromCorefile(data, zone)
	})
}

// doUpdate implements task.
func (u *UserDomain) doUpdate(ctx context.Context, user *kubesphere.User) error {
	return u.updateCorefile(ctx, user, func(data, zone, ip string) (string, error) {
		return watchers.UpsertCorefile(data, zone, ip)
	})
}

func (u *UserDomain) updateCorefile(ctx context.Context, user *kubesphere.User, f func(data, zone, ip string) (string, error)) error {
	zone, ok := user.Annotations[UserAnnotationZoneKey]
	if !ok || zone == "" {
		// zone not bind, ignore
		return nil
	}

	userIndex, ok := user.Annotations[UserIndexAna]
	if !ok || userIndex == "" {
		klog.Infof("can not find user index from annotations")
		return nil
	}

	userMaxStr := os.Getenv("OLARES_MAX_USERS")
	if userMaxStr == "" {
		userMaxStr = "1024"
	}
	userMax, err := strconv.ParseInt(userMaxStr, 10, 64)
	if err != nil {
		klog.Infof("parse user index failed %v", err)
		return err
	}
	localIp := subDNSSplit(userMax)[userIndex]
	if localIp == nil || localIp.String() == "" {
		return fmt.Errorf("invalid ip address %v", localIp)
	}
	klog.Infof("localIp: %v", localIp)
	corednsCm, err := u.client.CoreV1().ConfigMaps("kube-system").Get(ctx, "coredns", metav1.GetOptions{})
	if err != nil {
		klog.Error("get core dns config map error, ", err)
		return err
	}

	corefileData, ok := corednsCm.Data["Corefile"]
	if !ok || corefileData == "" {
		klog.Warning("core dns config map is empty")
		return nil
	}

	newCorefileData, err := f(corefileData, zone, localIp.String())
	if err != nil {
		return err
	}

	corednsCm.Data["Corefile"] = newCorefileData

	_, err = u.client.CoreV1().ConfigMaps("kube-system").Update(ctx, corednsCm, metav1.UpdateOptions{})
	if err != nil {
		klog.Error("update core dns configmap error, ", err)
	}

	return err
}

type Subscriber struct {
	tasks []task
}

func (s *Subscriber) Do(ctx context.Context, obj interface{}, action watchers.Action) error {
	user := obj.(*kubesphere.User)
	switch action {
	case watchers.ADD:
		klog.Info("user ", user.Name, " is created")
		for _, t := range s.tasks {
			if err := t.doAdd(ctx, user); err != nil {
				return err
			}
		}
	case watchers.DELETE:
		klog.Info("user ", user.Name, " is deleted")
		for _, t := range s.tasks {
			if err := t.doDelete(ctx, user); err != nil {
				return err
			}
		}
	case watchers.UPDATE:
		klog.Info("user ", user.Name, " is updated")
		for _, t := range s.tasks {
			if err := t.doUpdate(ctx, user); err != nil {
				return err
			}
		}
	}
	return nil
}

func subDNSSplit(n int64) map[string]net.IP {
	subDNSMap := make(map[string]net.IP)
	log2n := int(math.Ceil(math.Log2(float64(n))))
	alignedN := 1 << log2n
	_, ipNet, _ := net.ParseCIDR("100.64.0.0/10")

	baseIP := ipNet.IP.To4()
	originalMaskLen, _ := ipNet.Mask.Size()

	newMaskLen := originalMaskLen + log2n
	ipsPerSubnet := 1 << (32 - newMaskLen)

	for i := 0; i < alignedN; i++ {
		offset := uint32(i * ipsPerSubnet)
		subnetIP := make(net.IP, 4)
		copy(subnetIP, baseIP)
		for j := 3; j >= 0 && offset > 0; j-- {
			subnetIP[j] += byte(offset & 0xFF)
			offset >>= 8
		}
		firstUsableIP := make(net.IP, 4)
		copy(firstUsableIP, subnetIP)
		firstUsableIP[3]++
		index := strconv.FormatInt(int64(i), 10)
		subDNSMap[index] = firstUsableIP
	}
	return subDNSMap
}
