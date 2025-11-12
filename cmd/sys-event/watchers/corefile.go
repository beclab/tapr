package watchers

import (
	"context"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/coredns/corefile-migration/migration/corefile"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

func RegenerateCorefile(ctx context.Context, kubeClient kubernetes.Interface, dynamicClient dynamic.Interface) error {
	corefileConfigMap, err := kubeClient.CoreV1().ConfigMaps("kube-system").Get(ctx, "coredns", metav1.GetOptions{})
	if err != nil {
		klog.Error("get coredns configmap error, ", err)
		return err
	}

	corefileData := corefileConfigMap.Data["Corefile"]
	file, err := corefile.New(corefileData)
	if err != nil {
		klog.Error("parse corefile error, ", err)
		return err
	}

	if len(file.Servers) < 1 {
		klog.Warning("invalid corefile configuration")
		return nil
	}

	defaultsServer := file.Servers[0]
	var defaultPlugins []*corefile.Plugin
	for _, p := range defaultsServer.Plugins {
		switch p.Name {
		case "errors", "health", "ready", "kubernetes", "prometheus", "forward", "cache", "loop", "reload", "loadbalance":
			defaultPlugins = append(defaultPlugins, p)
		}
	}

	userList, err := dynamicClient.Resource(schema.GroupVersionResource{
		Group:    "iam.kubesphere.io",
		Version:  "v1alpha2",
		Resource: "users",
	}).List(ctx, metav1.ListOptions{})
	if err != nil {
		klog.Error("get userlist error, ", err)
		return err
	}

	nodeList, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		klog.Error("get nodelist error, ", err)
		return err
	}

	var masterNodeIp string
	for _, node := range nodeList.Items {
		if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; ok {
			for _, addr := range node.Status.Addresses {
				if addr.Type == "InternalIP" {
					masterNodeIp = addr.Address
					break
				}
			}
		}
	}

	var templatesPlugins []*corefile.Plugin
	var localTemplatesPlugins []*corefile.Plugin
	var localDomainTemplatesPlugins []*corefile.Plugin

	addUserTemplates := func(zone, ip string, plugins []*corefile.Plugin) []*corefile.Plugin {
		newOptions := []*corefile.Option{
			{
				Name: "match",
				Args: []string{fmt.Sprintf("\"\\w*\\.?(%s\\.)$\"", zone)},
			},
			{
				Name: "answer",
				Args: []string{fmt.Sprintf("\"{{ .Name }} 60 IN A %s\"", ip)},
			},
			{
				Name: "fallthrough",
				Args: []string{},
			},
		}
		anyOptions := []*corefile.Option{
			{
				Name: "rcode",
				Args: []string{"NOERROR"},
			},
		}
		userTemplateArgs := []string{"IN", "A", zone}
		userTemplateAnyArgs := []string{"IN", "ANY", zone}

		plugins = append(plugins, &corefile.Plugin{
			Name:    "template",
			Args:    userTemplateArgs,
			Options: newOptions,
		})

		plugins = append(plugins, &corefile.Plugin{
			Name:    "template",
			Args:    userTemplateAnyArgs,
			Options: anyOptions,
		})

		return plugins
	} // func addUserTemplates

	for _, u := range userList.Items {
		userzone := u.GetAnnotations()[UserAnnotationZoneKey]
		if userzone == "" {
			klog.Info("user ", u.GetName(), " has no zone annotation, skip corefile update")
			continue
		}

		ip, err := getUserLocalIp(&u)
		if err != nil {
			klog.Error("get user local ip error, ", err)
			return err
		}
		if ip == nil || ip.String() == "" {
			klog.Info("user ", u.GetName(), " has no valid local ip, skip corefile update")
			continue
		}

		templatesPlugins = addUserTemplates(userzone, ip.String(), templatesPlugins)

		if masterNodeIp == "" {
			klog.Info("no master node ip found, skip adding local domain dns record")
			continue
		}

		username := u.GetName()
		userLocalZone := fmt.Sprintf("%s.olares.local", username)
		localTemplatesPlugins = addUserTemplates(userzone, masterNodeIp, localTemplatesPlugins)
		localDomainTemplatesPlugins = addUserTemplates(userLocalZone, masterNodeIp, localDomainTemplatesPlugins)
	}

	var adguardIp string
	pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{LabelSelector: "applications.app.bytetrade.io/name=adguardhome"})
	if err != nil {
		klog.Error("get adguardhome pod error, ", err)
	} else {
		adguardIp = pods.Items[0].Status.PodIP
	}

	inclusterExpr := "incidr(client_ip(), '10.233.0.0/16')"
	if adguardIp != "" {
		inclusterExpr = fmt.Sprintf("( %s && client_ip() != '%s' )", inclusterExpr, adguardIp)
	}
	inclusterExpr = fmt.Sprintf("%s || client_ip() == '%s'", inclusterExpr, masterNodeIp)

	inclusterView := &corefile.Plugin{
		Name: "view",
		Args: []string{"incluster"},
		Options: []*corefile.Option{
			{
				Name: "expr",
				Args: []string{inclusterExpr},
			},
		},
	}

	inclusterServer := &corefile.Server{
		DomPorts: defaultsServer.DomPorts,
		Plugins:  append([]*corefile.Plugin{inclusterView}, append(defaultPlugins, templatesPlugins...)...),
	}

	otherServer := &corefile.Server{
		DomPorts: defaultsServer.DomPorts,
		Plugins: append(defaultPlugins,
			append(localTemplatesPlugins, localDomainTemplatesPlugins...)...),
	}

	file.Servers = []*corefile.Server{inclusterServer, otherServer}

	newCorefileData := file.ToString()
	corefileConfigMap.Data["Corefile"] = newCorefileData

	_, err = kubeClient.CoreV1().ConfigMaps("kube-system").Update(ctx, corefileConfigMap, metav1.UpdateOptions{})
	if err != nil {
		klog.Error("update coredns configmap error, ", err)
		return err
	}

	klog.Info("coredns corefile regenerated successfully")
	return nil
}

