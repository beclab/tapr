package nats

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"time"

	aprv1 "bytetrade.io/web3os/tapr/pkg/apis/apr/v1alpha1"
	aprclientset "bytetrade.io/web3os/tapr/pkg/generated/clientset/versioned"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/thoas/go-funk"
	"golang.org/x/crypto/bcrypt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

const ConfPath = "/dbdata/nats_data/config/nats.conf"
const Allow = "allow"

func createOrUpdateUser(request *aprv1.MiddlewareRequest, namespace, password string, loadConfig func() (*Config, error)) (*Config, error) {
	encryptedPassword, err := encryptPassword(password)

	if err != nil {
		return nil, err
	}
	allowPubSubject, allowSubSubject, err := getAllowPubSubSubjectFromMR(request, namespace)
	if err != nil {
		klog.Infof("getAllowPubSubSubjectFromMR, err=%v", err)
		return nil, err
	}
	req := request.Spec.Nats
	user := User{
		Username: req.User,
		Password: encryptedPassword,
		Permissions: Permissions{
			Publish: Publish{
				Allow: allowPubSubject,
			},
			Subscribe: Subscribe{
				Allow: allowSubSubject,
			},
		},
	}
	config, err := loadConfig()
	if err != nil {
		klog.Infof("loadconfig err=%v", err)
		return nil, err
	}
	klog.Infof("nats Config: %#v", config)
	isUserExists := false
	for i, c := range config.Accounts.Terminus.Users {
		if c.Username == req.User {
			config.Accounts.Terminus.Users[i] = user
			isUserExists = true
		}
	}
	if !isUserExists {
		config.Accounts.Terminus.Users = append(config.Accounts.Terminus.Users, user)
	}
	return config, nil
}
func CreateOrUpdateUser(request *aprv1.MiddlewareRequest, namespace, password string) (*Config, error) {
	config, err := createOrUpdateUser(request, namespace, password, loadConf)
	if err != nil {
		klog.Infof("createOrUpdateUser err=%v", err)
		return nil, err
	}
	err = RenderConfigFile(config)
	if err != nil {
		klog.Info("renderConfigFile err=%v", err)
		return nil, err
	}
	return config, nil
}

func getAllowPubSubSubjectFromMR(request *aprv1.MiddlewareRequest, namespace string) ([]string, []string, error) {
	req := request.Spec.Nats.DeepCopy()
	for i, s := range req.Subjects {
		req.Subjects[i].Name = MakeRealSubjectName(s.Name, request.Spec.AppNamespace)

	}
	for i, ref := range req.Refs {
		for j, s := range ref.Subjects {
			req.Refs[i].Subjects[j].Name = MakeRealNameForRefSubjectName(ref.AppNamespace, ref.AppName, s.Name, GetOwnerNameFromNs(request.Namespace))
		}
	}

	allowPubSubject := make([]string, 0)
	allowSubSubject := make([]string, 0)
	for _, subject := range req.Subjects {
		if subject.Permission.Pub == Allow {
			allowPubSubject = append(allowPubSubject, subject.Name)
		}
		if subject.Permission.Sub == Allow {
			allowSubSubject = append(allowSubSubject, subject.Name)
		}
	}

	config, err := ctrl.GetConfig()
	if err != nil {
		return allowPubSubject, allowSubSubject, err
	}
	aprClient, err := aprclientset.NewForConfig(config)
	if err != nil {
		return allowPubSubject, allowSubSubject, err
	}
	mrs, err := aprClient.AprV1alpha1().MiddlewareRequests("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return allowPubSubject, allowSubSubject, err
	}
	type export struct {
		subjectName string
		appName     string
		pub         string
		sub         string
	}

	appExportMap := make(map[string][]export)
	for _, mr := range mrs.Items {
		if mr.Spec.Middleware != aprv1.TypeNats {
			continue
		}
		for _, s := range mr.Spec.Nats.Subjects {
			for _, e := range s.Export {
				appExportMap[e.AppName] = append(appExportMap[e.AppName],
					export{
						subjectName: MakeRealSubjectName(s.Name, mr.Spec.AppNamespace),
						appName:     e.AppName,
						pub:         e.Pub,
						sub:         e.Sub,
					},
				)
			}

		}
	}
	klog.Infof("appExportMap: %#v", appExportMap)
	klog.Infof("req.Nats: %#v", req)

	for _, ref := range req.Refs {
		for _, subject := range ref.Subjects {
			if _, ok := appExportMap[request.Spec.App]; !ok {
				return allowPubSubject, allowPubSubject, errors.New("not found export permission")
			}
			klog.Infof("request.spec.App: %v", request.Spec.App)
			ep := appExportMap[request.Spec.App]
			klog.Infof("ep: %#v\n", ep)

			if funk.Contains(subject.Perm, "pub") {
				getPerm := false
				for _, e := range ep {
					klog.Infof("subject.Name: %v, e.subjectName: %v\n", subject.Name, e.subjectName)
					if subject.Name == e.subjectName && e.pub == Allow {
						allowPubSubject = append(allowPubSubject, subject.Name)
						getPerm = true
					}
				}
				if !getPerm {
					return allowPubSubject, allowPubSubject, fmt.Errorf("not found export permission for subject %s", subject.Name)
				}

			}
			if funk.Contains(subject.Perm, "sub") {
				getPerm := false
				for _, e := range ep {
					klog.Infof("subject.Name: %v, e.subjectName: %v\n", subject.Name, e.subjectName)

					if subject.Name == e.subjectName && e.sub == Allow {
						allowSubSubject = append(allowSubSubject, subject.Name)
						getPerm = true
					}
				}
				if !getPerm {
					return allowPubSubject, allowPubSubject, fmt.Errorf("not found export permission for subject %s", subject.Name)
				}

			}
		}
	}

	allowPubSubject = append(allowPubSubject, []string{"$JS.API.INFO", "$JS.API.STREAM.NAMES", "$JS.API.CONSUMER.CREATE.>", "_INBOX.>", "$JS.ACK.>"}...)
	allowSubSubject = append(allowSubSubject, []string{"$JS.API.STREAM.NAMES", "$JS.API.CONSUMER.CREATE.>", "_INBOX.>"}...)

	return allowPubSubject, allowSubSubject, nil
}

