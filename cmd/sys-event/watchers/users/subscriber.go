package users

import (
	"context"

	"bytetrade.io/web3os/tapr/cmd/sys-event/watchers"
	aprv1 "bytetrade.io/web3os/tapr/pkg/apis/apr/v1alpha1"
	"bytetrade.io/web3os/tapr/pkg/kubesphere"
	"k8s.io/klog/v2"
)

type Subscriber struct {
	notification *watchers.Notification
}

func (s *Subscriber) Do(ctx context.Context, obj interface{}, action watchers.Action) error {
	admin, err := s.notification.AdminUser(ctx)
	if err != nil {
		return err
	}
	user := obj.(*kubesphere.User)
	switch action {
	case watchers.ADD:
		klog.Info("user ", user.Name, " is created")
		if s.notification != nil {
			return s.notification.Send(ctx, admin, "user "+user.Name+" is created", &watchers.EventPayload{
				Type: string(aprv1.UserCreate),
				Data: map[string]interface{}{
					"user": user.Name,
				},
			})
		}
	case watchers.DELETE:
		klog.Info("user ", user.Name, " is deleted")
		if s.notification != nil {
			return s.notification.Send(ctx, admin, "user "+user.Name+" is deleted", &watchers.EventPayload{
				Type: string(aprv1.UserDelete),
				Data: map[string]interface{}{
					"user": user.Name,
				},
			})
		}
	}
	return nil
}
