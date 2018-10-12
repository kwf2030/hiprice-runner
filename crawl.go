package main

import (
  "encoding/json"
  "encoding/xml"
  "html"
  "strconv"
  "strings"
  "time"
  "unicode"

  "github.com/kwf2030/commons/cdp"
  "github.com/kwf2030/commons/conv"
)

const space = rune(' ')

type msgXml struct {
  XMLName xml.Name `xml:"msg"`
  AppMsg  struct {
    Title string `xml:"title"`
    Desc  string `xml:"des"`
    URL   string `xml:"url"`
  } `xml:"appmsg"`
}

func crawlMessage(m *Message) *Product {
  logger.Debug().Msgf("crawl message %s", m.ID)
  if m.URL != "" {
    return doCrawl(normalizeURL(html.UnescapeString(m.URL)))
  }
  if m.Content == "" {
    return nil
  }
  var addr string
  if len(m.Content) >= 7 && m.Content[:7] == "&lt;msg" {
    v := &msgXml{}
    e := xml.Unmarshal([]byte(html.UnescapeString(m.Content)), v)
    if e != nil {
      return nil
    }
    if v.AppMsg.URL != "" {
      addr = html.UnescapeString(v.AppMsg.URL)
    } else {
      addr = findURLFromText(html.UnescapeString(v.AppMsg.Title))
      if addr == "" {
        addr = findURLFromText(html.UnescapeString(v.AppMsg.Desc))
      }
    }
  } else {
    addr = findURLFromText(html.UnescapeString(m.Content))
  }
  if addr == "" {
    return nil
  }
  return doCrawl(normalizeURL(addr))
}

func crawlProduct(p *Product) *Product {
  logger.Debug().Msgf("crawl product %s", p.ID)
  if p.URL == "" {
    return nil
  }
  return doCrawl(normalizeURL(html.UnescapeString(p.URL)))
}

func findURLFromText(text string) string {
  l := strings.Index(text, "http")
  if l == -1 {
    l = strings.Index(text, "www")
    if l == -1 {
      return ""
    }
  }
  h := len(text)
  for i, c := range text[l:] {
    if c == space || unicode.Is(unicode.Han, c) {
      h = l + i
      break
    }
  }
  return text[l:h]
}

func findRuleByURL(addr string) *rule {
  for _, r := range Rules {
    for i := range r.Match {
      if r.MatchRegex[i].MatchString(addr) {
        return r
      }
    }
  }
  return nil
}

func findChainByURL(addr string) (*rule, *chain) {
  rule := findRuleByURL(addr)
  if rule == nil {
    return nil, nil
  }
  for _, chain := range rule.Chain {
    for i := range chain.Match {
      if chain.MatchRegex[i].MatchString(addr) {
        return rule, chain
      }
    }
  }
  return rule, nil
}

func matchURLFromChain(addr string, chain *chain) string {
  if chain == nil {
    return ""
  }
  arr := chain.IndexRegex.FindStringSubmatchIndex(addr)
  if len(arr) == chain.IndexCount {
    result := make([]byte, 0, chain.Alloc)
    result = chain.IndexRegex.ExpandString(result, chain.Template, addr, arr)
    return string(result)
  }
  return ""
}

func normalizeURL(addr string) (string, *rule, *chain) {
  rule, chain := findChainByURL(addr)
  if rule != nil {
    if chain == nil {
      return addr, rule, nil
    }
    str := matchURLFromChain(addr, chain)
    if str != "" {
      return str, rule, chain
    }
  }
  var addr1, addr2 string
  // 返回的是document.URL和chain表达式（如果有）计算的结果（都是URL，优先使用addr1）
  addr1, addr2, rule, chain = evalURL(addr)
  if chain == nil {
    return addr1, rule, nil
  }
  str := matchURLFromChain(addr1, chain)
  if str != "" {
    return str, rule, chain
  }
  str = matchURLFromChain(addr2, chain)
  if str != "" {
    return str, rule, chain
  }
  return addr, rule, nil
}

func evalURL(addr string) (string, string, *rule, *chain) {
  var addr1, addr2 string
  var rule *rule
  var chain *chain
  done := make(chan struct{})
  tab, _ := chrome.NewTab()
  tab.Subscribe(cdp.Page.LoadEventFired)
  tab.Call(cdp.Page.Enable)
  tab.Call(cdp.Page.Navigate, cdp.Params{"url": addr})
  go func() {
    time.Sleep(time.Second * 2)
    for msg := range tab.C {
      if msg.Method != cdp.Page.LoadEventFired {
        continue
      }
      params := cdp.Params{"objectGroup": "console", "includeCommandLineAPI": true}
      params["expression"] = "document.URL"
      v1 := tab.Call(cdp.Runtime.Evaluate, params)
      addr1 = conv.String(conv.Map(v1.Result, "result"), "value")
      if addr1 == "" {
        break
      }
      rule, chain = findChainByURL(addr1)
      if chain == nil {
        break
      }
      if chain.Script != "" && chain.ScriptTemplate != "" {
        params["expression"] = chain.Script
        v2 := tab.Call(cdp.Runtime.Evaluate, params)
        addr2 = strings.Replace(chain.ScriptTemplate, "$id", conv.String(conv.Map(v2.Result, "result"), "value"), -1)
        if addr2 == "" {
          break
        }
        rule, chain = findChainByURL(addr2)
      }
      break
    }
    close(done)
  }()
  select {
  case <-time.After(time.Second * time.Duration(Conf.Task.CrawlTimeout)):
    tab.C <- &cdp.Message{Method: cdp.Page.LoadEventFired}
    <-done
  case <-done:
  }
  tab.Close()
  return addr1, addr2, rule, chain
}

