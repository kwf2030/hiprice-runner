package main

import (
  "io/ioutil"
  "net/http"
  "sync"
  "sync/atomic"

  "github.com/gorilla/websocket"
)

type Params map[string]interface{}

type Result map[string]interface{}

// 请求/响应/事件通知
type Message struct {
  // 请求的ID，响应中会带有相同的ID，每次请求Tab.lastMessageId自增后赋值给Message.Id，
  // 事件通知没有该字段
  Id int32 `json:"id,omitempty"`

  // 请求、响应和事件通知都有该字段
  Method string `json:"method,omitempty"`

  // 请求的参数（可选）、事件通知的数据（可选），
  // 响应没有该字段
  Params Params `json:"params,omitempty"`

  // 响应数据（请求和事件通知没有该字段）
  Result Result `json:"result,omitempty"`

  // 请求是否等待响应（只有请求有该字段）
  async bool

  // 请求在此channel上等待响应（只有请求有该字段）
  syncChan chan *Message
}

type tabMeta struct {
  Id                   string `json:"id"`
  Type                 string `json:"type"`
  Title                string `json:"title"`
  Url                  string `json:"url"`
  FaviconUrl           string `json:"faviconUrl"`
  Description          string `json:"description"`
  DevtoolsFrontendUrl  string `json:"devtoolsFrontendUrl"`
  WebSocketDebuggerUrl string `json:"webSocketDebuggerUrl"`
}

type Tab struct {
  // ChromeDevToolsProtocol的API地址（http://host:port/json）
  endpoint string

  meta *tabMeta

  conn *websocket.Conn

  // 每次请求自增
  lastMessageId int32

  // 非零表示Tab已经关闭
  closed int32

  // 广播，用于通知WebSocket关闭读写goroutine
  closeChan chan struct{}

  // WebSocket发送数据的channel
  sendChan chan *Message

  // WebSocket读取到的数据经过处理后发送给该channel
  C chan *Message

  // 存放两类数据：
  // 1.订阅的事件（string-->bool），key是Message.Method，用于过滤WebSocket读取到的事件，
  // 2.请求的Message（int32-->*Message），key是Message.Id，用于读取到数据时找到对应的请求Message
  eventsAndMessages sync.Map
}

func (t *Tab) wsConnect() (*websocket.Conn, error) {
  conn, _, e := websocket.DefaultDialer.Dial(t.meta.WebSocketDebuggerUrl, nil)
  if e != nil {
    return nil, e
  }
  return conn, nil
}

func (t *Tab) wsRead() {
  for {
    select {
    case <-t.closeChan:
      return

    default:
      msg := &Message{}
      e := t.conn.ReadJSON(msg)
      if e != nil {
        t.Close()
        return
      }
      t.dispatch(msg)
    }
  }
}

func (t *Tab) wsWrite() {
  for {
    select {
    case <-t.closeChan:
      close(t.sendChan)
      return

    case msg := <-t.sendChan:
      e := t.conn.WriteJSON(msg)
      if e != nil {
        t.Close()
        return
      }
    }
  }
}

func (t *Tab) dispatch(msg *Message) {
  // Message.id为0表示事件通知
  if msg.Id == 0 {
    // 若注册过该类事件，则进行通知
    if _, ok := t.eventsAndMessages.Load(msg.Method); ok {
      t.C <- msg
    }
    return
  }
  // Message.id非0表示响应，
  // 找到对应的请求，同步异步分别处理
  if v, ok := t.eventsAndMessages.Load(msg.Id); ok {
    if req, ok := v.(*Message); ok {
      t.eventsAndMessages.Delete(msg.Id)
      // 响应中没有Method，找到对应的请求，用请求中的method给其赋值
      msg.Method = req.Method
      if req.async {
        msg.async = true
        t.C <- msg
      } else {
        req.syncChan <- msg
        close(req.syncChan)
      }
    }
  }
}

func (t *Tab) Call(method string, params ...Params) *Message {
  if method == "" {
    return nil
  }
  var p Params
  if len(params) > 0 {
    p = params[0]
  }
  id := atomic.AddInt32(&t.lastMessageId, 1)
  ch := make(chan *Message)
  msg := &Message{
    Id:       id,
    Method:   method,
    Params:   p,
    syncChan: ch,
  }
  t.eventsAndMessages.Store(id, msg)
  t.sendChan <- msg
  return <-ch
}

func (t *Tab) CallAsync(method string, params ...Params) {
  if method == "" {
    return
  }
  var p Params
  if len(params) > 0 {
    p = params[0]
  }
  id := atomic.AddInt32(&t.lastMessageId, 1)
  msg := &Message{
    Id:     id,
    Method: method,
    Params: p,
    async:  true,
  }
  t.eventsAndMessages.Store(id, msg)
  t.sendChan <- msg
}

func (t *Tab) Subscribe(method string) {
  if method != "" {
    t.eventsAndMessages.Store(method, true)
  }
}

func (t *Tab) Unsubscribe(method string) {
  if method != "" {
    t.eventsAndMessages.Delete(method)
  }
}

func (t *Tab) Activate() {
  resp, e := http.Get(t.endpoint + "/activate/" + t.meta.Id)
  if e == nil {
    ioutil.ReadAll(resp.Body)
    resp.Body.Close()
  }
}

func (t *Tab) Close() {
  // 只要调用Close，就把Tab.closed标识设为1，
  // 防止Close被多次调用
  if !atomic.CompareAndSwapInt32(&t.closed, 0, 1) {
    return
  }
  close(t.closeChan)
  close(t.C)
  t.eventsAndMessages.Range(func(k, v interface{}) bool {
    if msg, ok := v.(*Message); ok && msg.syncChan != nil {
      close(msg.syncChan)
    }
    return true
  })
  t.conn.Close()
  resp, e := http.Get(t.endpoint + "/close/" + t.meta.Id)
  if e == nil {
    ioutil.ReadAll(resp.Body)
    resp.Body.Close()
  }
}
