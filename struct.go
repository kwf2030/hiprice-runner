package main

import (
  "time"
)

const (
  Unknown = iota

  // 淘宝
  TaoBao

  // 天猫
  TMall

  // 京东
  JingDong

  // 亚马逊（中国）
  AmazonCN

  // 亚马逊（日本）
  AmazonJP

  // 亚马逊（美国）
  AmazonUS

  // 亚马逊（英国）
  AmazonEN

  // 亚马逊（德国）
  AmazonDE

  // 唯品会
  WeiPinHui

  // 蘑菇街
  MoGuJie

  // 聚美优品
  JuMei

  // 苏宁易购
  SuNing

  // 网易考拉
  KaoLa

  // 网易严选
  YanXuan

  // 小米有品
  YouPin
)

const (
  // 规则配置中没有相关字段的脚本
  NoScript = -1

  // 规则配置中有相关字段的脚本，但没抓到值（表达式有错或解析有错）
  NoValue = -2

  // 表示价格字段是一个区间
  RangePrice = -3
)

type Task struct {
  ID string `json:"id,omitempty"`

  // 任务创建时间，由Dispatcher赋值
  CreateTime time.Time `json:"create_time,omitempty"`

  // 任务报告时间，由Runner赋值
  ReportTime time.Time `json:"report_time,omitempty"`

  // 消息抓完成后先提交（一个Task最多可以分成两次提交），
  // 如果Payloads[i].Message有值，Payloads[i]中的Message和Product一定是对应的
  Payloads []*Payload `json:"payloads,omitempty"`
}

type Payload struct {
  Message *Message `json:"message,omitempty"`
  Product *Product `json:"product,omitempty"`
}

type Message struct {
  ID  string `json:"id,omitempty"`
  URL string `json:"url,omitempty"`

  // 为了减少传输量，如果URL有值，Content就为空
  Content string `json:"content,omitempty"`
}

type Product struct {
  ID       string `json:"id,omitempty"`
  URL      string `json:"url,omitempty"`
  ShortURL string `json:"short_url,omitempty"`
  Source   int    `json:"source,omitempty"`
  Title    string `json:"title,omitempty"`

  // 价格单位，
  // 0:RMB, 1:JPY, 2:USD, 3:GBP, 4:EUR
  Currency int `json:"currency,omitempty"`

  // 0：价格为0（基本不存在这种情况，但亚马逊的电子书可能搞活动限时免费），
  // -1：规则配置中没有价格字段，
  // -2：没抓到值（表达式有错或解析有错），
  // -3：商品的价格是一个区间，即[PriceLow,PriceHigh]
  Price float64 `json:"price,omitempty"`

  // 价格区间
  PriceLow  float64 `json:"price_low,omitempty"`
  PriceHigh float64 `json:"price_high,omitempty"`

  // 库存，不是所有平台都有库存，
  // 10000000：有货但没有数量（如亚马逊只显示现在有货，只有在库存不足时才显示仅剩xx件），
  // 0：库存为0（已售完/下架等），
  // -1：规则配置中没有库存字段，
  // -2：没抓到值（表达式有错或解析有错）
  Stock int `json:"stock,omitempty"`

  // 销量，不是所有平台都有销量，
  // 并且销量的单位可能不一样，例如淘宝天猫是月销量，蘑菇街是总销量，业务中自行处理，
  // 0：销量为0，
  // -1：规则配置中没有销量字段，
  // -2：没抓到值（表达式有错或解析有错）
  Sales int `json:"sales,omitempty"`

  // 商品分类，以下划线分隔
  Category string `json:"category,omitempty"`

  // 评论统计，不是所有平台都有评论
  Comments Comments `json:"comments,omitempty"`

  // 抓取时间
  UpdateTime time.Time `json:"update_time,omitempty"`
}

func NewProduct() *Product {
  // PriceLow/PriceHigh/Comments下的默认值没有用NoScript或NoValue是因为他们都依赖于某个属性（Price或Total），
  // 离开了这个属性，这些字段本身就没有意义，所以没有必要初始化成NoScript或NoValue
  return &Product{
    Price: NoScript,
    Stock: NoScript,
    Sales: NoScript,
    Comments: Comments{
      Total: NoScript,
    },
  }
}

type Comments struct {
  // 评论总数（对于1000+和2.7万+这类概数就算1000和27000，不会影响整体数据），
  // 0：评论总数为0（没有评论），
  // -1：规则配置中没有评论字段，
  // -2：没抓到值（表达式有错或解析有错）
  Total int `json:"total,omitempty"`

  // 亚马逊是按星级评价的，从1-5分成5个等级，
  // 为了统一存储形式，其他平台的好评/中评/差评，分别用5/3/1表示
  Star5 int `json:"star5,omitempty"`
  Star4 int `json:"star4,omitempty"`
  Star3 int `json:"star3,omitempty"`
  Star2 int `json:"star2,omitempty"`
  Star1 int `json:"star1,omitempty"`

  // 图片/追评，淘宝/天猫/京东等有这两种评论
  Image  int `json:"image,omitempty"`
  Append int `json:"append,omitempty"`
}
