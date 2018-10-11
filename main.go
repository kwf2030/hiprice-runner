package main

import (
  "encoding/json"
  "errors"
  "fmt"
  "io/ioutil"
  "os"
  "os/signal"
  "runtime"
  "strings"
  "time"

  "github.com/kwf2030/commons/beanstalk"
  "github.com/kwf2030/commons/boltdb"
  "github.com/kwf2030/commons/cdp"
  "github.com/kwf2030/commons/httputil"
  "github.com/kwf2030/commons/times"
  "github.com/rs/zerolog"
)

const Version = "1.0.1"

var (
  // 保存所有抓过的商品，
  // 每个商品在抓之前会先查询是否在指定的时间段内（例如6小时内）已经抓过
  bucketProducts = []byte("product")

  // 定时取任务
  loopChan = make(chan struct{})

  // 日志文件，以天为单位，每天自动创建新的文件
  logFile *os.File
  logger  *zerolog.Logger

  store  *boltdb.Store
  chrome cdp.Chrome

  // Beanstalk的任务ID，抓完之后要删除
  jobID string

  conn *beanstalk.Conn
)

func main2() {
  file := "conf.yaml"
  if len(os.Args) == 2 {
    file = os.Args[1]
  }
  e := LoadConf(file)
  if e != nil {
    panic(e)
  }
  e = LoadRules(Conf.Task.Rules)
  if e != nil {
    panic(e)
  }

  initLogger()
  defer logFile.Close()
  logger.Info().Msg("Hiprice Runner " + Version)

  initStore()
  defer store.Close()

  initChrome()
  defer func() {
    tab, e := chrome.NewTab()
    if e == nil {
      tab.CallAsync(cdp.Browser.Close)
    }
  }()

  initBeanstalk()
  defer conn.Quit()

  go run()
  loopChan <- struct{}{}

  s := make(chan os.Signal, 1)
  signal.Notify(s, os.Interrupt)
  <-s
}

func initLogger() {
  dir := Conf.Log.Dir
  e := os.MkdirAll(dir+"/dump", os.ModePerm)
  if e != nil {
    panic(e)
  }
  l := zerolog.DebugLevel
  switch strings.ToLower(Conf.Log.Level) {
  case "info":
    l = zerolog.InfoLevel
  case "warn":
    l = zerolog.WarnLevel
  case "error":
    l = zerolog.ErrorLevel
  case "fatal":
    l = zerolog.FatalLevel
  case "disable":
    l = zerolog.Disabled
  }
  zerolog.SetGlobalLevel(l)
  zerolog.TimeFieldFormat = ""
  if logFile != nil {
    logFile.Close()
  }
  logFile, _ = os.Create(fmt.Sprintf("%s/runner_%s.log", dir, times.NowStrFormat(times.DateFormat3)))
  lg := zerolog.New(logFile).Level(l).With().Timestamp().Logger()
  logger = &lg
  now := times.Now()
  next := now.Add(time.Hour * 24)
  next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
  time.AfterFunc(next.Sub(now), func() {
    logger.Info().Msg("create log file")
    go initLogger()
  })
}

func initStore() {
  var e error
  store, e = boltdb.Open("runner.db", "product")
  if e != nil {
    panic(e)
  }
}

func initChrome() {
  var c Chrome
  switch runtime.GOOS {
  case "windows":
    c = Conf.Chrome.Windows
  case "linux":
    c = Conf.Chrome.Linux
  default:
    panic(errors.New("platform not supported"))
  }
  var e error
  chrome, e = cdp.LaunchChrome(c.Exec, c.Args...)
  if e != nil {
    panic(e)
  }
  tab, _ := chrome.NewTab()
  defer tab.Close()
  msg := tab.Call(cdp.Browser.GetVersion)
  logger.Info().Msg(msg.Result["product"].(string))
}

func initBeanstalk() {
  var e error
  for i := 0; i < 3; i++ {
    conn, e = beanstalk.Dial(Conf.Beanstalk.Host, Conf.Beanstalk.Port)
    if e != nil {
      logger.Info().Msg("beanstalk connect failed, will retry 30 seconds later")
      time.Sleep(time.Second * 30)
      continue
    }
    conn.Use(Conf.Beanstalk.PutTube)
    conn.Watch(Conf.Beanstalk.ReserveTube)
    conn.Ignore("default")
    conn.EnableHeartbeat(Conf.Beanstalk.Heartbeat)
    break
  }
  if conn == nil {
    panic(e)
  }
}

