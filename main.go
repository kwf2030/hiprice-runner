package main

import (
  "fmt"
  "io/ioutil"
  "os"
  "os/signal"
  "runtime"
  "time"

  "github.com/kwf2030/commons/cdp"
  "github.com/kwf2030/commons/times"
  "github.com/nats-io/go-nats"
  "github.com/rs/zerolog"
  "gopkg.in/yaml.v2"
)

const Version = "2.0.0"

var (
  logFile *os.File
  logger  *zerolog.Logger

  chrome cdp.Chrome

  conn *nats.Conn
)

func main() {
  initLogger()
  defer logFile.Close()
  logger.Info().Msg("Hiprice Runner " + Version)

  initChrome()
  defer func() {
    tab, e := chrome.NewTab()
    if e == nil {
      tab.CallAsync(cdp.Browser.Close)
    }
  }()

  initNats()
  defer conn.Close()

  s := make(chan os.Signal, 1)
  signal.Notify(s, os.Interrupt)
  <-s
}

func initLogger() {
  e := os.MkdirAll("log", os.ModePerm)
  if e != nil {
    panic(e)
  }
  zerolog.SetGlobalLevel(zerolog.InfoLevel)
  zerolog.TimeFieldFormat = ""
  if logFile != nil {
    logFile.Close()
  }
  logFile, _ = os.Create(fmt.Sprintf("log/runner_%s.log", times.NowStrFormat(times.DateFormat3)))
  lg := zerolog.New(logFile).Level(zerolog.InfoLevel).With().Timestamp().Logger()
  logger = &lg
  now := times.Now()
  next := now.Add(time.Hour * 24)
  next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
  time.AfterFunc(next.Sub(now), func() {
    logger.Info().Msg("create log file")
    go initLogger()
  })
}

func initChrome() {
  var e error
  c := &struct {
    Exec string
    Args []string
  }{}
  f := "chrome.yml"
  _, e = os.Stat(f)
  if e == nil {
    data, err := ioutil.ReadFile(f)
    if err == nil {
      yaml.Unmarshal(data, c)
    }
  }
  if c.Exec == "" {
    switch runtime.GOOS {
    case "windows":
      c.Exec = "C:/Program Files (x86)/Google/Chrome/Application/chrome.exe"
    case "linux":
      c.Exec = "/usr/bin/google-chrome-stable"
    }
  }
  chrome, e = cdp.LaunchChrome(c.Exec, c.Args...)
  if e != nil {
    panic(e)
  }
  tab, _ := chrome.NewTab()
  defer tab.Close()
  msg := tab.Call(cdp.Browser.GetVersion)
  logger.Info().Msg(msg.Result["product"].(string))
}

func initNats() {
  var e error
  conn, e = nats.Connect(nats.DefaultURL)
  if e != nil {
    panic(e)
  }
  // 请求规则的队列，规则更新的队列
  // 获取任务的队列，提交任务的队列
}