func CreateOrUpdateStream(appNamespace, app string) error {
	name := fmt.Sprintf("%s-%s", appNamespace, app)
	adminPassword, err := getAdminPassword()
	if err != nil {
		return err
	}
	nc, err := nats.Connect("nats://nats.os-system", nats.UserInfo("admin", adminPassword))
	if err != nil {
		return err
	}
	defer nc.Drain()
	js, err := jetstream.New(nc)
	if err != nil {
		return err
	}
	cfg := jetstream.StreamConfig{
		Name:     name,
		Subjects: []string{"terminus.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   15 * 24 * 60 * 60 * time.Second,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = js.CreateStream(ctx, cfg)
	if err != nil && errors.Is(err, jetstream.ErrStreamNameAlreadyInUse) {
		return err
	}
	return nil
}

func DeleteStream(appNamespace, app string) error {
	name := fmt.Sprintf("%s-%s", appNamespace, app)
	adminPassword, err := getAdminPassword()
	if err != nil {
		return err
	}
	nc, err := nats.Connect("nats://nats.os-system", nats.UserInfo("admin", adminPassword))
	if err != nil {
		return err
	}
	defer nc.Drain()
	js, err := jetstream.New(nc)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = js.DeleteStream(ctx, name)
	if err != nil && errors.Is(err, nats.ErrStreamNotFound) {
		return err
	}
	return nil
}

func DeleteUser(username string) error {
	config, err := loadConf()
	if err != nil {
		return err
	}
	for i, u := range config.Accounts.Terminus.Users {
		if u.Username == username {
			config.Accounts.Terminus.Users = append(config.Accounts.Terminus.Users[:i],
				config.Accounts.Terminus.Users[i+1:]...)
		}
	}
	err = RenderConfigFile(config)
	if err != nil {
		return err
	}
	return nil
}

func encryptPassword(password string) (string, error) {
	encryptedPass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(encryptedPass), nil
}

func loadConf() (*Config, error) {
	password, err := getAdminPassword()
	if err != nil {
		return nil, err
	}
	err = os.Setenv("ADMIN_PASSWORD", password)
	if err != nil {
		klog.Infof("set env error=%v", err)
		return nil, err
	}
	return ParseFile(ConfPath)
}

var ch = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@$#%^&*()")

func sizedBytes(sz int) []byte {
	b := make([]byte, sz)
	for i := range b {
		b[i] = ch[rand.Intn(len(ch))]
	}
	return b
}

func sizedString(sz int) string {
	return string(sizedBytes(sz))
}

var re = regexp.MustCompile(`^(?:[^.]*\.){3}(.*)$`)

func GetOriginSubjectName(subjectName string) string {
	match := re.FindStringSubmatch(subjectName)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

func newClientSet() (*kubernetes.Clientset, error) {
	config, err := ctrl.GetConfig()
	if err != nil {
		klog.Infof("get config err=%v")
		return nil, err
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Infof("create clientset err=%v", clientSet)
		return nil, err
	}
	return clientSet, nil
}

func getAdminPassword() (string, error) {
	clientSet, err := newClientSet()
	if err != nil {
		return "", err
	}
	secret, err := clientSet.CoreV1().Secrets("os-system").Get(context.TODO(), "nats-secrets", metav1.GetOptions{})
	if err != nil {
		klog.Infof("get secret err=%v", secret)
		return "", err
	}
	password, ok := secret.Data["nats_password"]
	if !ok {
		klog.Infof("empty nats-Password")
		return "", err
	}

	return string(password), nil
}

func MakeRealSubjectName(subject string, appNamespace string) string {
	return fmt.Sprintf("terminus.%s.%s", appNamespace, subject)
}

func MakeRealNameForRefSubjectName(refNamespace, app, subject, ownerName string) string {
	refAppNs := ""
	if strings.HasPrefix(refNamespace, "user-space") {
		refAppNs = fmt.Sprintf("user-space-%s", ownerName)
	} else if strings.HasPrefix(refNamespace, "user-system") {
		refAppNs = fmt.Sprintf("user-system-%s", ownerName)
	} else if refNamespace == "os-system" {
		refAppNs = "os-system"
	} else {
		refAppNs = refNamespace
	}
	return fmt.Sprintf("terminus.%s.%s", refAppNs, subject)
}

func GetOwnerNameFromNs(ns string) string {
	nsSplict := strings.Split(ns, "-")
	return nsSplict[len(nsSplict)-1]
}
