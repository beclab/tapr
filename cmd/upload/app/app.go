package app

import "bytetrade.io/web3os/tapr/pkg/upload/fileutils"

type appController struct {
	server      *Server
	fileHandler *fileutils.FileHandler
}

func newController(server *Server) *appController {
	return &appController{
		server:      server,
		fileHandler: fileutils.NewFileHandler(),
	}
}
