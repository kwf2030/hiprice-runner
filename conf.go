package main

import (
  "io/ioutil"

  "gopkg.in/yaml.v2"
)

var Conf = &struct {
  Log       LogConf       `yaml:"log"`
  Beanstalk BeanstalkConf `yaml:"beanstalk"`
  Chrome    ChromeConf    `yaml:"chrome"`
  Task      TaskConf      `yaml:"task"`
}{}

type LogConf struct {
  Dir   string `yaml:"dir"`
  Level string `yaml:"level"`
}

type BeanstalkConf struct {
  Host           string `yaml:"host"`
  Port           int    `yaml:"port"`
  ReserveTube    string `yaml:"reserve_tube"`
  ReserveTimeout int    `yaml:"reserve_timeout"`
  PutTube        string `yaml:"put_tube"`
  PutPriority    int    `yaml:"put_priority"`
  PutDelay       int    `yaml:"put_delay"`
  PutTTR         int    `yaml:"put_ttr"`
  Heartbeat      int    `yaml:"heartbeat"`
}

type ChromeConf struct {
  Windows Chrome `yaml:"windows"`
  Linux   Chrome `yaml:"linux"`
}

type Chrome struct {
  Exec string   `yaml:"exec"`
  Args []string `yaml:"args"`
}

type TaskConf struct {
  PollingInterval int    `yaml:"polling_interval"`
  Rules           string `yaml:"rules"`
  CrawlDuration   int    `yaml:"crawl_duration"`
  CrawlRetry      int    `yaml:"crawl_retry"`
  CrawlTimeout    int    `yaml:"crawl_timeout"`
}

func LoadConf(file string) error {
  data, e := ioutil.ReadFile(file)
  if e != nil {
    return e
  }
  return yaml.Unmarshal(data, Conf)
}
