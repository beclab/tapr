package middlewarerequest

import (
	"errors"

	aprv1 "bytetrade.io/web3os/tapr/pkg/apis/apr/v1alpha1"

	workload_nats "bytetrade.io/web3os/tapr/pkg/workload/nats"
	"k8s.io/klog/v2"
)

func (c *controller) createOrUpdateNatsUser(req *aprv1.MiddlewareRequest) error {
	if req.Spec.Nats.User == "" {
		return errors.New("nats user is empty")
	}
	password, err := req.Spec.Nats.Password.GetVarValue(c.ctx, c.k8sClientSet, req.Spec.AppNamespace)
	if err != nil {
		klog.Infof("get password err=%v", err)
		return err
	}
	_, err = workload_nats.CreateOrUpdateUser(req, req.Namespace, password)
	if err != nil {
		klog.Infof("create user err=%v", err)
		return err
	}
	err = workload_nats.CreateOrUpdateStream(req.Spec.AppNamespace, req.Spec.App)
	if err != nil {
		klog.Infof("create stream err=%v", err)
		return nil
	}
	return nil
}

func (c *controller) deleteNatsUserAndStream(req *aprv1.MiddlewareRequest) error {
	err := workload_nats.DeleteUser(req.Spec.Nats.User)
	if err != nil {
		return err
	}
	err = workload_nats.DeleteStream(req.Spec.AppNamespace, req.Spec.App)
	if err != nil {
		return err
	}
	return nil
}
