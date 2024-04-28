package app

type appController struct {
	server *Server
}

func newController(server *Server) *appController {
	return &appController{
		server: server,
	}
}
