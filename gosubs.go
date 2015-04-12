package wsubs

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	SubscribeType   = "Subscribe"
	UnsubscribeType = "Unsubscribe"
	UpdateType      = "Update"
	CreateType      = "Create"
	DeleteType      = "Delete"
	RPCType         = "RPC"
	ErrorType       = "Error"
)

const (
	FatalLevel = iota
	ErrorLevel
	InfoLevel
	DebugLevel
	TraceLevel
)

const (
	FetchType = "Fetch"
)

/*
Prettify will return a nicely indented JSON encoding
of obj
*/
func Prettify(obj interface{}) string {
	b, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(b)
}

/*
JSON wraps anything that is a JSON object.
*/
type JSON struct {
	Data interface{}
}

/*
Get returns the value under key as another JSON.
*/
func (self JSON) Get(key string) JSON {
	return JSON{self.Data.(map[string]interface{})[key]}
}

/*
Overwrite will JSON encode itself and decode it into dest.
*/
func (self JSON) Overwrite(dest interface{}) {
	b, err := json.Marshal(self.Data)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(b, dest)
	if err != nil {
		panic(err)
	}
}

/*
GetStringSLice returns the value under key as a string slice.
*/
func (self JSON) GetStringSlice(key string) (result []string) {
	is := self.Data.(map[string]interface{})[key].([]interface{})
	result = make([]string, len(is))
	for index, i := range is {
		result[index] = fmt.Sprint(i)
	}
	return
}

/*
GetString returns the value under key as a string.
*/
func (self JSON) GetString(key string) string {
	return fmt.Sprint(self.Data.(map[string]interface{})[key])
}

/*
Message wraps Objects in JSON messages.
*/
type Message struct {
	Type   string
	Object *Object `json:",omitempty"`
	Method *Method `json:",omitempty"`
	Error  *Error  `json:",omitempty"`
}

/*
Error encapsulates an error
*/
type Error struct {
	Cause interface{}
	Error string
}

/*
Object is used to send JSON messages to subscribing WebSockets.
*/
type Object struct {
	URI  string
	Data interface{} `json:",omitempty"`
}

/*
Method is used to send JSON RPC requests.
*/
type Method struct {
	Name string
	Id   string
	Data interface{} `json:",omitempty"`
}

/*
ResourceHandler will handle a message regarding an operation on a resource
*/
type ResourceHandler func(c Context) error

/*
Resource describes how the router ought to treat incoming requests for a
resource found under a given URI regexp
*/
type Resource struct {
	Path          *regexp.Regexp
	Handlers      map[string]ResourceHandler
	Authenticated map[string]bool
	lastOp        string
}

/*
Handle tells the router how to handle a given operation on the resource
*/
func (self *Resource) Handle(op string, handler ResourceHandler) *Resource {
	self.Handlers[op] = handler
	self.lastOp = op
	return self
}

/*
Auth tells the router that the op/handler combination defined
in the last Handle call should only receive messages from authenticated
requests (where the Context has a Principal())
*/
func (self *Resource) Auth() *Resource {
	self.Authenticated[self.lastOp] = true
	return self
}

type Resources []*Resource

/*
RPCHandlers handle RPC requests
*/
type RPCHandler func(c Context) (result interface{}, err error)

/*
RPC describes how the router ought to treat incoming requests for
a given RPC method
*/
type RPC struct {
	Method        string
	Handler       RPCHandler
	Authenticated bool
}

/*
Auth tells the router that the RPC should only receive messages from
authenticated requests (where the Context has a Principal())
*/
func (self *RPC) Auth() *RPC {
	self.Authenticated = true
	return self
}

type RPCs []*RPC

type subscriber struct {
	lastActive    time.Time
	subscriptions map[string]struct{}
}

/*
NewRouter returns a router connected to db
*/
func NewRouter() (result *Router) {
	result = &Router{
		Logger:      log.New(os.Stdout, "", 0),
		lock:        &sync.RWMutex{},
		subscribers: map[string]subscriber{},
	}
	result.OnDisconnectFactory = result.DefaultOnDisconnectFactory
	result.OnConnect = result.DefaultOnConnect
	return
}

/*
Router controls incoming WebSocket messages
*/
type Router struct {
	Resources           Resources
	RPCs                RPCs
	Logger              *log.Logger
	LogLevel            int
	OnDisconnectFactory func(ws *websocket.Conn, principal string) func()
	OnConnect           func(ws *websocket.Conn, principal string)
	DevMode             bool
	subscribers         map[string]subscriber
	lock                *sync.RWMutex
	Secret              string
}

/*
MarkActive will timestamp the principal as active, for purposes of checking whether the principal is subscribing.
*/
func (self *Router) MarkActive(principal string) {
	self.lock.Lock()
	defer self.lock.Unlock()
	subs, found := self.subscribers[principal]
	if found {
		subs.lastActive = time.Now()
	}
}