func run() {
  // 外层循环是定时任务
  for range loopChan {
    // 内层循环是一直取任务直到没有为止
    for {
      t := reserveTask()
      if t == nil || len(t.Payloads) == 0 {
        break
      }
      ch := make(chan *Product, 1)
      go func() {
        for p := range ch {
          data, _ := json.Marshal(p)
          store.UpdateV(bucketProducts, []byte(p.ID), data)
        }
      }()
      messages := make([]*Message, 0, len(t.Payloads))
      products := make([]*Product, 0, len(t.Payloads))
      for _, payload := range t.Payloads {
        if payload == nil {
          continue
        }
        if payload.Message != nil && payload.Message.ID != "" {
          messages = append(messages, payload.Message)
        } else if payload.Product != nil && payload.Product.URL != "" {
          products = append(products, payload.Product)
        }
      }
      logger.Info().Msgf("%d messages, %d products", len(messages), len(products))
      if len(messages) > 0 {
        payloads := processMessages(ch, messages)
        task := &Task{
          ID:         t.ID,
          CreateTime: t.CreateTime,
          ReportTime: times.Now(),
          Payloads:   payloads,
        }
        reportMessages(task)
      }
      if len(products) > 0 {
        payloads := processProducts(ch, products)
        task := &Task{
          ID:         t.ID,
          CreateTime: t.CreateTime,
          ReportTime: times.Now(),
          Payloads:   payloads,
        }
        reportProducts(task)
      }
      e := conn.Delete(jobID)
      if e != nil {
        logger.Error().Err(e).Msg("ERR: Delete")
      }
      jobID = ""
      close(ch)
    }
    scheduleNextTime()
  }
}

func scheduleNextTime() {
  logger.Info().Msg("schedule next time")
  time.AfterFunc(time.Minute*time.Duration(Conf.Task.PollingInterval), func() {
    loopChan <- struct{}{}
  })
}

func reserveTask() *Task {
  var e error
  var job []byte
  jobID, job, e = conn.ReserveWithTimeout(Conf.Beanstalk.ReserveTimeout)
  if e != nil {
    if e != beanstalk.ErrTimedOut {
      logger.Error().Err(e).Msg("ERR: ReserveWithTimeout")
    }
    return nil
  }
  t := &Task{}
  e = json.Unmarshal(job, t)
  if e != nil {
    logger.Error().Err(e).Msg("ERR: Unmarshal")
    return nil
  }
  dump(fmt.Sprintf("%s/dump/%s_reserve.json", Conf.Log.Dir, t.ID), job)
  logger.Info().Msgf("check task, ok, jobID=%s, taskID=%s, count=%d", jobID, t.ID, len(t.Payloads))
  return t
}

func processMessages(ch chan<- *Product, arr []*Message) []*Payload {
  payloads := make([]*Payload, len(arr))
  // i为重试的次数，j为实际抓取的数量
  i, j := 0, 0
  for {
    if i >= Conf.Task.CrawlRetry {
      break
    }
    if i != 0 {
      time.Sleep(time.Second * 10)
    }
    i++
    left := false
    logger.Info().Msgf("[%d]process messages", i)
    for n, m := range arr {
      if m == nil {
        continue
      }
      payload := payloads[n]
      if payload != nil && payload.Product != nil && payload.Product.ID != "" && payload.Product.Price != NoValue {
        continue
      }
      p := crawlMessage(m)
      // 返回nil表示发生了不可恢复的错误（如提取不到链接）
      if p == nil {
        continue
      }
      // 返回空Product表示发生了可恢复的错误（如超时），可能重试一次就好了
      if p.ID == "" {
        left = true
        continue
      }
      if p.Price == NoValue || (p.Price == RangePrice && p.PriceLow == 0 && p.PriceHigh == 0) {
        left = true
        continue
      }
      p.ShortURL = shortenURL(p.URL)
      if p.ShortURL == "" {
        logger.Warn().Msg("get short url failed")
      }
      p.UpdateTime = times.Now()
      payloads[n] = &Payload{Message: m, Product: p}
      ch <- p
      j++
      if p.Price == RangePrice {
        logger.Debug().Msgf("id=%s, price=[%.2f, %.2f]", p.ID, p.PriceLow, p.PriceHigh)
      } else {
        logger.Debug().Msgf("id=%s, price=%.2f", p.ID, p.Price)
      }
    }
    if !left {
      break
    }
  }
  logger.Info().Msgf("process messages, ok, tried %d times, %d messages processed", i, j)
  ret := make([]*Payload, 0, len(arr))
  for _, v := range payloads {
    if v != nil {
      ret = append(ret, v)
    }
  }
  return ret
}

