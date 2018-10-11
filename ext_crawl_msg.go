package main

import (
  "database/sql"
  "encoding/json"
  "fmt"
  "strings"
  "time"

  "github.com/go-sql-driver/mysql"
  "github.com/kwf2030/commons/times"
)

const db = "127.0.0.1:3306"
const dbu = "root"
const dbp = "root"
const dbn = "test"

func doInit() {
  LoadConf("conf.yaml")
  LoadRules(Conf.Task.Rules)
  initLogger()
  initStore()
  initChrome()
}

func connectMariaDB() *sql.DB {
  c := mysql.NewConfig()
  c.Net = "tcp"
  c.Addr = fmt.Sprintf(db)
  c.Collation = "utf8mb4_unicode_ci"
  c.User = dbu
  c.Passwd = dbp
  c.DBName = dbn
  c.Loc = times.TimeZoneSH
  c.ParseTime = true
  db, e := sql.Open("mysql", c.FormatDSN())
  if e != nil {
    panic(e)
  }
  e = db.Ping()
  if e != nil {
    panic(e)
  }
  return db
}

func loadMessages(db *sql.DB, start, limit int) []*Message {
  if limit <= 0 {
    return nil
  }
  ret := make([]*Message, 0, limit)
  rows, e := db.Query(`SELECT _id, id, content, url FROM msg WHERE _id>? AND (type=1 OR type=49) LIMIT ?`, start, limit)
  if e != nil {
    return nil
  }
  var aid uint64
  for rows.Next() {
    msg := &Message{}
    e := rows.Scan(&aid, &msg.ID, &msg.Content, &msg.URL)
    if e != nil || (msg.Content == "" && msg.URL == "") {
      continue
    }
    ret = append(ret, msg)
  }
  rows.Close()
  for _, msg := range ret {
    fmt.Printf("\n%7s:%d\n%7s:%s\n%7s:%s\nContent:%s\n", "_id", aid, "id", msg.ID, "url", msg.URL, msg.Content)
    fmt.Println("\nEnter Real URL:")
    var addr string
    fmt.Scanln(&addr)
    addr = strings.TrimSpace(addr)
    if len(addr) > 0 {
      msg.URL = addr
    }
    fmt.Println(ret[0].URL)
  }
  return ret
}

func update(db *sql.DB, payloads []*Payload) {
  tx, _ := db.Begin()
  defer tx.Commit()
  for _, payload := range payloads {
    msg := payload.Message
    p := payload.Product
    if p == nil || p.ID == "" || p.Price == NoScript || p.Price == NoValue {
      continue
    }
    aid := 0
    if msg != nil && msg.ID != "" {
      var uid string
      var ct time.Time
      tx.QueryRow(`SELECT from_user_id, create_time FROM msg WHERE id=? LIMIT 1`, msg.ID).Scan(&uid, &ct)
      if uid != "" {
        wt := ct.Format(times.DateTimeSFormat)
        var state int
        tx.QueryRow(`SELECT _id, state FROM product_watch WHERE user_id=? AND product_id=? LIMIT 1`, uid, p.ID).Scan(&aid, &state)
        if aid == 0 {
          tx.Exec(`INSERT INTO product_watch (user_id, product_id, currency, price, price_low, price_high, stock, watch_time, state) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
            uid, p.ID, p.Currency, p.Price,
            p.PriceLow, p.PriceHigh, p.Stock, wt, 0)
        } else if state == 1 {
          tx.Exec(`UPDATE product_watch SET watch_time=?, state=? WHERE user_id=? AND product_id=?`, wt, 0, uid, p.ID)
        }
      }
    }

    ut := p.UpdateTime.Format(times.DateTimeSFormat)
    var price, priceLow, priceHigh float64 = NoValue, 0, 0
    var stock int
    tx.QueryRow(`SELECT price, price_low, price_high, stock FROM product_update WHERE id=? ORDER BY update_time DESC LIMIT 1`, p.ID).Scan(&price, &priceLow, &priceHigh, &stock)
    if validateChanged(p, price, priceLow, priceHigh) {
      var comments string
      if p.Comments.Total > 0 {
        data, _ := json.Marshal(p.Comments)
        comments = string(data)
      }
      tx.Exec(`INSERT INTO product_update (id, source, url, short_url, title, currency, price, price_low, price_high, stock, sales, category, comments, update_time) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
        p.ID, p.Source, p.URL, p.ShortURL, p.Title,
        p.Currency, p.Price, p.PriceLow, p.PriceHigh, p.Stock,
        p.Sales, p.Category, comments, ut)

      aid = 0
      tx.QueryRow(`SELECT _id FROM product WHERE id=? LIMIT 1`, p.ID).Scan(&aid)
      if aid == 0 {
        tx.Exec(`INSERT INTO product (id, source, url, short_url, title, currency, price, price_low, price_high, stock, sales, category, comments, update_time, last_dispatch_time) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
          p.ID, p.Source, p.URL, p.ShortURL, p.Title,
          p.Currency, p.Price, p.PriceLow, p.PriceHigh, p.Stock,
          p.Sales, p.Category, comments, ut, ut)
      } else {
        tx.Exec(`UPDATE product SET source=?, url=?, short_url=?, title=?, currency=?, price=?, price_low=?, price_high=?, stock=?, sales=?, category=?, comments=?, update_time=? WHERE id=?`,
          p.Source, p.URL, p.ShortURL, p.Title, p.Currency,
          p.Price, p.PriceLow, p.PriceHigh, p.Stock, p.Sales,
          p.Category, comments, ut, p.ID)
      }
    }
  }
}

func validateChanged(p *Product, price, priceLow, priceHigh float64) bool {
  if price == NoValue {
    return true
  }
  if price == RangePrice && p.Price == RangePrice {
    if priceLow < 0 || priceHigh < 0 || p.PriceLow < 0 || p.PriceHigh < 0 {
      return false
    }
    return priceLow != p.PriceLow || priceHigh != p.PriceHigh
  }
  if price >= 0 && p.Price >= 0 {
    return price != p.Price
  }
  return false
}

func main() {
  doInit()
  db := connectMariaDB()
  arr := loadMessages(db, 0, 100)
  fmt.Printf("load %d messages\n", len(arr))
  if len(arr) == 0 {
    return
  }
  payloads := make([]*Payload, 0, len(arr))
  for _, m := range arr {
    p := crawlMessage(m)
    if p == nil || p.ID == "" || p.Price == NoScript || p.Price == NoValue {
      continue
    }
    payloads = append(payloads, &Payload{Message: m, Product: p})
  }
  fmt.Printf("%d messages crawled\n", len(payloads))
  update(db, payloads)
  fmt.Printf("done")
}
