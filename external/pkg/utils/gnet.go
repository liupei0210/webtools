package utils

import (
	"bytes"
	"errors"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/panjf2000/gnet/v2"
	"io"
	"strings"
)

const websocketPrefix = "GET /"

type GNetUtil struct {
}

var gnetUtil = GNetUtil{}

func GetGnetUtil() GNetUtil {
	return gnetUtil
}

func (GNetUtil) NewWsCtx() GnetContext {
	return &wsCtx{}
}
func (GNetUtil) NewTcpCtx() GnetContext {
	return &tcpCtx{}
}
func (GNetUtil) IsWsConn(c gnet.Conn) (bool, error) {
	prefix, err := c.Peek(5)
	if err != nil {
		return false, err
	}
	return strings.HasPrefix(string(prefix), websocketPrefix), nil
}

type GnetContext interface {
	GetType() string
}

type tcpCtx struct {
}

func (*tcpCtx) GetType() string {
	return "tcp"
}

type wsCtx struct {
	upgraded  bool
	curHeader *ws.Header
	cachedBuf bytes.Buffer
}

func (w *wsCtx) GetType() string {
	return "ws"
}
func (w *wsCtx) upgrade(c gnet.Conn) (err error) {
	var peek []byte
	peek, err = c.Peek(-1)
	if err != nil {
		return
	}
	reader := bytes.NewReader(peek)
	_, err = ws.Upgrade(struct {
		io.Reader
		io.Writer
	}{reader, c})
	if err != nil {
		err = w.handleEOFError(err)
		return
	}
	_, err = c.Discard(c.InboundBuffered() - reader.Len())
	if err != nil {
		return
	}
	w.upgraded = true
	return
}
func (w *wsCtx) read(c gnet.Conn) (payloads [][]byte, err error) {
	messages, err := w.readFrame(c)
	if err != nil || messages == nil || len(messages) <= 0 {
		return
	}
	for _, message := range messages {
		if message.OpCode.IsControl() {
			err = wsutil.HandleClientControlMessage(c, message)
			if err != nil {
				return
			}
			continue
		}
		if message.OpCode == ws.OpText || message.OpCode == ws.OpBinary {
			payloads = append(payloads, message.Payload)
		}
	}
	return
}
func (w *wsCtx) readFrame(c gnet.Conn) (messages []wsutil.Message, err error) {
	for {
		if w.curHeader == nil {
			if c.InboundBuffered() < ws.MinHeaderSize {
				return
			}
			var header ws.Header
			var peek []byte
			peek, err = c.Peek(-1)
			if err != nil {
				return
			}
			reader := bytes.NewReader(peek)
			header, err = ws.ReadHeader(reader)
			if err != nil {
				err = w.handleEOFError(err)
				return
			}
			_, err = c.Discard(c.InboundBuffered() - reader.Len())
			if err != nil {
				return
			}
			w.curHeader = &header
			err = ws.WriteHeader(&w.cachedBuf, header)
			if err != nil {
				return
			}
		}
		dataLen := (int)(w.curHeader.Length)
		if dataLen > 0 {
			if c.InboundBuffered() < dataLen {
				return
			}
			_, err = io.CopyN(&w.cachedBuf, c, int64(dataLen))
			if err != nil {
				return
			}

		}
		//处理完整消息
		if w.curHeader.Fin {
			messages, err = wsutil.ReadClientMessage(&w.cachedBuf, messages)
			if err != nil {
				return
			}
			w.cachedBuf.Reset()
		}
		w.curHeader = nil
	}
}
func (w *wsCtx) handleEOFError(err error) error {
	if err == io.EOF || errors.Is(err, io.ErrUnexpectedEOF) {
		err = nil
	}
	return err
}

// HandleWsTraffic The f method you provide needs to handle asynchronous data processing; otherwise, it will lead to server blocking.
func (GNetUtil) HandleWsTraffic(c gnet.Conn, f func(message []byte)) (err error) {
	ctx := c.Context().(*wsCtx)
	if c.InboundBuffered() <= 0 {
		return
	}
	if !ctx.upgraded {
		err = ctx.upgrade(c)
		if err != nil {
			return
		}
	}
	messages, err := ctx.read(c)
	if err != nil || messages == nil {
		return
	}
	for _, message := range messages {
		f(message)
	}
	return
}
