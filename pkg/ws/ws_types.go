package ws

const queueSize = 500
const expirationTicker = 30
const expirationDuration = 45

type CloseMessage struct {
	ConnIds         []string
	Tokens          []string
	Users           []string
	UsersWithPublic bool `json:"users_with_public"`
}

type ReadMessage struct {
	Data         interface{} `json:"data"`
	Action       string      `json:"action"`
	Cookie       string      `json:"-"`
	UserName     string      `json:"user_name"`
	ConnId       string      `json:"conn_id"`
	Token        string      `json:"token"`
	AccessPublic bool        `json:"access_public"`
}

type WriteMessage struct {
	MessageType     int
	Message         interface{} `json:"message"`
	ConnId          string      `json:"conn_id"`
	Tokens          []string    `json:"tokens"`
	Users           []string    `json:"users"`
	UsersWithPublic bool        `json:"users_with_public"`
}