func reportMessages(task *Task) {
  var e error
  data, _ := json.Marshal(task)
  dump(fmt.Sprintf("%s/dump/%s_report_messages.json", Conf.Log.Dir, task.ID), data)
  _, e = conn.Put(Conf.Beanstalk.PutPriority, Conf.Beanstalk.PutDelay, Conf.Beanstalk.PutTTR, data)
  if e != nil {
    logger.Error().Err(e).Msg("ERR: Put")
    panic(e)
  }
  logger.Info().Msg("report messages, ok")
}

func processProducts(ch chan<- *Product, arr []*Product) []*Payload {
  payloads := make([]*Payload, len(arr))
  // i为重试的次数，j为实际抓取的数量
  i, j := 0, 0
  for {
    if i >= Conf.Task.CrawlRetry {
      break
    }
    if i != 0 {
      time.Sleep(time.Second * 10)
    }
    i++
    left := false
    logger.Info().Msgf("[%d]process products", i)
    for n, m := range arr {
      if m == nil {
        continue
      }
      payload := payloads[n]
      if payload != nil && payload.Product != nil && payload.Product.ID != "" && payload.Product.Price != NoValue {
        continue
      }
      if Conf.Task.CrawlDuration > 0 {
        duplicate := false
        store.QueryV(bucketProducts, []byte(m.ID), func(k, v []byte, n int) error {
          product := Product{}
          json.Unmarshal(v, &product)
          duplicate = times.Now().Sub(product.UpdateTime).Minutes() < float64(Conf.Task.CrawlDuration)
          return nil
        })
        if duplicate {
          continue
        }
      }
      p := crawlProduct(m)
      // 返回nil表示发生了不可恢复的错误
      if p == nil {
        continue
      }
      // 返回空Product表示发生了可恢复的错误（如超时），可能重试一次就好了
      if p.ID == "" || (p.ID != "" && m.ID != "" && p.ID != m.ID) {
        left = true
        continue
      }
      if p.Price == NoValue || (p.Price == RangePrice && p.PriceLow == 0 && p.PriceHigh == 0) {
        left = true
        continue
      }
      p.ShortURL = shortenURL(p.URL)
      if p.ShortURL == "" {
        logger.Warn().Msg("get short url failed")
      }
      p.UpdateTime = times.Now()
      payloads[n] = &Payload{Product: p}
      ch <- p
      j++
      if p.Price == RangePrice {
        logger.Debug().Msgf("id=%s, price=[%.2f, %.2f]", p.ID, p.PriceLow, p.PriceHigh)
      } else {
        logger.Debug().Msgf("id=%s, price=%.2f", p.ID, p.Price)
      }
    }
    if !left {
      break
    }
  }
  logger.Info().Msgf("process products, ok, tried %d times, %d products processed", i, j)
  ret := make([]*Payload, 0, len(arr))
  for _, v := range payloads {
    if v != nil {
      ret = append(ret, v)
    }
  }
  return ret
}

func reportProducts(task *Task) {
  var e error
  data, _ := json.Marshal(task)
  dump(fmt.Sprintf("%s/dump/%s_report_products.json", Conf.Log.Dir, task.ID), data)
  _, e = conn.Put(Conf.Beanstalk.PutPriority, Conf.Beanstalk.PutDelay, Conf.Beanstalk.PutTTR, data)
  if e != nil {
    logger.Error().Err(e).Msg("ERR: Put")
    panic(e)
  }
  logger.Info().Msg("report products, ok")
}

func shortenURL(addr string) string {
  for i := 0; i < 3; i++ {
    r := httputil.ShortenURL(addr)
    if r != "" {
      return r
    }
    time.Sleep(time.Second)
  }
  return ""
}

func dump(file string, data []byte) {
  if file == "" || len(data) == 0 {
    return
  }
  ioutil.WriteFile(file, data, os.ModePerm)
}
