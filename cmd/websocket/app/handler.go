package app

import (
	"encoding/json"
	"net/http"
	"time"

	"bytetrade.io/web3os/tapr/pkg/constants"
	"bytetrade.io/web3os/tapr/pkg/ws"
	"github.com/go-resty/resty/v2"
	"github.com/gofiber/fiber/v2"
	"k8s.io/klog/v2"
)

type appController struct {
	server     *Server
	httpClient *resty.Client
}

type sendMesssageReq struct {
	Payload interface{} `json:"payload"`
	ConnId  string      `json:"conn_id"`
	Users   []string    `json:"users"`
}

type receiveMessageReq struct {
	Data     interface{} `json:"data"`
	Action   string      `json:"action"`
	UserName string      `json:"user_name"`
	ConnId   string      `json:"conn_id"`
}

type disConnectionReq struct {
	Conns []string `json:"conns"`
	Users []string `json:"users"`
}

func NewController(server *Server) *appController {
	return &appController{
		server:     server,
		httpClient: resty.New().SetTimeout(2 * time.Second),
	}
}

func (a *appController) ListConnection(c *fiber.Ctx) error {
	res := a.server.webSocketServer.List()

	return c.JSON(fiber.Map{
		"code":    0,
		"message": "success",
		"data":    res,
	})
}

func (a *appController) CloseConnection(c *fiber.Ctx) error {
	body := c.Request().Body()
	var closeReq disConnectionReq
	err := json.Unmarshal(body, &closeReq)
	if err != nil {
		klog.Errorf("close connection data invalid, %+v, data: %s", err, string(body))
		return c.JSON(fiber.Map{
			"code":    http.StatusBadRequest,
			"message": "receive data invalid, " + err.Error(),
		})
	}

	a.server.webSocketServer.Close(closeReq.Users, closeReq.Conns)

	return c.JSON(fiber.Map{
		"code":    0,
		"message": "success",
	})
}

func (a *appController) SendMessage(c *fiber.Ctx) error {
	body := c.Request().Body()
	var message sendMesssageReq
	err := json.Unmarshal(body, &message)
	if err != nil {
		klog.Errorf("send message data invalid, %+v, data: %s", err, string(body))
		return c.JSON(fiber.Map{
			"code":    http.StatusBadRequest,
			"message": "send data invalid, " + err.Error(),
		})
	}

	if message.ConnId == "" && (message.Users == nil || len(message.Users) == 0) {
		klog.Errorf("send message target is nil,  data: %s", string(body))
		return c.JSON(fiber.Map{
			"code":    http.StatusBadRequest,
			"message": "send message target is nil",
		})
	}

	a.server.webSocketServer.Push(message.ConnId, message.Users, message.Payload)

	return c.JSON(fiber.Map{
		"code":    0,
		"message": "success",
	})
}

func (a *appController) handleWebSocketMessage(data *ws.ReadMessage) {
	if a.server.appPath == "" {
		return
	}

	var cookie = data.Cookie
	resp, err := a.httpClient.R().SetHeader(constants.WsHeaderCookie, cookie).SetBody(data).Post(a.server.appPath)
	if err != nil {
		klog.Errorf("send to app error, %+v, user: %s, connId: %s", err, data.UserName, data.ConnId)
		return
	}

	if resp.StatusCode() >= 400 {
		klog.Errorf("send to app response status error, %d, user: %s, connId: %s", resp.StatusCode(), data.UserName, data.ConnId)
	}
}

func (a *appController) DebugFunc(c *fiber.Ctx) error {
	// test for debug

	// body := c.Request().Body()
	// var message receiveMessageReq
	// err := json.Unmarshal(body, &message)
	// if err != nil {
	// 	return c.JSON(fiber.Map{
	// 		"code":    http.StatusBadRequest,
	// 		"message": "receive data invalid, " + err.Error(),
	// 	})
	// }

	// var h = c.GetReqHeaders()
	// var cookie = h[constants.WsHeaderCookie]
	// klog.Infof("[debug] receive from websocket data: %s, cookie: %s", body, cookie)

	// if message.Action == "open" || message.Action == "close" {
	// 	return c.JSON(fiber.Map{
	// 		"code":    0,
	// 		"message": "success",
	// 	})
	// }

	// var data = map[string]interface{}{}
	// data["name"] = "hello"
	// data["debug"] = true
	// data["age"] = 20
	// data["orders"] = []string{"order-1", "order-2"}

	// var sm = sendMesssageReq{
	// 	ConnId:  message.ConnId,
	// 	Payload: data,
	// }

	// _, err = a.httpClient.R().SetBody(sm).Post("http://localhost:40010/tapr/ws/conn/send")
	// if err != nil {
	// 	return c.JSON(fiber.Map{
	// 		"code":    http.StatusBadRequest,
	// 		"message": "send to ws-gateway error, " + err.Error(),
	// 	})
	// }

	return c.JSON(fiber.Map{
		"code":    0,
		"message": "success",
	})
}
