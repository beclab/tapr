package ws

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"k8s.io/klog/v2"
)

type callback func(data *ReadMessage)

type WebSocketServer interface {
	New() func(c *fiber.Ctx) error
	SetHandler(cb callback)

	List() []map[string]interface{}
	Close(users []string, tokens []string, conns []string)
	Push(connId string, tokens []string, users []string, message interface{})
}

type Server struct {
	queue struct {
		read  chan *ReadMessage  // from client
		write chan *WriteMessage // from app
		close chan *CloseMessage // from app
	}

	handler callback

	publics map[string]*Public // token
	users   map[string]*User

	sync.RWMutex
}

type Public struct {
	id    string // connId
	token string
	conn  *Client
}

type User struct {
	name  string             // userName
	conns map[string]*Client // connId / client
	sync.RWMutex
}

func NewWebSocketServer() WebSocketServer {
	var server = &Server{}
	server.users = map[string]*User{}
	server.publics = map[string]*Public{}
	server.queue.read = make(chan *ReadMessage, queueSize)
	server.queue.write = make(chan *WriteMessage, queueSize)
	server.queue.close = make(chan *CloseMessage, queueSize)

	go server.routineRead()
	go server.routineWrite()
	go server.routineClose()
	go server.checkExpired()

	return server
}

func (server *Server) New() func(c *fiber.Ctx) error {
	return websocket.New(func(c *websocket.Conn) {
		ctx, cancelFunc := context.WithCancel(context.Background())

		var client = &Client{
			conn:         c,
			ctx:          ctx,
			cancel:       cancelFunc,
			closeHandler: server.close,
			readHandler:  server.read,
			writeHandler: server.write,
		}

		client.setLocals()

		server.addClient(client).noticeConnected(client).onConnection()
	})
}

func (server *Server) SetHandler(cb callback) {
	server.handler = cb
}

func (server *Server) List() []map[string]interface{} {
	server.RLock()
	defer server.RUnlock()

	var res = []map[string]interface{}{}

	for _, z := range server.users {
		if z == nil {
			continue
		}

		var ccs = []map[string]string{}
		var r = map[string]interface{}{}
		r["name"] = z.name
		z.RLock()
		for _, c := range z.conns {
			tokenOriginal := c.getTokenOriginal()
			connId := c.getConnId()
			userAgent := c.getUserAgent()
			userAgentTag := c.md5([]byte(userAgent))

			var cs = map[string]string{}
			cs["id"] = connId
			cs["token"] = tokenOriginal
			cs["userAgent"] = userAgent
			cs["userAgentTag"] = userAgentTag
			ccs = append(ccs, cs)
		}
		r["conns"] = ccs
		r["conns_number"] = len(ccs)
		res = append(res, r)
		z.RUnlock()
	}

	var publics = map[string]interface{}{}
	var ccs = []map[string]string{}
	for _, p := range server.publics {
		if p == nil {
			continue
		}

		var cs = map[string]string{}
		cs["id"] = p.id
		cs["token"] = p.token
		cs["userAgent"] = p.conn.getUserAgent()
		cs["userAgentTag"] = p.conn.md5([]byte(p.conn.getUserAgent()))
		ccs = append(ccs, cs)
	}
	publics["conns"] = ccs
	publics["conns_number"] = len(ccs)
	publics["name"] = "publics"
	res = append(res, publics)

	return res
}

func (server *Server) Close(users []string, tokens []string, conns []string) {
	var m = &CloseMessage{
		Users:  users,
		Tokens: tokens,
		Conns:  conns,
	}

	server.queue.close <- m
}

func (server *Server) Push(connId string, tokens []string, users []string, message interface{}) {
	var m = &WriteMessage{
		MessageType: websocket.TextMessage,
		ConnId:      connId,
		Tokens:      tokens,
		Users:       users,
		Message:     message,
	}

	server.queue.write <- m
}

func (server *Server) addClient(c *Client) *Client {
	server.Lock()
	defer server.Unlock()

	var userName = c.getUser()
	var accessPublic = c.getAccessLevel()
	var connId = c.getConnId()

	if !accessPublic {
		user, ok := server.users[userName]
		if !ok {
			var newUser = &User{conns: map[string]*Client{}}
			newUser.Lock()
			newUser.name = userName
			newUser.conns[connId] = c
			server.users[userName] = newUser
			newUser.Unlock()
			return c
		}

		user.Lock()
		user.conns[connId] = c
		user.Unlock()

		return c
	}

	var token = c.getToken()
	var tokenOriginal = c.getTokenOriginal()
	server.publics[token] = &Public{
		id:    connId,
		token: tokenOriginal,
		conn:  c,
	}

	return c
}

