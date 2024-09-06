package app

import (
	"bytetrade.io/web3os/tapr/pkg/constants"
	"bytetrade.io/web3os/tapr/pkg/signals"
	"bytetrade.io/web3os/tapr/pkg/upload/fileutils"

	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"math"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"strconv"
	"strings"
)

const (
	filePath = "filepath"
)

type Server struct {
	app *fiber.App

	controller  *appController
	fileInfoMgr *fileutils.FileInfoMgr

	supportedFileTypes map[string]bool
	allowAllFileType   bool
	limitedSize        int64
	context            context.Context
	k8sClient          *kubernetes.Clientset
}

func (server *Server) Init() error {
	server.getEnvAppInfo()
	server.app = fiber.New(fiber.Config{
		BodyLimit: math.MaxInt, // this is the default limit of 10MB
	})
	// middleware to allow all clients to communicate using http and allow cors
	server.app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,HEAD,PUT,DELETE,PATCH",
		AllowHeaders: "Origin, Content-Type, Accept, Content-Length, Upload-Offset, Upload-Metadata, Upload-Length, X-Authorization, x-authorization, Content-Disposition, Content-Range, Referer, User-Agent",
	}))
	server.controller = newController(server)
	server.fileInfoMgr = fileutils.NewFileInfoMgr()
	server.fileInfoMgr.Init()

	fileutils.Init()

	ctx, cancel := context.WithCancel(context.Background())
	_ = signals.SetupSignalHandler(ctx, cancel)
	server.context = ctx

	config := ctrl.GetConfigOrDie()
	server.k8sClient = kubernetes.NewForConfigOrDie(config)

	PVCs = NewPVCCache(server)

	return nil
}

func (server *Server) ServerRun() {
	//small file upload
	server.app.Post("/upload/tapr/small", server.controller.UploadSmallFile)
	//chunk upload
	server.app.Post("/upload/tapr/chunk", server.controller.UploadChunk)
	server.app.Post("/upload/tapr/chunk/merge", server.controller.MergeChunk)
	server.app.Get("/upload/tapr/chunk/state", server.controller.ChunkState)

	server.app.Post("/upload/", server.controller.UploadFile)
	server.app.Patch("/upload/:uid", server.controller.PatchFile)
	//server.app.Get("/upload/info/:uid?", server.controller.Info)

	server.app.Get("/upload/upload-link", server.controller.UploadLink)
	server.app.Get("/upload/file-uploaded-bytes", server.controller.UploadedBytes)
	server.app.Post("/upload/upload-link/:uid", server.controller.UploadChunks)

	klog.Info("upload server listening on 40030")
	klog.Fatal(server.app.Listen(":40030"))
}

func (s *Server) getEnvAppInfo() {
	var uploadFileType, uploadLimitedSize string

	uploadFileType = os.Getenv(constants.UploadFileType)
	s.supportedFileTypes = make(map[string]bool)
	if uploadFileType == "" {
		s.allowAllFileType = true
	} else {
		fileTypes := strings.Split(uploadFileType, ",")
		for _, ft := range fileTypes {
			if ft == "*" {
				s.allowAllFileType = true
			}
			s.supportedFileTypes[ft] = true
		}
	}

	uploadLimitedSize = os.Getenv(constants.UploadLimitedSize)

	size, err := strconv.ParseInt(uploadLimitedSize, 10, 64)
	if err != nil {
		klog.Errorf("uploadLimitedSize:%s parse int err:%v", uploadLimitedSize, err)
	}
	s.limitedSize = size
	if s.limitedSize <= 0 {
		s.limitedSize = fileutils.DefaultMaxFileSize
	}

	klog.Infof("uploadFileType:%s, uploadLimitedSize:%s", uploadFileType, uploadLimitedSize)
	klog.Infof("allowAllFileType:%t supportedFileTypes:%v, limitedSize:%d", s.allowAllFileType, s.supportedFileTypes, s.limitedSize)
}

func (s *Server) checkType(filetype string) bool {
	if s.allowAllFileType {
		return true
	}

	return s.supportedFileTypes[filetype]
}

func (s *Server) checkSize(filesize int64) bool {
	if filesize < 0 {
		return false
	}

	if s.limitedSize <= 0 {
		return true
	}

	return s.limitedSize >= filesize
}