/*
IsSubscriber returns true if principal is currently subscribing to uri.
*/
func (self *Router) IsSubscribing(principal, uri string, timeout time.Duration) (result bool) {
	self.lock.RLock()
	defer self.lock.RUnlock()
	subs, found := self.subscribers[principal]
	if found && subs.lastActive.Add(timeout).After(time.Now()) {
		_, result = subs.subscriptions[uri]
	}
	return
}

/*
DefaultOnConnect will just log the incoming connection
*/
func (self *Router) DefaultOnConnect(ws *websocket.Conn, principal string) {
	self.Infof("%v\t%v <-", ws.RemoteAddr(), principal)
}

/*
DefaultOnDisconnectFactory will return functions that just log the disconnecting connection
*/
func (self *Router) DefaultOnDisconnectFactory(ws *websocket.Conn, principal string) func() {
	return func() {
		self.Infof("%v\t%v -> [unsubscribing all]", ws.RemoteAddr(), principal)
	}
}

/*
Logf will log the format and args if level is less than the LogLevel of this Router
*/
func (self *Router) Logf(level int, format string, args ...interface{}) {
	if level <= self.LogLevel {
		log.Printf(format, args...)
	}
}

/*
Fatalf is shorthand for Logf(FatalLevel...
*/
func (self *Router) Fatalf(format string, args ...interface{}) {
	self.Logf(FatalLevel, "\033[1;31mFATAL\t"+format+"\033[0m", args...)
}

/*
Errorf is shorthand for Logf(ErrorLevel...
*/
func (self *Router) Errorf(format string, args ...interface{}) {
	self.Logf(ErrorLevel, "\033[31mERROR\t"+format+"\033[0m", args...)
}

/*
Infof is shorthand for Logf(InfoLevel...
*/
func (self *Router) Infof(format string, args ...interface{}) {
	self.Logf(InfoLevel, "INFO\t"+format, args...)
}

/*
Debugf is shorthand for Logf(DebugLevel...
*/
func (self *Router) Debugf(format string, args ...interface{}) {
	self.Logf(DebugLevel, "\033[32mDEBUG\t"+format+"\033[0m", args...)
}

/*
Tracef is shorthand for Logf(TraceLevel...
*/
func (self *Router) Tracef(format string, args ...interface{}) {
	self.Logf(TraceLevel, "\033[1;32mTRACE\t"+format+"\033[0m", args...)
}

/*
Resource creates a resource receiving messages matching the provided regexp
*/
func (self *Router) Resource(exp string) (result *Resource) {
	result = &Resource{
		Path:          regexp.MustCompile(exp),
		Handlers:      map[string]ResourceHandler{},
		Authenticated: map[string]bool{},
	}
	self.Resources = append(self.Resources, result)
	return
}

/*
RPC creates an RPC method receiving messages matching the provided method name
*/
func (self *Router) RPC(method string, handler RPCHandler) (result *RPC) {
	result = &RPC{
		Method:  method,
		Handler: handler,
	}
	self.RPCs = append(self.RPCs, result)
	return
}

/*
RemoveSubscriber will remember that principal stopped subscribing to uri
*/
func (self *Router) RemoveSubscriber(principal, uri string) {
	self.lock.Lock()
	defer self.lock.Unlock()
	subs, found := self.subscribers[principal]
	if found {
		delete(subs.subscriptions, uri)
	}
	if len(subs.subscriptions) == 0 {
		delete(self.subscribers, principal)
	}
}

/*
HandleResourceMessage will handle the message that produced c, by finding
a matching resource (if there is one) and sending it the context.
*/
func (self *Router) HandleResourceMessage(c Context) (err error) {
	for _, resource := range self.Resources {
		if !resource.Authenticated[c.Message().Type] || c.Principal() != "" {
			if handler, found := resource.Handlers[c.Message().Type]; found {
				if match := resource.Path.FindStringSubmatch(c.Message().Object.URI); match != nil {
					c.SetMatch(match)
					c.SetData(JSON{c.Message().Object.Data})
					if err = handler(c); err == nil && c.Principal() != "" {
						switch c.Message().Type {
						case SubscribeType:
							self.lock.Lock()
							defer self.lock.Unlock()
							subs, found := self.subscribers[c.Principal()]
							if !found {
								subs = subscriber{
									subscriptions: map[string]struct{}{},
								}
								self.subscribers[c.Principal()] = subs
							}
							subs.lastActive = time.Now()
							subs.subscriptions[c.Message().Object.URI] = struct{}{}
						case UnsubscribeType:
							self.RemoveSubscriber(c.Principal(), c.Message().Object.URI)
						}
					}
					return
				}
			}
		}
	}
	return fmt.Errorf("Unrecognized URI for %v", Prettify(c.Message()))
}