func (server *Server) close(connId string) {
	server.queue.close <- &CloseMessage{
		Conns: []string{connId},
	}
}

func (server *Server) routineClose() {
	for {
		select {
		case elem, ok := <-server.queue.close:
			if !ok {
				server.queue.close = make(chan *CloseMessage, queueSize)
				continue
			}
			server.closeConns(elem.Users, elem.Tokens, elem.Conns)
		}
	}
}

func (server *Server) closeConns(users []string, tokens []string, conns []string) {
	var filter = NewFilter(server)

	if users != nil && len(users) > 0 {
		filter.FilterByUsers(users)
	}
	if tokens != nil && len(tokens) > 0 {
		filter.FilterByTokens(tokens)
	}
	if conns != nil && len(conns) > 0 {
		filter.FilterByConnIds(conns)
	}

	var result = filter.Result()
	if result == nil || len(result) == 0 {
		return
	}

	var removeusers []string
	var removepublics []string
	server.Lock()
	for userName, userClients := range server.users {
		userClients.Lock()
		for _, connId := range result {
			client, ok := userClients.conns[connId]
			if ok && client.conn != nil {
				delete(userClients.conns, connId)
				client.close()
			}
		}
		if userClients.conns == nil || len(userClients.conns) == 0 {
			removeusers = append(removeusers, userName)
		}
		userClients.Unlock()
	}

	for token, public := range server.publics {
		for _, connId := range result {
			if public.id == connId {
				public.conn.close()
				removepublics = append(removepublics, token)
			}
		}
	}

	for _, removeuser := range removeusers {
		delete(server.users, removeuser)
	}

	for _, removetoken := range removepublics {
		delete(server.publics, removetoken)
	}

	server.Unlock()
}

func (server *Server) read(accessPublic bool, token, connId, userName string, message interface{}, cookie string, action string) {
	server.queue.read <- &ReadMessage{
		AccessPublic: accessPublic,
		Token:        token,
		ConnId:       connId,
		UserName:     userName,
		Data:         message,
		Action:       action,
		Cookie:       cookie,
	}
}

func (server *Server) routineRead() {
	for {
		select {
		case elem, ok := <-server.queue.read:
			if !ok {
				server.queue.read = make(chan *ReadMessage, queueSize)
				continue
			}
			server.handler(elem)
		}
	}
}

func (server *Server) write(connId string, msgType int, data interface{}) {
	var w = &WriteMessage{
		MessageType: msgType,
		Message:     data,
		ConnId:      connId,
	}
	server.queue.write <- w
}

func (server *Server) routineWrite() {
	for {
		select {
		case elem, ok := <-server.queue.write:
			if !ok {
				server.queue.write = make(chan *WriteMessage, queueSize)
				continue
			}
			msg, err := json.Marshal(elem.Message)
			if err != nil {
				klog.Errorf("send message marshal error %+v, data: %v", err, elem.Message)
				continue
			}

			klog.Infof("send message data: %s, connId: %s, token: %v, users: %v", string(msg), elem.ConnId, elem.Tokens, elem.Users)

			var filter = NewFilter(server)
			if elem.Users != nil && len(elem.Users) > 0 {
				filter.FilterByUsers(elem.Users)
			}
			if elem.Tokens != nil && len(elem.Tokens) > 0 {
				filter.FilterByTokens(elem.Tokens)
			}
			if elem.ConnId != "" {
				filter.FilterByConnIds([]string{elem.ConnId})
			}

			var result = filter.Result()

			if result != nil && len(result) > 0 {
				server.RLock()
				for _, userClients := range server.users {
					userClients.RLock()
					for _, connId := range result {
						conn, ok := userClients.conns[connId]
						if ok && conn != nil {
							conn.conn.WriteMessage(elem.MessageType, msg)
						}
					}
					userClients.RUnlock()
				}

				for _, public := range server.publics {
					for _, connId := range result {
						if public.id == connId {
							public.conn.conn.WriteMessage(elem.MessageType, msg)
						}
					}
				}

				server.RUnlock()
			}
		}
	}
}

func (server *Server) checkExpired() {
	for range time.NewTicker(expirationTicker * time.Second).C {
		f := NewFilter(server)
		result := f.FilterByExpired().Result()
		if len(result) > 0 {
			server.queue.close <- &CloseMessage{
				Conns: result,
			}
		}
	}
}
