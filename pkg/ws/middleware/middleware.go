package middleware

import (
	"fmt"
	"strings"

	"bytetrade.io/web3os/tapr/pkg/constants"
	"bytetrade.io/web3os/tapr/pkg/utils"
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

		var connId = fmt.Sprintf("%d", c.Context().ConnID())

		var token, err = GetToken(headers[constants.WsHeaderCookie])
		if err != nil {
			klog.Errorf("get token error, %+v, user: %s, connId: %s, headers: %+v", err, headers[constants.WsHeaderBflUser], connId, headers)
			return err
		}

		klog.Infof("ws-client conn: %s, token: %s, header: %+v", connId, token, headers)

		var secWebsocketProtocol, ok = headers[constants.WsHeaderSecWebsocketProtocol]
		if ok {
			c.Set(constants.WsHeaderSecWebsocketProtocol, secWebsocketProtocol)
		}

		c.Locals(constants.WsLocalUserKey, headers[constants.WsHeaderBflUser])
		c.Locals(constants.WsLocalConnIdKey, connId)
		c.Locals(constants.WsLocalTokenKey, utils.MD5(token))
		c.Locals(constants.WsLocalTokenKeyOriginal, token)
		c.Locals(constants.WsLocalUserAgentKey, headers[constants.WsHeaderUserAgent])
		c.Locals(constants.WsLocalClientIpKey, headers[constants.WsHeaderForwardeFor])
		c.Locals(constants.WsLocalCookie, headers[constants.WsHeaderCookie])

		return c.Next()
	}
}

func GetToken(cookie string) (string, *fiber.Error) {
	var token string
	var authToken string
	var items = strings.Split(cookie, ";")
	if items == nil || len(items) == 0 {
		return token, fiber.ErrForbidden
	}

	var found bool
	for _, item := range items {
		item = strings.TrimSpace(item)
		if strings.Contains(item, "auth_token=") {
			found = true
			authToken = item
			break
		}
	}

	if !found {
		return token, fiber.ErrUnauthorized
	}

	var tokensplit = strings.Split(authToken, "=")
	if tokensplit == nil || len(tokensplit) != 2 {
		return token, fiber.ErrUnauthorized
	}

	if tokensplit[1] == "" {
		return token, fiber.ErrUnauthorized
	}

	token = tokensplit[1]

	return token, nil

}
