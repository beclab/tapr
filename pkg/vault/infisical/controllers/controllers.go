package controllers

import (
	"k8s.io/client-go/dynamic"
)

type controllers struct {
	workspaceController
	secretController
	authController
	adminController
}

func New() *controllers {
	return &controllers{}
}

func (c *controllers) WithClientset(cs *Clientset) *controllers {
	f := func() *Clientset { return cs }
	c.secretController.Clientset = f
	c.workspaceController.Clientset = f
	c.adminController.Clientset = f
	return c
}

func (c *controllers) WithDynamicClient(client *dynamic.DynamicClient) *controllers {
	c.adminController.DynamicClient = func() *dynamic.DynamicClient { return client }

	return c
}
