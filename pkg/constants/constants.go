package constants

const (
	KubeSphereNamespace        = "kubesphere-system"
	KubeSphereConfigName       = "kubesphere-config"
	KubeSphereConfigMapDataKey = "kubesphere.yaml"

	AuthorizationTokenKey       = "X-Authorization"
	AuthorizationTokenCookieKey = "auth_token"

	SystemNamespace = "os-system"
)

var (
	UsernameCtxKey           = []byte("username")
	UserPwdCtxKey            = []byte("userpassword")
	UserEmailCtxKey          = []byte("useremail")
	UserAuthTokenCtxKey      = []byte("userAuthToken")
	UserRefreshTokenCtxKey   = []byte("userRefreshToken")
	UserCtxKey               = []byte("user")
	UserPrivateKeyCtxKey     = []byte("userPrivateKey")
	UserOrganizationIdCtxKey = []byte("userOrganizationId")
)

var (
	WsHeaderCtxKey               = "wsHeader"
	WsHeaderUserAgent            = "User-Agent"
	WsHeaderBflUser              = "X-Bfl-User"
	WsHeaderToken                = "X-Authorization"
	WsHeaderForwardeFor          = "X-Original-Forwarded-For"
	WsHeaderSecWebsocketProtocol = "Sec-Websocket-Protocol"
	WsHeaderConnId               = "connId"
	WsHeaderCookie               = "Cookie"

	WsLocalClientIpKey      = "client-ip"
	WsLocalClientAddrKey    = "client-addr"
	WsLocalUserKey          = "user-name"
	WsLocalTokenKey         = "token"
	WsLocalTokenKeyOriginal = "token-original"
	WsLocalConnIdKey        = "id"
	WsLocalUserAgentKey     = "user-agent"
	WsLocaExpiredKey        = "expired"
	WsLocalCookie           = "cookie"

	WsEnvAppPort = "WS_PORT"
	WsEnvAppUrl  = "WS_URL"

	UploadFileType    = "UPLOAD_FILE_TYPE"
	UploadLimitedSize = "UPLOAD_LIMITED_SIZE"
)
