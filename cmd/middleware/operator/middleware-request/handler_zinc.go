package middlewarerequest

import (
	"errors"

	aprv1 "bytetrade.io/web3os/tapr/pkg/apis/apr/v1alpha1"
	"bytetrade.io/web3os/tapr/pkg/workload/zinc"
	"k8s.io/klog/v2"
)

func init() {
	initFunc = append(initFunc, func(c *controller) {
		adminUser, adminPwd, err := zinc.FindAdminUser(c.ctx, c.k8sClientSet)
		if err != nil {
			panic(err)
		}

		klog.Info("init default role to zinc server")
		err = zinc.InitRole(adminUser, adminPwd)
		if err != nil {
			// make sure zinc is running normally
			panic(err)
		}
	})
}

func (c *controller) createOrUpdataIndexForUser(req *aprv1.MiddlewareRequest) error {
	if req.Spec.Zinc.Indexes == nil || req.Spec.Zinc.User == "" {
		return errors.New("zinc config is nil or user is empty")
	}

	adminUser, adminPwd, err := zinc.FindAdminUser(c.ctx, c.k8sClientSet)
	if err != nil {
		return err
	}

	password, err := req.Spec.Zinc.Password.GetVarValue(c.ctx, c.k8sClientSet, req.Namespace)
	if err != nil {
		return err
	}

	klog.Info("create user into zinc server, ", req.Spec.Zinc.User)
	err = zinc.CreateOrUpdateUser(adminUser, adminPwd, req.Spec.Zinc.User, password)
	if err != nil {
		return err
	}

	for _, index := range req.Spec.Zinc.Indexes {
		klog.Info("create index into zinc server, ", index.Name, " for user, ", req.Namespace)
		schema, err := zinc.FindIndexConfig(c.ctx, c.k8sClientSet, index.Namespace, index.Name, index.Key)
		if err != nil {
			return err
		}

		err = zinc.CreateOrUpdateIndex(adminUser, adminPwd, req.Spec.AppNamespace, index.Name, schema)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *controller) deleteIndexAndUser(req *aprv1.MiddlewareRequest) error {
	if req.Spec.Zinc.Indexes == nil || req.Spec.Zinc.User == "" {
		klog.Warning("zinc config is nil or user is empty")
		return nil
	}

	adminUser, adminPwd, err := zinc.FindAdminUser(c.ctx, c.k8sClientSet)
	if err != nil {
		return err
	}

	klog.Info("delete user from zinc server, ", req.Spec.Zinc.User)
	err = zinc.DeleteUser(adminUser, adminPwd, req.Spec.Zinc.User)
	if err != nil {
		return err
	}

	for _, index := range req.Spec.Zinc.Indexes {
		klog.Info("delete index from zinc server, ", index.Name, " for user, ", req.Namespace)

		err = zinc.DeleteIndex(adminUser, adminPwd, req.Spec.AppNamespace, index.Name)
		if err != nil {
			return err
		}
	}

	return nil
}
