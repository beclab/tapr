package app

import (
	"context"
	"fmt"
	"time"

	"bytetrade.io/web3os/tapr/pkg/app/middleware"
	"bytetrade.io/web3os/tapr/pkg/kubesphere"
	"bytetrade.io/web3os/tapr/pkg/vault/infisical"
	"bytetrade.io/web3os/tapr/pkg/vault/infisical/controllers"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type Server struct {
	KubeConfig *rest.Config
}

func (s *Server) Init() error {
	ctx := context.Background()
	pguser, password, err := s.getPostgresUserAndPwd(ctx)
	if err != nil {
		return err
	}

	var client *infisical.PostgresClient

	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", pguser, password, infisical.InfisicalDBAddr, infisical.InfisicalDBName)

	// try and wait for infisical postgres to connect
	func() {
		for {
			if client, err = infisical.NewClient(dsn); err != nil {
				klog.Info("connecting infisical postres error, ", err, ".  Waiting ... ")
				time.Sleep(time.Second)
			} else {
				return
			}
		}
	}()
	defer client.Close()

	// init user
	user, err := kubesphere.GetUser(ctx, s.KubeConfig, infisical.Owner)
	if err != nil {
		return err
	}

	u, err := client.GetUser(ctx, user.Spec.Email)
	if err != nil {
		return err
	}

	if u == nil {
		err = infisical.InsertKsUserToPostgres(ctx, client, infisical.Owner, user.Spec.Email, infisical.Password)
		if err != nil {
			klog.Error("init user error, ", err)
			return err
		}
	}

	return nil
}

func (s *Server) ServerRun() {
	// create new fiber instance  and use across whole app
	app := fiber.New()

	// middleware to allow all clients to communicate using http and allow cors
	app.Use(cors.New())

	//
	// routes
	//
	routes := controllers.New()
	clientSet := controllers.NewClientset()
	routes.WithClientset(clientSet).
		WithDynamicClient(dynamic.NewForConfigOrDie(s.KubeConfig))

	tokenIssuer := infisical.NewTokenIssuer(s.KubeConfig).WithUserAndPwd(s.getPostgresUserAndPwd)
	// tapr auth token for infisical
	app.Post("/tapr/auth/token",
		middleware.GetOwnerInfo(s.KubeConfig, infisical.Owner,
			tokenIssuer.IssueInfisicalToken(routes.AuthToken)))

	app.Post("/tapr/privatekey",
		middleware.GetOwnerInfo(s.KubeConfig, infisical.Owner,
			tokenIssuer.IssueInfisicalToken(routes.PrivateKey)))

	// put secret in workspace
	app.Post("/secret/create",
		middleware.GetOwnerInfo(s.KubeConfig, infisical.Owner,
			tokenIssuer.IssueInfisicalToken(
				controllers.FetchUserPrivateKey(clientSet,
					controllers.FetchUserOrganizationId(clientSet, routes.CreateSecret)))))

	// delete secret in workspace
	app.Post("/secret/delete",
		middleware.GetOwnerInfo(s.KubeConfig, infisical.Owner,
			tokenIssuer.IssueInfisicalToken(
				controllers.FetchUserPrivateKey(clientSet,
					controllers.FetchUserOrganizationId(clientSet, routes.DeleteSecret)))))

	// update secret in workspace
	app.Post("/secret/update",
		middleware.GetOwnerInfo(s.KubeConfig, infisical.Owner,
			tokenIssuer.IssueInfisicalToken(
				controllers.FetchUserPrivateKey(clientSet,
					controllers.FetchUserOrganizationId(clientSet, routes.UpdateSecret)))))

	// get secret in workspace
	app.Post("/secret/retrieve",
		middleware.GetOwnerInfo(s.KubeConfig, infisical.Owner,
			tokenIssuer.IssueInfisicalToken(
				controllers.FetchUserPrivateKey(clientSet,
					controllers.FetchUserOrganizationId(clientSet, routes.RetrieveSecret)))))

	// list secrets in workspace
	app.Post("/secret/list",
		middleware.GetOwnerInfo(s.KubeConfig, infisical.Owner,
			tokenIssuer.IssueInfisicalToken(
				controllers.FetchUserPrivateKey(clientSet,
					controllers.FetchUserOrganizationId(clientSet, routes.ListSecret)))))

	// api for settings
	// check app secrets permission
	app.Get("/admin/permission/:appid",
		middleware.GetOwnerInfo(s.KubeConfig, infisical.Owner,
			tokenIssuer.IssueInfisicalToken(
				controllers.FetchUserPrivateKey(clientSet,
					controllers.FetchUserOrganizationId(clientSet, routes.CheckAppSecretPerm)))))

	// list app secrets
	app.Get("/admin/secret/:appid",
		middleware.GetOwnerInfo(s.KubeConfig, infisical.Owner,
			tokenIssuer.IssueInfisicalToken(
				controllers.FetchUserPrivateKey(clientSet,
					controllers.FetchUserOrganizationId(clientSet, routes.ListAppSecret)))))

	// create app secrets
	app.Post("/admin/secret/:appid",
		middleware.GetOwnerInfo(s.KubeConfig, infisical.Owner,
			tokenIssuer.IssueInfisicalToken(
				controllers.FetchUserPrivateKey(clientSet,
					controllers.FetchUserOrganizationId(clientSet, routes.CreateAppSecret)))))

	// delete app secrets
	app.Delete("/admin/secret/:appid",
		middleware.GetOwnerInfo(s.KubeConfig, infisical.Owner,
			tokenIssuer.IssueInfisicalToken(
				controllers.FetchUserPrivateKey(clientSet,
					controllers.FetchUserOrganizationId(clientSet, routes.DeleteAppSecret)))))

	// update app secrets
	app.Put("/admin/secret/:appid",
		middleware.GetOwnerInfo(s.KubeConfig, infisical.Owner,
			tokenIssuer.IssueInfisicalToken(
				controllers.FetchUserPrivateKey(clientSet,
					controllers.FetchUserOrganizationId(clientSet, routes.UpdateAppSecret)))))

	klog.Info("secret-vault http server listening on 8080 ")
	klog.Fatal(app.Listen(":8080"))
}

func (s *Server) getSecretPwd(ctx context.Context, secretName string, secretKey string) (pwd string, err error) {
	client, err := kubernetes.NewForConfig(s.KubeConfig)
	if err != nil {
		return "", err
	}

	secret, err := client.CoreV1().Secrets(infisical.InfisicalNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	pwd = string(secret.Data[secretKey])

	return pwd, nil
}

func (s *Server) getMongoUserAndPwd(ctx context.Context) (user string, pwd string, err error) {
	user = infisical.InfisicalDBUser
	pwd, err = s.getSecretPwd(ctx, "infisical-mongodb", "mongodb-passwords")

	return user, pwd, nil
}

func (s *Server) getPostgresUserAndPwd(ctx context.Context) (user string, pwd string, err error) {
	user = infisical.InfisicalDBUser
	pwd, err = s.getSecretPwd(ctx, "infisical-postgres", "postgres-passwords")

	return user, pwd, nil
}