func matchIDFromRule(addr string, rule *rule) string {
  if rule == nil {
    return ""
  }
  for i := range rule.ID.Match {
    if arr := rule.ID.MatchRegex[i].FindStringSubmatch(addr); len(arr) > rule.ID.Index {
      return arr[rule.ID.Index]
    }
  }
  return ""
}

func doCrawl(addr string, rule *rule, _ *chain) *Product {
  p := NewProduct()
  done := make(chan struct{})
  tab, _ := chrome.NewTab()
  tab.Subscribe(cdp.Page.LoadEventFired)
  tab.Call(cdp.Page.Enable)
  tab.Call(cdp.Page.Navigate, cdp.Params{"url": addr})
  go func() {
    params := cdp.Params{"objectGroup": "console", "includeCommandLineAPI": true}
    for msg := range tab.C {
      if msg.Method != cdp.Page.LoadEventFired {
        continue
      }
      id := matchIDFromRule(addr, rule)
      if id == "" {
        break
      }
      p.ID = id
      p.URL = addr
      p.Source = rule.Source
      p.Currency = rule.Currency
      for _, v := range rule.Scripts {
        params["expression"] = strings.Replace(v.Script, "$id", id, -1)
        if v.Async {
          tab.CallAsync(cdp.Runtime.Evaluate, params)
        } else {
          r := tab.Call(cdp.Runtime.Evaluate, params)
          s := conv.String(conv.Map(r.Result, "result"), "value")
          handle(rule.Source, v.Name, s, p)
        }
        if v.Sleep > 0 {
          time.Sleep(time.Millisecond * time.Duration(v.Sleep))
        }
      }
      break
    }
    close(done)
  }()
  select {
  case <-time.After(time.Second * time.Duration(Conf.Task.CrawlTimeout)):
    logger.Debug().Msg("crawl timeout, execute expression")
    tab.C <- &cdp.Message{Method: cdp.Page.LoadEventFired}
    <-done
  case <-done:
    logger.Debug().Msg("crawl done")
  }
  tab.Close()
  return p
}

func handle(_ int, name string, value string, p *Product) {
  switch name {
  case "title":
    p.Title = value

  case "price":
    arr := parsePrice(value)
    if len(arr) == 1 {
      p.Price = arr[0]
    } else if len(arr) == 2 {
      p.Price = RangePrice
      p.PriceLow = arr[0]
      p.PriceHigh = arr[1]
    }

  case "stock":
    p.Stock = atoi(value)

  case "sales":
    p.Sales = atoi(value)

  case "category":
    p.Category = value

  case "comments":
    m := make(map[string]string, 6)
    e := json.Unmarshal([]byte(value), &m)
    if e != nil {
      p.Comments.Total = NoValue
      break
    }
    if v, ok := m["total"]; ok && v != "" {
      p.Comments.Total = atoi(v)
    } else {
      p.Comments.Total = NoValue
      break
    }
    if v, ok := m["star5"]; ok && v != "" {
      p.Comments.Star5 = atoi(v)
    }
    if v, ok := m["star4"]; ok && v != "" {
      p.Comments.Star4 = atoi(v)
    }
    if v, ok := m["star3"]; ok && v != "" {
      p.Comments.Star3 = atoi(v)
    }
    if v, ok := m["star2"]; ok && v != "" {
      p.Comments.Star2 = atoi(v)
    }
    if v, ok := m["star1"]; ok && v != "" {
      p.Comments.Star1 = atoi(v)
    }
    if v, ok := m["image"]; ok && v != "" {
      p.Comments.Image = atoi(v)
    }
    if v, ok := m["append"]; ok && v != "" {
      p.Comments.Append = atoi(v)
    }
  }
}

// string转int，主要是数量的转换，如库存和销量，
// 数量可能存在小数点和万，如1.4万，
// 已经在JS中去掉加号、逗号和空格了
func atoi(value string) int {
  if value == "" {
    return NoValue
  }
  init := float64(1)
  if strings.Contains(value, "万") {
    value = strings.TrimRight(value, "万")
    init = 10000
  }
  v, e := strconv.ParseFloat(value, 64)
  if e != nil {
    return NoValue
  }
  return int(v * init)
}

// string转float64，价格转换，
// 已经在JS中去掉货币符号、逗号和空格了
func atof(value string) float64 {
  if value == "" {
    return NoValue
  }
  v, e := strconv.ParseFloat(value, 64)
  if e != nil {
    return NoValue
  }
  return v
}

// 解析价格，
// 如果是唯一价格，返回的len(slice)是1，
// 如果是区间，返回的len(slice)是2
func parsePrice(value string) []float64 {
  ret := make([]float64, 0, 2)
  if !strings.Contains(value, "-") && !strings.Contains(value, "~") {
    ret = append(ret, atof(value))
    return ret
  }
  arr := strings.Split(value, "-")
  l := len(arr)
  if l != 2 {
    arr = strings.Split(value, "~")
    l = len(arr)
    if l != 2 {
      ret = append(ret, NoValue)
      return ret
    }
  }
  ret = append(ret, atof(arr[0]), atof(arr[1]))
  return ret
}
