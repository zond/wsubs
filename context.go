package wsubs

import (
	"time"

	"github.com/gorilla/websocket"
)

/*
SubscriptionManager be managin' subscriptions
*/
type SubscriptionManager interface {
	IsSubscribing(principal, uri string, timeout time.Duration) bool
}

/*
Loggers be loggin'
*/
type Logger interface {
	Fatalf(format string, params ...interface{})
	Errorf(format string, params ...interface{})
	Infof(format string, params ...interface{})
	Debugf(format string, params ...interface{})
	Tracef(format string, params ...interface{})
}

/*
LoggingSubscriptionManager logs and manages subscriptions...
*/
type LoggingSubscriptionManager interface {
	SubscriptionManager
	Logger
}

/*
Context describes a single WebSocket message and its environment
*/
type Context interface {
	Logger
	Conn() *websocket.Conn
	Message() *Message
	Principal() string
	Match() []string
	SetMatch([]string)
	Data() JSON
	SetData(JSON)
	IsSubscribing(principal, uri string, timeout time.Duration) bool
}

type defaultContext struct {
	conn      *websocket.Conn
	message   *Message
	principal string
	match     []string
	data      JSON
	parent    LoggingSubscriptionManager
}

func NewContext(conn *websocket.Conn, message *Message, principal string, parent LoggingSubscriptionManager) Context {
	return &defaultContext{
		conn:      conn,
		message:   message,
		principal: principal,
		parent:    parent,
	}
}

func (self *defaultContext) Conn() *websocket.Conn {
	return self.conn
}

func (self *defaultContext) Message() *Message {
	return self.message
}

func (self *defaultContext) Principal() string {
	return self.principal
}

func (self *defaultContext) Match() []string {
	return self.match
}

func (self *defaultContext) SetMatch(m []string) {
	self.match = m
}

func (self *defaultContext) Data() JSON {
	return self.data
}

func (self *defaultContext) SetData(j JSON) {
	self.data = j
}

func (self *defaultContext) Fatalf(format string, args ...interface{}) {
	self.parent.Fatalf(format, args...)
}

func (self *defaultContext) Errorf(format string, args ...interface{}) {
	self.parent.Errorf(format, args...)
}

func (self *defaultContext) Infof(format string, args ...interface{}) {
	self.parent.Infof(format, args...)
}

func (self *defaultContext) Debugf(format string, args ...interface{}) {
	self.parent.Debugf(format, args...)
}

func (self *defaultContext) Tracef(format string, args ...interface{}) {
	self.parent.Tracef(format, args...)
}

func (self *defaultContext) IsSubscribing(principal, uri string, timeout time.Duration) bool {
	return self.parent.IsSubscribing(principal, uri, timeout)
}
