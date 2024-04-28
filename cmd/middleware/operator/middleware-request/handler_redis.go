package middlewarerequest

import (
	aprv1 "bytetrade.io/web3os/tapr/pkg/apis/apr/v1alpha1"
	rediscluster "bytetrade.io/web3os/tapr/pkg/workload/redis-cluster"
)

func (c *controller) reconcileRedisPassword(req *aprv1.MiddlewareRequest) error {
	return rediscluster.UpdateProxyConfig(c.ctx, c.k8sClientSet, c.aprClientSet, c.dynamicClient)
}
