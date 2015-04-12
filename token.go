package gosubs

import (
	"bytes"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"time"
)

/*
Token handles authentication of the sockets connecting to a router
*/
type Token struct {
	Principal string
	Timeout   time.Time
	Hash      []byte
	Encoded   string
}

/*
GetHash calculates a hash for this Token
*/
func (self *Token) GetHash(secret string) (result []byte, err error) {
	h := sha512.New()
	if _, err = h.Write([]byte(fmt.Sprintf("%#v,%v,%#v", self.Principal, self.Timeout.UnixNano(), secret))); err != nil {
		return
	}
	result = h.Sum(nil)
	return
}

/*
Encode will set the Encoded field for this Token to an encoded version of itself
*/
func (self *Token) Encode(secret string) (err error) {
	self.Encoded = ""
	if self.Hash, err = self.GetHash(secret); err != nil {
		return
	}
	buf := &bytes.Buffer{}
	baseEnc := base64.NewEncoder(base64.URLEncoding, buf)
	gobEnc := gob.NewEncoder(baseEnc)
	if err = gobEnc.Encode(self); err != nil {
		return
	}
	if err = baseEnc.Close(); err != nil {
		return
	}
	self.Encoded = buf.String()
	return
}

/*
DecodeToken unpacks s into a Token, that it validates (hash and timeout) and returns
*/
func DecodeToken(secret, s string) (result *Token, err error) {
	dec := gob.NewDecoder(base64.NewDecoder(base64.URLEncoding, bytes.NewBufferString(s)))
	tok := &Token{}
	if err = dec.Decode(tok); err != nil {
		return
	}
	if tok.Timeout.Before(time.Now()) {
		err = fmt.Errorf("Token %+v is timed out", tok)
		return
	}
	correctHash, err := tok.GetHash(secret)
	if err != nil {
		return
	}
	if len(tok.Hash) != len(correctHash) || subtle.ConstantTimeCompare(correctHash, tok.Hash) != 1 {
		err = fmt.Errorf("Token %+v has incorrect hash")
		return
	}
	result = tok
	return
}
