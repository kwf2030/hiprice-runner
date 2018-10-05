package cdp

import (
  "io/ioutil"
  "net/http"
  "sync"
  "sync/atomic"

  "github.com/gorilla/websocket"
)

type Params map[string]interface{}

type Result map[string]interface{}

type Message struct {
  // 请求的ID，响应中会带有相同的ID，
  // 事件通知没有该字段
  Id int32 `json:"id,omitempty"`

  // 请求、响应和事件通知都有该字段
  Method string `json:"method,omitempty"`

  // 请求的参数（可选）、事件通知的数据（可选），
  // 响应没有该字段
  Params Params `json:"params,omitempty"`

  // 响应数据（请求和事件通知没有该字段）
  Result Result `json:"result,omitempty"`

  // 是否是异步请求（只有请求有该字段）
  async bool `json:"-"`

  // 同步请求需要通知的channel（只有请求有该字段）
  syncChan chan *Message `json:"-"`
}

type tabMeta struct {
  Description          string `json:"description"`
  DevtoolsFrontendUrl  string `json:"devtoolsFrontendUrl"`
  FaviconUrl           string `json:"faviconUrl"`
  Id                   string `json:"id"`
  Title                string `json:"title"`
  Type                 string `json:"type"`
  Url                  string `json:"url"`
  WebSocketDebuggerUrl string `json:"webSocketDebuggerUrl"`
}

type Tab struct {
  // ChromeDevToolsProtocol的API地址，
  // http://host:port/json
  endpoint string

  meta *tabMeta

  conn *websocket.Conn

  // 每次请求ID都会自增
  id int32

  // 非零表示Tab已经关闭
  closed int32

  // 广播channel，用于关闭WebSocket读写的channel
  closeChan chan struct{}

  // WebSocket上发送数据监听的channel
  sendChan chan *Message

  // WebSocket上读取到的数据经过处理后发送给该channel
  C chan *Message

  // 存放两类数据：
  // 1.订阅的事件，key是method（string-->bool），用于过滤WebSocket读取到的事件，
  // 2.请求对应的响应Message，key是请求Id（int64-->*Message），用于将请求和响应关联起来
  eventsAndCalls sync.Map
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
    if _, ok := t.eventsAndCalls.Load(msg.Method); ok {
      t.C <- msg
    }
    return
  }
  // Message.id非0表示响应，
  // 找到对应的请求，同步异步分别处理
  if v, ok := t.eventsAndCalls.Load(msg.Id); ok {
    if req, ok := v.(*Message); ok {
      t.eventsAndCalls.Delete(msg.Id)
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
  id := atomic.AddInt32(&t.id, 1)
  ch := make(chan *Message)
  msg := &Message{
    Id:       id,
    Method:   method,
    Params:   p,
    syncChan: ch,
  }
  t.eventsAndCalls.Store(id, msg)
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
  id := atomic.AddInt32(&t.id, 1)
  msg := &Message{
    Id:     id,
    Method: method,
    Params: p,
    async:  true,
  }
  t.eventsAndCalls.Store(id, msg)
  t.sendChan <- msg
}

func (t *Tab) Subscribe(method string) {
  if method != "" {
    t.eventsAndCalls.Store(method, true)
  }
}

func (t *Tab) Unsubscribe(method string) {
  if method != "" {
    t.eventsAndCalls.Delete(method)
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
  // 只要调用过Close，就把Tab.closed标识设为1，
  // 防止一个Tab多次调用Close
  if !atomic.CompareAndSwapInt32(&t.closed, 0, 1) {
    return
  }
  close(t.closeChan)
  close(t.C)
  t.eventsAndCalls.Range(func(k, v interface{}) bool {
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
