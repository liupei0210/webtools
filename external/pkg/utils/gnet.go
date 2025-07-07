package utils

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/panjf2000/gnet/v2"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	websocketPrefix = "GET /"
	maxMessageSize  = 1024 * 1024 // 1MB
)

// GNetUtil 网络工具结构体
type GNetUtil struct {
	// 配置选项
	config *GNetConfig
}

// GNetConfig 配置结构体
type GNetConfig struct {
	MaxMessageSize   int64
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	HandshakeTimeout time.Duration
	ReaderSize       int
}

// GNetUtilOption 配置选项函数类型
type GNetUtilOption func(*GNetConfig)

// WithMaxMessageSize 设置最大消息大小
func WithMaxMessageSize(size int64) GNetUtilOption {
	return func(c *GNetConfig) {
		c.MaxMessageSize = size
	}
}

// WithTimeouts 设置超时时间
func WithTimeouts(handshake time.Duration) GNetUtilOption {
	return func(c *GNetConfig) {
		c.HandshakeTimeout = handshake
	}
}
func WithReaderSize(size int) GNetUtilOption {
	return func(c *GNetConfig) {
		c.ReaderSize = size
	}
}

// NewGNetUtil 创建新的GNetUtil实例
func NewGNetUtil(opts ...GNetUtilOption) *GNetUtil {
	config := &GNetConfig{
		MaxMessageSize:   maxMessageSize,
		HandshakeTimeout: time.Second * 10,
		ReaderSize:       4096,
	}

	for _, opt := range opts {
		opt(config)
	}

	return &GNetUtil{config: config}
}

// NewWsCtx 创建WebSocket上下文
func (g *GNetUtil) NewWsCtx() GnetContext {
	return &WSContext{
		config: g.config,
	}
}

// NewTcpCtx 创建TCP上下文
func (g *GNetUtil) NewTcpCtx(c gnet.Conn) GnetContext {
	return &TCPContext{
		config: g.config,
		conn:   c,
	}
}

// IsWsConn 判断是否为WebSocket连接
func (g *GNetUtil) IsWsConn(c gnet.Conn) (bool, error) {
	prefix, err := c.Peek(5)
	if err != nil {
		return false, fmt.Errorf("peek connection failed: %v", err)
	}
	return bytes.HasPrefix(prefix, []byte(websocketPrefix)), nil
}

// GnetContext 网络上下文接口
type GnetContext interface {
	GetType() string
	Close() error
	Write(data []byte) error
	Conn() gnet.Conn
}

// TCPContext TCP上下文实现
type TCPContext struct {
	conn   gnet.Conn
	config *GNetConfig
	mutex  sync.Mutex
}

func (t *TCPContext) GetType() string {
	return "tcp"
}

func (t *TCPContext) Close() error {
	return t.conn.Close()
}
func (t *TCPContext) Conn() gnet.Conn {
	return t.conn
}
func (t *TCPContext) Write(data []byte) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	_, err := t.conn.Write(data)
	return err
}

// WSContext WebSocket上下文实现
type WSContext struct {
	upgraded  bool
	curHeader *ws.Header
	cachedBuf bytes.Buffer
	opCode    *ws.OpCode
	config    *GNetConfig
	conn      gnet.Conn
	PongState bool
	mutex     sync.Mutex
	headers   http.Header // 存储HTTP Header
	query     url.Values  // 存储Query参数
}

func (w *WSContext) GetType() string {
	return "ws"
}

func (w *WSContext) Close() error {
	return w.conn.Close()
}
func (w *WSContext) Conn() gnet.Conn {
	return w.conn
}
func (w *WSContext) Write(data []byte) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if !w.upgraded {
		return errors.New("connection not upgraded")
	}

	return wsutil.WriteServerText(w.conn, data)
}

// GetHeaders 获取HTTP Header
func (w *WSContext) GetHeaders() http.Header {
	return w.headers
}

// GetQuery 获取Query参数
func (w *WSContext) GetQuery() url.Values {
	return w.query
}

