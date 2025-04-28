package middleware

import (
	"strings"

	"bytetrade.io/web3os/tapr/pkg/constants"
	"bytetrade.io/web3os/tapr/pkg/utils"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"k8s.io/klog/v2"
)

var accessPublic = func() (string, string, bool) {
	var token = uuid.New().String()
	var userName = token
	return token, userName, true
}

func RequireHeader() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		if !websocket.IsWebSocketUpgrade(c) {
			return fiber.ErrUpgradeRequired
		}
		var headers = c.GetReqHeaders()
		if headers == nil {
			return fiber.ErrBadRequest
		}

		var connId = uuid.New().String()

		// If it's a public environment access, cookie data is invalid, set to anonymous state, making token equal to userName, and also equal to uuid

		var token, userName, accessPublic = GetHeadersUserInfo(headers)
		var userAgent = headers[constants.WsHeaderUserAgent]
		var forwarded = headers[constants.WsHeaderForwardeFor]
		var cookie = headers[constants.WsHeaderCookie]

		klog.Infof("ws-client conn: %s, accessPublic: %v, token: %s, user: %s , header: %+v", connId, accessPublic, token, userName, headers)

		var secWebsocketProtocol, ok = headers[constants.WsHeaderSecWebsocketProtocol]
		if ok {
			c.Set(constants.WsHeaderSecWebsocketProtocol, secWebsocketProtocol)
		}

		c.Locals(constants.WsLocalAccessPublic, accessPublic)
		c.Locals(constants.WsLocalUserKey, userName)
		c.Locals(constants.WsLocalConnIdKey, connId)
		c.Locals(constants.WsLocalTokenKey, utils.MD5(token))
		c.Locals(constants.WsLocalTokenKeyOriginal, token)
		c.Locals(constants.WsLocalUserAgentKey, userAgent)
		c.Locals(constants.WsLocalClientIpKey, forwarded)
		c.Locals(constants.WsLocalCookie, cookie)

		return c.Next()
	}
}

func GetHeadersUserInfo(headers map[string]string) (string, string, bool) {
	var username = headers[constants.WsHeaderBflUser]
	if strings.EqualFold(username, "") {
		return accessPublic()
	}

	var cookie = headers[constants.WsHeaderCookie]

	if strings.EqualFold(cookie, "") {
		return accessPublic()
	}

	var token string
	var authToken string
	var items = strings.Split(cookie, ";")
	if items == nil || len(items) == 0 {
		return accessPublic()
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
		return accessPublic()
	}

	var tokensplit = strings.Split(authToken, "=")
	if tokensplit == nil || len(tokensplit) != 2 {
		return accessPublic()
	}

	if tokensplit[1] == "" {
		return accessPublic()
	}

	token = tokensplit[1]

	return token, username, false
}