/*
HandleRPCMessage will handle the message that produced c, by finding
a matching RPC method (if there is one) and sending it the context.
*/
func (self *Router) HandleRPCMessage(c Context) (err error) {
	for _, rpc := range self.RPCs {
		if !rpc.Authenticated || c.Principal() != "" {
			if rpc.Method == c.Message().Method.Name {
				var resp interface{}
				c.SetData(JSON{c.Message().Method.Data})
				if resp, err = rpc.Handler(c); err != nil {
					return
				}
				if c.Principal() != "" {
					self.MarkActive(c.Principal())
				}
				return c.Conn().WriteJSON(Message{
					Type: RPCType,
					Method: &Method{
						Name: c.Message().Method.Name,
						Id:   c.Message().Method.Id,
						Data: resp,
					},
				})
			}
		}
	}
	return fmt.Errorf("Unrecognized Method for %v", Prettify(c.Message()))
}

func (self *Router) handleMessage(ws *websocket.Conn, message *Message, principal string) (err error) {
	c := NewContext(ws, message, principal, self)
	switch message.Type {
	case UnsubscribeType, SubscribeType, CreateType, UpdateType, DeleteType:
		return self.HandleResourceMessage(c)
	case RPCType:
		return self.HandleRPCMessage(c)
	}
	return fmt.Errorf("Unknown message type for %v", Prettify(message))
}

/*
DeliverError sends an error message along the provided WebSocket connection
*/
func (self *Router) DeliverError(ws *websocket.Conn, cause interface{}, err error) {
	if err = ws.WriteJSON(&Message{
		Type: ErrorType,
		Error: &Error{
			Cause: cause,
			Error: err.Error(),
		},
	}); err != nil {
		self.Errorf("%v", err)
	}
}

func (self *Router) CheckPrincipal(r *http.Request) (principal string, ok bool) {
	if tok := r.URL.Query().Get("token"); tok != "" {
		token, err := DecodeToken(self.Secret, r.URL.Query().Get("token"))
		if err != nil {
			self.Errorf("%v\t[invalid token: %v]", r.RemoteAddr, err)
			return
		}
		principal = token.Principal
	} else if self.DevMode {
		principal = r.URL.Query().Get("email")
	}
	ok = true
	return
}

/*
SetupConnection will try to find a principal for the provided connection, log it, and
return if it's ok to continue processing it.
*/
func (self *Router) SetupConnection(ws *websocket.Conn) (principal string, ok bool) {
	ok = true
	return
}

func (self *Router) handleConnection(ws *websocket.Conn) {
	if principal, ok := self.SetupConnection(ws); ok {
		defer self.OnDisconnectFactory(ws, principal)()

		handlerFunc := func(message *Message) error {
			return self.handleMessage(ws, message, principal)
		}

		self.ProcessMessages(ws, principal, handlerFunc)
	}
}

/*
ProcessMessages will decode messages from the ws, and handle them with handlerFunc
*/
func (self *Router) ProcessMessages(ws *websocket.Conn, principal string, handlerFunc func(*Message) error) {
	var start time.Time
	for {
		message := &Message{}
		err := ws.ReadJSON(message)
		if err != nil {
			self.DeliverError(ws, nil, err)
			self.Errorf("%v", err)
			break
		}
		start = time.Now()
		if err = handlerFunc(message); err != nil {
			if message.Method != nil {
				self.Errorf("%v\t%v\t%v\t%v\t%v", ws.RemoteAddr(), principal, message.Type, message.Method.Name, err)
			} else if message.Object != nil {
				self.Errorf("%v\t%v\t%v\t%v\t%v", ws.RemoteAddr(), principal, message.Type, message.Object.URI, err)
			} else {
				self.Errorf("%v\t%v\t%+v\t%v", ws.RemoteAddr(), principal, message, err)
			}
			self.DeliverError(ws, message, err)
		}
		if message.Method != nil {
			self.Debugf("%v\t%v\t%v\t%v\t%v <-", ws.RemoteAddr(), principal, message.Type, message.Method.Name, time.Now().Sub(start))
		}
		if message.Object != nil {
			self.Debugf("%v\t%v\t%v\t%v\t%v <-", ws.RemoteAddr(), principal, message.Type, message.Object.URI, time.Now().Sub(start))
		}
		if self.LogLevel > TraceLevel {
			if message.Method != nil && message.Method.Data != nil {
				self.Tracef("%+v", Prettify(message.Method.Data))
			}
			if message.Object != nil && message.Object.Data != nil {
				self.Tracef("%+v", Prettify(message.Object.Data))
			}
		}
	}
}

var upgrader = &websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (self *Router) Upgrader() *websocket.Upgrader {
	return upgrader
}

/*
Implements http.Handler
*/
func (self *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	principal, ok := self.CheckPrincipal(r)
	if !ok {
		return
	}
	ws, err := self.Upgrader().Upgrade(w, r, nil)
	if err != nil {
		self.Errorf("Unable to upgrade %+r: %v", r, err)
		return
	}
	handlerFunc := func(message *Message) error {
		return self.handleMessage(ws, message, principal)
	}

	self.ProcessMessages(ws, principal, handlerFunc)
	defer self.OnDisconnectFactory(ws, principal)()
	self.OnConnect(ws, principal)
}
