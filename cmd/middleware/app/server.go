package app

import (
	"context"

	"bytetrade.io/web3os/tapr/pkg/app/middleware"
	aprclientset "bytetrade.io/web3os/tapr/pkg/generated/clientset/versioned"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type Server struct {
	Ctx           context.Context
	KubeConfig    *rest.Config
	app           *fiber.App
	k8sClientSet  *kubernetes.Clientset
	aprClientSet  *aprclientset.Clientset
	dynamicClient *dynamic.DynamicClient
}

func (s *Server) ServerRun() {
	s.k8sClientSet = kubernetes.NewForConfigOrDie(s.KubeConfig)
	s.aprClientSet = aprclientset.NewForConfigOrDie(s.KubeConfig)
	s.dynamicClient = dynamic.NewForConfigOrDie(s.KubeConfig)

	// create new fiber instance  and use across whole app
	app := fiber.New()

	// middleware to allow all clients to communicate using http and allow cors
	app.Use(cors.New())

	app.Post("/middleware/v1/request/info", middleware.RequireAuth(s.KubeConfig, s.handleGetMiddlewareRequestInfo))
	app.Get("/middleware/v1/requests", middleware.RequireAuth(s.KubeConfig,
		middleware.RequireAdmin(s.KubeConfig, s.handleListMiddlewareRequests)))

	app.Get("/middleware/v1/:middleware/list", middleware.RequireAuth(s.KubeConfig,
		middleware.RequireAdmin(s.KubeConfig, s.handleListMiddlewares)))
	app.Post("/middleware/v1/:middleware/scale", middleware.RequireAuth(s.KubeConfig,
		middleware.RequireAdmin(s.KubeConfig, s.handleScaleMiddleware)))
	app.Post("/middleware/v1/:middleware/password", middleware.RequireAuth(s.KubeConfig,
		middleware.RequireAdmin(s.KubeConfig, s.handleUpdateMiddlewareAdminPassword)))

	s.app = app
	err := app.Listen(":9080")
	if err != nil {
		klog.Fatal(err)
	}
}

func (s *Server) Shutdown() {
	s.app.Shutdown()
}
