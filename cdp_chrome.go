package main

import (
  "encoding/json"
  "errors"
  "fmt"
  "net/http"
  "os/exec"
  "strings"
  "sync"
)

type Chrome string

func LaunchInstance(bin string, args ...string) (Chrome, error) {
  if bin == "" {
    return "", errors.New("empty <bin>")
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
        return "", errors.New("invalid '--remote-debugging-port'")
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
  // LaunchInstance返回后立即调用Chrome.NewTab可能会出现空指针，
  // 因为浏览器尚未启动完毕，
  // 所以需要延迟一段时间等待浏览器启动完毕后再调用Chrome.NewTab，
  // 最好是提前调用LaunchInstance，先把浏览器启动起来再做其他初始化工作
  return Chrome(fmt.Sprintf("http://127.0.0.1:%s/json", port)), nil
}

func AttachInstance(host string, port int) (Chrome, error) {
  if host == "" || port <= 0 {
    return "", errors.New("invalid <host>/<port>")
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
  t := &Tab{
    endpoint:          endpoint,
    meta:              meta,
    closeChan:         make(chan struct{}),
    sendChan:          make(chan *Message, 2),
    C:                 make(chan *Message, 4),
    eventsAndMessages: sync.Map{},
  }
  t.conn, e = t.wsConnect()
  if e != nil {
    return nil, e
  }
  go t.wsRead()
  go t.wsWrite()
  return t, nil
}
