package gosubs

import "code.google.com/p/go.net/websocket"

type defaultContext struct {
	conn      *websocket.Conn
	message   *Message
	principal string
	match     []string
	data      JSON
	logger    Logger
}

func NewContext(conn *websocket.Conn, message *Message, principal string, logger Logger) Context {
	return &defaultContext{
		conn:      conn,
		message:   message,
		principal: principal,
		logger:    logger,
	}
}

func (self *defaultContext) Clean() *defaultContext {
	self.conn = nil
	self.message = nil
	self.data = JSON{nil}
	self.match = nil
	return self
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
	self.logger.Fatalf(format, args...)
}

func (self *defaultContext) Errorf(format string, args ...interface{}) {
	self.logger.Errorf(format, args...)
}

func (self *defaultContext) Infof(format string, args ...interface{}) {
	self.logger.Infof(format, args...)
}

func (self *defaultContext) Debugf(format string, args ...interface{}) {
	self.logger.Debugf(format, args...)
}

func (self *defaultContext) Tracef(format string, args ...interface{}) {
	self.logger.Tracef(format, args...)
}
