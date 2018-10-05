package cdp

import (
  "encoding/json"
  "errors"
  "fmt"
  "net/http"
  "os/exec"
  "strings"
  "sync"
  "time"
)

var (
  ErrInvalidArgs     = errors.New("invalid args")
  ErrInvalidResponse = errors.New("invalid response")
)

type Chrome string

func LaunchChrome(bin string, args ...string) (Chrome, error) {
  if bin == "" {
    return "", ErrInvalidArgs
  }
  _, e := exec.LookPath(bin)
  if e != nil {
    return "", e
  }
  var port string
  for _, v := range args {
    if strings.Contains(v, "--remote-debugging-port") {
      arr := strings.Split(v, "=")
      if len(arr) != 2 {
        return "", ErrInvalidArgs
      }
      port = strings.TrimSpace(arr[1])
      break
    }
  }
  if port == "" {
    port = "9222"
    args = append(args, fmt.Sprintf("--remote-debugging-port=%s", port))
  }
  exec.Command(bin, args...).Start()
  // 等待Chrome启动（1秒是估值，性能差的机器可能不够），
  // 如果不等待，则在LaunchChrome后立即调用Chrome.CreateTab可能会出现空指针（Chrome的Server没启动完成），
  // 所以最好提前调用LaunchChrome，先把Chrome启动起来再做其他初始化工作
  time.Sleep(time.Second)
  return Chrome(fmt.Sprintf("http://127.0.0.1:%s/json", port)), nil
}

func AttachToChrome(host string, port int) (Chrome, error) {
  if host == "" || port < 0 {
    return "", ErrInvalidArgs
  }
  return Chrome(fmt.Sprintf("http://%s:%d/json", host, port)), nil
}

func (c Chrome) NewTab() (*Tab, error) {
  endpoint := string(c)
  meta := &tabMeta{}
  resp, e := http.Get(endpoint + "/new")
  if e != nil {
    return nil, e
  }
  e = json.NewDecoder(resp.Body).Decode(meta)
  resp.Body.Close()
  if e != nil {
    return nil, e
  }
  if meta.Id == "" || meta.WebSocketDebuggerUrl == "" {
    return nil, ErrInvalidResponse
  }
  t := &Tab{
    endpoint:       endpoint,
    meta:           meta,
    closeChan:      make(chan struct{}),
    sendChan:       make(chan *Message, 2),
    C:              make(chan *Message, 4),
    eventsAndCalls: sync.Map{},
  }
  t.conn, e = t.wsConnect()
  if e != nil {
    return nil, e
  }
  go t.wsRead()
  go t.wsWrite()
  return t, nil
}
