package main

import (
  "io/ioutil"
  "os"
  "path/filepath"
  "regexp"

  "gopkg.in/yaml.v2"
)

var Rules []*rule

type rule struct {
  Name       string           `yaml:"name"`
  Source     int              `yaml:"source"`
  Currency   int              `yaml:"currency"`
  Match      []string         `yaml:"match"`
  MatchRegex []*regexp.Regexp `yaml:"-"`
  Chain      []*chain         `yaml:"chain"`
  ID         *id              `yaml:"id"`
  Scripts    []*script        `yaml:"scripts"`
}

type chain struct {
  Match          []string         `yaml:"match"`
  MatchRegex     []*regexp.Regexp `yaml:"-"`
  Index          string           `yaml:"index"`
  IndexRegex     *regexp.Regexp   `yaml:"-"`
  IndexCount     int              `yaml:"index_count"`
  Template       string           `yaml:"template"`
  Alloc          int              `yaml:"alloc"`
  Script         string           `yaml:"script"`
  ScriptTemplate string           `yaml:"script_template"`
}

type id struct {
  Match      []string         `yaml:"match"`
  MatchRegex []*regexp.Regexp `yaml:"-"`
  Index      int              `yaml:"index"`
}

type script struct {
  Name   string `yaml:"name"`
  Script string `yaml:"script"`
  Async  bool   `yaml:"async"`
  Sleep  int    `yaml:"sleep"`
}

func LoadRules(dir string) error {
  Rules = make([]*rule, 0, 10)
  return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
    if path == dir {
      return nil
    }
    if info.Name()[:1] == "." {
      if info.IsDir() {
        return filepath.SkipDir
      }
      return nil
    }
    ext := filepath.Ext(path)
    if ext == ".yaml" || ext == ".yml" {
      r, e := loadRule(path)
      if e != nil {
        return e
      }
      Rules = append(Rules, r)
    }
    return nil
  })
}

func loadRule(file string) (*rule, error) {
  data, e := ioutil.ReadFile(file)
  if e != nil {
    return nil, e
  }
  ret := &rule{}
  e = yaml.Unmarshal(data, ret)
  if e != nil {
    return nil, e
  }
  ret.MatchRegex = make([]*regexp.Regexp, len(ret.Match))
  for i, m := range ret.Match {
    ret.MatchRegex[i] = regexp.MustCompile(m)
  }
  for _, c := range ret.Chain {
    c.MatchRegex = make([]*regexp.Regexp, len(c.Match))
    for i, m := range c.Match {
      c.MatchRegex[i] = regexp.MustCompile(m)
    }
    c.IndexRegex = regexp.MustCompile(c.Index)
  }
  ret.ID.MatchRegex = make([]*regexp.Regexp, len(ret.ID.Match))
  for i, m := range ret.ID.Match {
    ret.ID.MatchRegex[i] = regexp.MustCompile(m)
  }
  return ret, nil
}