func UpsertCorefile(data, userzone, ip string) (string, error) {
	file, err := corefile.New(data)
	if err != nil {
		klog.Error("parse corefile error, ", err)
		return "", err
	}

	if len(file.Servers) != 1 {
		klog.Warning("invalid corefile configuration")
		return data, nil
	}

	var newPlugins []*corefile.Plugin
	found := false
	newOptions := []*corefile.Option{
		{
			Name: "match",
			Args: []string{fmt.Sprintf("\"\\w*\\.?(%s\\.)$\"", userzone)},
		},
		{
			Name: "answer",
			Args: []string{fmt.Sprintf("\"{{ .Name }} 60 IN A %s\"", ip)},
		},
		{
			Name: "fallthrough",
			Args: []string{},
		},
	}
	anyOptions := []*corefile.Option{
		{
			Name: "rcode",
			Args: []string{"NOERROR"},
		},
	}
	userTemplateArgs := []string{"IN", "A", userzone}
	userTemplateAnyArgs := []string{"IN", "ANY", userzone}

	for _, p := range file.Servers[0].Plugins {
		// only care about template plugins
		if p.Name != "template" {
			newPlugins = append(newPlugins, p)
			continue
		}

		if len(p.Args) != 3 {
			// the template is not added by us, keep it
			klog.Info(p.Args)
			newPlugins = append(newPlugins, p)
			continue
		}

		// update query type A with new options
		if p.Args[2] == userTemplateArgs[2] && p.Args[1] == userTemplateArgs[1] {
			found = true
			p.Options = newOptions
			newPlugins = append(newPlugins, p)
		} else if p.Args[2] == userTemplateAnyArgs[2] && p.Args[1] == userTemplateAnyArgs[1] {
			// update query type ANY with ANY options
			p.Options = anyOptions
			newPlugins = append(newPlugins, p)
		} else {
			// another user's template, keep it
			for _, o := range p.Options {
				switch o.Name {
				case "match", "answer":
					// fix args to one string
					o.Args = []string{fmt.Sprintf("\"%s\"", strings.Join(o.Args, " "))}
				}
			}
			newPlugins = append(newPlugins, p)
		}
	}

	if !found {
		newPlugins = append(newPlugins, &corefile.Plugin{
			Name:    "template",
			Args:    userTemplateArgs,
			Options: newOptions,
		})

		newPlugins = append(newPlugins, &corefile.Plugin{
			Name:    "template",
			Args:    userTemplateAnyArgs,
			Options: anyOptions,
		})
	}

	file.Servers[0].Plugins = newPlugins

	return file.ToString(), nil
}

func RemoveTemplateFromCorefile(data, userzone string) (string, error) {
	file, err := corefile.New(data)
	if err != nil {
		klog.Error("parse corefile error, ", err)
		return "", err
	}

	if len(file.Servers) != 1 {
		klog.Warning("invalid corefile configuration")
		return data, nil
	}

	var newPlugins []*corefile.Plugin
	userTemplateArgs := []string{"IN", "A", userzone}
	for _, p := range file.Servers[0].Plugins {
		// only care about template plugins
		if p.Name != "template" {
			newPlugins = append(newPlugins, p)
			continue
		}

		if len(p.Args) != 3 {
			// the template is not added by us, keep it
			klog.Info(p.Args)
			newPlugins = append(newPlugins, p)
			continue
		}

		if p.Args[2] == userTemplateArgs[2] {
			// remove the template plugin
			continue
		}
	}

	file.Servers[0].Plugins = newPlugins

	return file.ToString(), nil
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

func getUserLocalIp(user *unstructured.Unstructured) (net.IP, error) {
	userIndex, ok := user.GetAnnotations()[UserIndexAna]
	if !ok || userIndex == "" {
		klog.Infof("can not find user index from annotations")
		return nil, nil
	}

	userMaxStr := os.Getenv("OLARES_MAX_USERS")
	if userMaxStr == "" {
		userMaxStr = "1024"
	}
	userMax, err := strconv.ParseInt(userMaxStr, 10, 64)
	if err != nil {
		klog.Infof("parse user index failed %v", err)
		return nil, err
	}
	localIp := subDNSSplit(userMax)[userIndex]
	if localIp == nil || localIp.String() == "" {
		return nil, fmt.Errorf("invalid ip address %v", localIp)
	}
	klog.Infof("localIp: %v", localIp)

	return localIp, nil
}

const UserAnnotationZoneKey = "bytetrade.io/zone"
const UserAnnotationLocalDomainDNSRecord = "bytetrade.io/local-domain-dns-record"
const UserIndexAna = "bytetrade.io/user-index"
