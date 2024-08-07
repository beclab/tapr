package app

import (
	"fmt"

	aprv1 "bytetrade.io/web3os/tapr/pkg/apis/apr/v1alpha1"
	"bytetrade.io/web3os/tapr/pkg/workload/citus"
	"bytetrade.io/web3os/tapr/pkg/workload/nats"
	"bytetrade.io/web3os/tapr/pkg/workload/percona"
	rediscluster "bytetrade.io/web3os/tapr/pkg/workload/redis-cluster"
	"bytetrade.io/web3os/tapr/pkg/workload/zinc"

	"github.com/gofiber/fiber/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func (s *Server) getMiddlewareInfo(ctx *fiber.Ctx, mwReq *MiddlewareReq, m *aprv1.MiddlewareRequest) (*MiddlewareRequestResp, error) {
	resp := &MiddlewareRequestResp{}

	var err error
	switch m.Spec.Middleware {
	case aprv1.TypePostgreSQL:
		resp.UserName = m.Spec.PostgreSQL.User
		resp.Password, err = m.Spec.PostgreSQL.Password.GetVarValue(ctx.UserContext(), s.k8sClientSet, mwReq.Namespace)
		if err != nil {
			klog.Error("get middleware password error, ", err)
			return nil, err
		}

		klog.Info("find pg cluster service, ", citus.CitusMasterServiceName)
		svc, err := s.k8sClientSet.CoreV1().Services(mwReq.Namespace).Get(ctx.UserContext(), citus.CitusMasterServiceName, metav1.GetOptions{})
		if err != nil {
			klog.Error("get pg cluster service error, ", err)
			return nil, err
		}

		// default 5432
		resp.Port = 5432
		for _, port := range svc.Spec.Ports {
			if port.Name == "citus" {
				resp.Port = port.Port
			}
		}

		// klog.Info("match pods by service selecor, ", svc.Spec.Selector)
		// pods, err := s.k8sClientSet.CoreV1().Pods(citus.PGClusterNamespace).List(ctx.UserContext(), metav1.ListOptions{
		// 	LabelSelector: metav1.FormatLabelSelector(metav1.SetAsLabelSelector(labels.Set(svc.Spec.Selector))),
		// })

		// if err != nil {
		// 	klog.Error("find pods error, ", err)
		// 	return nil, err
		// }

		// for _, p := range pods.Items {
		// 	if strings.HasSuffix(p.Name, "-0") {
		// 		// first pods in sts is master
		// 		resp.Host = p.Name + "." + svc.Name + "." + mwReq.Namespace

		// 		return resp, nil
		// 	}
		// }

		resp.Host = citus.CitusMasterServiceName + "." + mwReq.Namespace
		resp.Databases = make(map[string]string)
		for _, db := range m.Spec.PostgreSQL.Databases {
			resp.Databases[db.Name] = citus.GetDatabaseName(m.Spec.AppNamespace, db.Name)
		}

		return resp, nil

	case aprv1.TypeMongoDB:
		resp.UserName = m.Spec.MongoDB.User
		resp.Password, err = m.Spec.MongoDB.Password.GetVarValue(ctx.UserContext(), s.k8sClientSet, mwReq.Namespace)
		if err != nil {
			klog.Error("get middleware password error, ", err)
			return nil, err
		}

		klog.Info("find percona cluster service, ", percona.PerconaMongoService)
		svc, err := s.k8sClientSet.CoreV1().Services(mwReq.Namespace).Get(ctx.UserContext(), percona.PerconaMongoService, metav1.GetOptions{})
		if err != nil {
			klog.Error("get percona cluster service error, ", err)
			return nil, err
		}

		resp.Port = 27017
		for _, port := range svc.Spec.Ports {
			if port.Name == percona.PerconaMongoProxy {
				resp.Port = port.Port
			}
		}

		resp.Host = percona.PerconaMongoService + "." + mwReq.Namespace

		resp.Databases = make(map[string]string)
		for _, db := range m.Spec.MongoDB.Databases {
			resp.Databases[db.Name] = percona.GetDatabaseName(m.Spec.AppNamespace, db.Name)
		}

		return resp, nil

	case aprv1.TypeRedis:
		resp.Password, err = m.Spec.Redis.Password.GetVarValue(ctx.UserContext(), s.k8sClientSet, mwReq.Namespace)
		if err != nil {
			klog.Error("get middleware password error, ", err)
			return nil, err
		}

		klog.Info("find redis cluster service, ", rediscluster.RedisClusterService)
		svc, err := s.k8sClientSet.CoreV1().Services(mwReq.Namespace).Get(ctx.UserContext(), rediscluster.RedisClusterService, metav1.GetOptions{})
		if err != nil {
			klog.Error("get redis cluster service error, ", err)
			return nil, err
		}

		resp.Port = 6379
		for _, port := range svc.Spec.Ports {
			if port.Name == "proxy" {
				resp.Port = port.Port
			}
		}

		resp.Databases = make(map[string]string)
		resp.Host = rediscluster.RedisClusterService + "." + mwReq.Namespace
		resp.Databases[m.Spec.Redis.Namespace] = rediscluster.GetDatabaseName(m.Spec.AppNamespace, m.Spec.Redis.Namespace)

		return resp, nil
	case aprv1.TypeZinc:
		resp.UserName = m.Spec.Zinc.User
		resp.Password, err = m.Spec.Zinc.Password.GetVarValue(ctx.UserContext(), s.k8sClientSet, mwReq.Namespace)
		if err != nil {
			klog.Error("get middleware password error, ", err)
			return nil, err
		}

		resp.Port = 80
		resp.Host = "zinc-server-svc." + mwReq.Namespace

		resp.Indexes = make(map[string]string)
		for _, index := range m.Spec.Zinc.Indexes {
			resp.Indexes[index.Name] = zinc.GetIndexName(m.Spec.AppNamespace, index.Name)
		}

		return resp, nil
	case aprv1.TypeNats:
		resp.UserName = m.Spec.Nats.User
		resp.Password, err = m.Spec.Nats.Password.GetVarValue(ctx.UserContext(), s.k8sClientSet, mwReq.Namespace)
		resp.Port = 4222
		resp.Host = "nats." + mwReq.Namespace
		resp.Subjects = make(map[string]string)
		for _, subject := range m.Spec.Nats.Subjects {
			resp.Subjects[subject.Name] = nats.MakeRealSubjectName(subject.Name, m.Spec.AppNamespace)
		}
		appSubjectMap := make(map[string]string)
		ownerName := nats.GetOwnerNameFromNs(m.Namespace)
		for _, ref := range m.Spec.Nats.Refs {
			for _, subject := range ref.Subjects {
				appSubjectMap[fmt.Sprintf("%s_%s", ref.AppName, subject.Name)] = nats.MakeRealNameForRefSubjectName(ref.AppNamespace, ref.AppName, subject.Name, ownerName)
			}
		}
		resp.Refs = appSubjectMap
		return resp, nil

	} // end of middleware type

	return nil, fiber.NewError(fiber.StatusNotImplemented, "middleware type unsupported")
}
