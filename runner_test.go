package main

import (
  "fmt"
  "testing"
)

func doInit() {
  LoadConf("conf.yaml")
  LoadRules(Conf.Task.Rules)
  initLogger()
  initStore()
  initChrome()
}

func TestURL(t *testing.T) {
  doInit()
  urls := []string{
    `https://item.jd.com/11929332775.html`,
    `https://item.m.jd.com/product/10597630681.html`,
    `https://item.taobao.com/item.htm?id=568399568669&ali_trackid=2:mm_121371575_20858889_70606584:1530370781_308_336547984&spm=a21bo.7925826.192013.3.5c284c0d8v7NUD`,
    `https://h5.m.taobao.com/need/weex/container.html?_wx_tpl=https://owl.alicdn.com/mupp-dy/develop/taobao/need/weex/bpu/entry.js&itemId=566756880269&bpuId=958408059&spm=a21to.8287046.mgbp.1&_wx_appbar=true`,
    `http://market.m.taobao.com/app/dinamic/h5-tb-detail/index.html?id=563557797907&scm=1007.12144.95804.100200300000000&pg1stepk=sdm:50630_item_563557797907&spm=a2114u.9628.2.1`,
  }
  for _, v := range urls {
    addr, _, _ := normalizeURL(v)
    t.Log(addr)
  }
}

func TestCrawl(t *testing.T) {
  doInit()
  urls := []string{
    `https://www.amazon.cn/dp/B06XKCV7X9/ref=cngwdyfloorv2_recs_0?pf_rd_p=3aeea79d-b33f-46f8-8020-d2edee624402&pf_rd_s=desktop-2&pf_rd_t=36701&pf_rd_i=desktop&pf_rd_m=A1AJ19PSB66TGU&pf_rd_r=G98F6MD6MF7T8873BEP8&pf_rd_r=G98F6MD6MF7T8873BEP8&pf_rd_p=3aeea79d-b33f-46f8-8020-d2edee624402`,
    `https://item.jd.com/25610083103.html#crumb-wrap`,
    `http://item.jumei.com/949379.html?from=store_inoherb_index_14_100081_4&site=sh`,
    `https://goods.kaola.com/product/2003008.html?ri=33544&rt=product&zid=zid_2312850582&zp=product29&zn=&isMarketPriceShow=true&hcAntiCheatSwitch=0&anstipamActiCheatSwitch=1&anstipamActiCheatToken=de3223456456fa2e3324354u4567lt&anstipamActiCheatValidate=anstipam_acti_default_validate`,
    `http://shop.mogujie.com/detail/1kbk3ie?acm=3.ms.9_4_1kbk3ie.150.16292-68998.8dDqvqWJ9HSmF.sd_117_116-swt_150-imt_6-t_8dDqvqWJ9HSmF-dit_23&ptp=1.shop_list_12345.0.0.043xuF4o`,
    `https://product.suning.com/0000000000/10320170644.html`,
    `https://item.taobao.com/item.htm?spm=a21bo.2017.201876.35.5af911d9QRBwQR&scm=1007.12493.92624.100200300000001&id=549226118434&pvid=2f816e1d-5b1d-4038-8104-227d8fee2add`,
    `https://detail.tmall.com/item.htm?id=564998912180&spm=a21bo.2017.201876.5.5af911d9QRBwQR&scm=1007.12493.92624.100200300000001&pvid=2f816e1d-5b1d-4038-8104-227d8fee2add`,
    `https://detail.vip.com/detail-2939221-566720643.html`,
  }
  for i, v := range urls {
    addr, rule, chain := normalizeURL(v)
    p := doCrawl(addr, rule, chain)
    if p == nil {
      t.Logf("nil(%d)\n", i)
      continue
    }
    if p.ID == "" || p.Price == NoValue {
      t.Logf("empty(%d)\n", i)
      continue
    }
    if p.Price == RangePrice {
      t.Log(fmt.Sprintf("ID=[%s], price=[%.2f,%.2f], comments=[%d,%d,%d,%d,%d,%d]\n", p.ID, p.PriceLow, p.PriceHigh, p.Comments.Total, p.Comments.Star5, p.Comments.Star3, p.Comments.Star1, p.Comments.Image, p.Comments.Append))
    } else {
      t.Log(fmt.Sprintf("ID=[%s], price=[%.2f], comments=[%d,%d,%d,%d,%d,%d]\n", p.ID, p.Price, p.Comments.Total, p.Comments.Star5, p.Comments.Star3, p.Comments.Star1, p.Comments.Image, p.Comments.Append))
    }
  }
}
