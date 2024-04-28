package ws

const queueSize = 500
const expirationTicker = 30
const expirationDuration = 45

type CloseMessage struct {
	Conns []string
	Users []string
}

type ReadMessage struct {
	Data     interface{} `json:"data"`
	Action   string      `json:"action"`
	Cookie   string      `json:"-"`
	UserName string      `json:"user_name"`
	ConnId   string      `json:"conn_id"`
}

type WriteMessage struct {
	MessageType int
	Message     interface{} `json:"message"`
	ConnId      string      `json:"conn_id"`
	Users       []string    `json:"users"`
}