// upgrade WebSocket握手升级
func (w *WSContext) upgrade(c gnet.Conn, fs ...func(ctx *WSContext) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), w.config.HandshakeTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		// 解析HTTP请求
		peek, err := c.Peek(-1)
		if err != nil {
			done <- err
			return
		}
		req, err := http.ReadRequest(bufio.NewReaderSize(bytes.NewReader(peek), w.config.ReaderSize))
		if err != nil {
			done <- fmt.Errorf("read http request failed: %v", err)
			return
		}

		// 保存HTTP Header和Query参数
		w.headers = req.Header
		w.query = req.URL.Query()
		_, err = ws.Upgrade(c)
		if err != nil {
			done <- err
			return
		}
		for _, f := range fs {
			if err = f(w); err != nil {
				done <- err
				return
			}
		}
		GetLogger().Debug("WebSocket upgrade successful")
		done <- nil
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("websocket upgrade failed: %v", err)
		}
		w.upgraded = true
		w.conn = c
		return nil
	case <-ctx.Done():
		return errors.New("websocket upgrade timeout")
	}
}

// read 读取WebSocket消息
func (w *WSContext) read(c gnet.Conn) ([][]byte, error) {
	messages, err := w.readFrame(c)
	if err != nil || messages == nil {
		return nil, err
	}

	var payloads [][]byte
	for _, message := range messages {
		if message.OpCode.IsControl() {
			//心跳处理，如果有设置心跳
			if message.OpCode == ws.OpPong {
				if w.PongState == false {
					w.PongState = true
				}
			}
			if err = wsutil.HandleClientControlMessage(c, message); err != nil {
				GetLogger().Debugf("handle control message error: %v", err)
			}
			continue
		}

		if message.OpCode == ws.OpText || message.OpCode == ws.OpBinary {
			if int64(len(message.Payload)) > w.config.MaxMessageSize {
				return nil, fmt.Errorf("message size exceeds limit: %d > %d", len(message.Payload), w.config.MaxMessageSize)
			}
			payloads = append(payloads, message.Payload)
		}
	}
	return payloads, nil
}

// HandleWsTraffic 处理WebSocket流量
func (g *GNetUtil) HandleWsTraffic(c gnet.Conn, handler func(message []byte), httpBusinessHandlers ...func(ctx *WSContext) error) error {
	ctx, ok := c.Context().(*WSContext)
	if !ok {
		return errors.New("invalid websocket context")
	}

	if c.InboundBuffered() <= 0 {
		return nil
	}

	if !ctx.upgraded {
		if err := ctx.upgrade(c, httpBusinessHandlers...); err != nil {
			return err
		}
	}

	messages, err := ctx.read(c)
	if err != nil {
		return err
	}

	for _, message := range messages {
		handler(message)
	}
	return nil
}

// readFrame 读取WebSocket帧
func (w *WSContext) readFrame(c gnet.Conn) ([]wsutil.Message, error) {
	var messages []wsutil.Message

	for {
		// 读取头部
		if w.curHeader == nil {
			if c.InboundBuffered() < ws.MinHeaderSize {
				return messages, nil
			}

			header, err := ws.ReadHeader(c)
			if err != nil {
				if err == io.EOF || errors.Is(err, io.ErrUnexpectedEOF) {
					return messages, nil
				}
				return nil, fmt.Errorf("read header failed: %v", err)
			}

			// 检查消息大小
			if header.Length > w.config.MaxMessageSize {
				return nil, fmt.Errorf("message too large: %d > %d", header.Length, w.config.MaxMessageSize)
			}

			w.curHeader = &header
			if w.opCode == nil {
				w.opCode = &header.OpCode
			}
		}

		// 读取消息体
		if w.curHeader.Length > 0 {
			dataLength := int(w.curHeader.Length)
			if c.InboundBuffered() < dataLength {
				return messages, nil
			}

			peek, err := c.Peek(dataLength)
			if err != nil {
				if err == io.EOF || errors.Is(err, io.ErrUnexpectedEOF) {
					return messages, nil
				}
				return nil, fmt.Errorf("peek data failed: %v", err)
			}

			// 解密消息
			cipherReader := wsutil.NewCipherReader(bytes.NewReader(peek), w.curHeader.Mask)
			if _, err = io.CopyN(&w.cachedBuf, cipherReader, w.curHeader.Length); err != nil {
				return nil, fmt.Errorf("decrypt message failed: %v", err)
			}

			if _, err = c.Discard(dataLength); err != nil {
				return nil, fmt.Errorf("discard data failed: %v", err)
			}
		}

		// 处理完整消息
		if w.curHeader.Fin {
			messages = append(messages, wsutil.Message{
				OpCode:  *w.opCode,
				Payload: w.cachedBuf.Bytes(),
			})
			w.cachedBuf.Reset()
			w.opCode = nil
		}

		w.curHeader = nil

		// 检查是否还有更多数据
		if c.InboundBuffered() < ws.MinHeaderSize {
			break
		}
	}

	return messages, nil
}
