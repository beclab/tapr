package middleware

import (
	"fmt"

	"bytetrade.io/web3os/tapr/pkg/constants"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"k8s.io/klog/v2"
)

func RequireHeader() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		if !websocket.IsWebSocketUpgrade(c) {
			return fiber.ErrUpgradeRequired
		}
		var headers = c.GetReqHeaders()
		if headers == nil {
			return fiber.ErrUpgradeRequired
		}

		klog.Infof("ws-client header: %+v", headers)

		var secWebsocketProtocol, ok = headers[constants.WsHeaderSecWebsocketProtocol]
		if ok {
			c.Set(constants.WsHeaderSecWebsocketProtocol, secWebsocketProtocol)
		}

		c.Locals(constants.WsLocalUserKey, headers[constants.WsHeaderBflUser])
		c.Locals(constants.WsLocalConnIdKey, fmt.Sprintf("%d", c.Context().ConnID()))
		c.Locals(constants.WsLocalTokenKey, headers[constants.WsHeaderToken])
		c.Locals(constants.WsLocalUserAgentKey, headers[constants.WsHeaderUserAgent])
		c.Locals(constants.WsLocalClientIpKey, headers[constants.WsHeaderForwardeFor])
		c.Locals(constants.WsLocalCookie, headers[constants.WsHeaderCookie])

		return c.Next()
	}
}
