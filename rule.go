package main

import (
  "errors"
  "regexp"
  "sort"
  "sync"
  "time"

  "gopkg.in/yaml.v2"
)

var (
  errInvalidArgs = errors.New("invalid args")
  errNoRuleGroup = errors.New("no rule group")

  capacity = 16

  allRules = &Rules{groups: make(map[string][]*Rule, capacity)}
)

type Rules struct {
  groups map[string][]*Rule
  sync.RWMutex
}

func (r *Rules) Match(group, url string) *Rule {
  if group == "" || url == "" {
    return nil
  }
  r.RLock()
  defer r.RUnlock()
  rules, ok := r.groups[group]
  if !ok {
    return nil
  }
  for _, rule := range rules {
    for _, m := range rule.Patterns {
      if m.Regex.MatchString(url) {
        return rule
      }
    }
  }
  return nil
}

func (r *Rules) Update(group string, data []byte) error {
  if group == "" || len(data) == 0 {
    return errInvalidArgs
  }
  rules := make([]*Rule, 0, capacity)
  e := yaml.Unmarshal(data, &rules)
  if e != nil {
    return e
  }
  r.Lock()
  defer r.Unlock()
  if _, ok := r.groups[group]; !ok {
    r.groups[group] = make([]*Rule, 0, capacity)
  }
  for _, rule := range rules {
    if rule.Group != group {
      continue
    }
    index := -1
    for i, old := range r.groups[group] {
      if old.Id == rule.Id {
        index = i
        break
      }
    }
    if index == -1 {
      r.groups[group] = append(r.groups[group], rule)
    } else {
      old := r.groups[group][index]
      if old.Version < rule.Version {
        r.groups[group][index] = rule
      }
    }
    initRule(rule)
  }
  sort.SliceStable(r.groups[group], func(i, j int) bool {
    return r.groups[group][i].Priority < r.groups[group][j].Priority
  })
  return nil
}

func (r *Rules) Remove(group string, ids ...string) error {
  if group == "" || len(ids) == 0 {
    return errInvalidArgs
  }
  r.Lock()
  defer r.Unlock()
  if _, ok := r.groups[group]; !ok {
    return errNoRuleGroup
  }
  for _, id := range ids {
    index := -1
    for i, r := range r.groups[group] {
      if id == r.Id {
        index = i
        break
      }
    }
    if index != -1 {
      r.groups[group] = append(r.groups[group][:index], r.groups[group][index+1:]...)
    }
  }
  return nil
}

func initRule(rule *Rule) {
  rule.Patterns = make([]*Pattern, 0, len(rule.PatternsConf))
  for _, pattern := range rule.PatternsConf {
    rule.Patterns = append(rule.Patterns, &Pattern{
      Content: pattern,
      Regex:   regexp.MustCompile(pattern),
    })
  }
  if rule.Prepare != nil && rule.Prepare.WaitWhenReadyConf != "" {
    var e error
    rule.Prepare.WaitWhenReady, e = time.ParseDuration(rule.Prepare.WaitWhenReadyConf)
    if e != nil {
      rule.Prepare.WaitWhenReady = time.Second
    }
  }
  if rule.Collector != nil {
    if len(rule.Collector.Once) > 0 {
      var e error
      for _, v := range rule.Collector.Once {
        if v.WaitConf != "" {
          v.Wait, e = time.ParseDuration(v.WaitConf)
          if e != nil {
            v.Wait = time.Second
          }
        }
      }
    }
    if rule.Collector.Loop != nil {
      var e error
      if rule.Collector.Loop.Prepare != nil && rule.Collector.Loop.Prepare.WaitWhenReadyConf != "" {
        rule.Collector.Loop.Prepare.WaitWhenReady, e = time.ParseDuration(rule.Collector.Loop.Prepare.WaitWhenReadyConf)
        if e != nil {
          rule.Collector.Loop.Prepare.WaitWhenReady = time.Second
        }
      }
      if rule.Collector.Loop.WaitConf != "" {
        rule.Collector.Loop.Wait, e = time.ParseDuration(rule.Collector.Loop.WaitConf)
        if e != nil {
          rule.Collector.Loop.Wait = time.Second
        }
      }
    }
  }
}

type Rule struct {
  Id           string     `yaml:"id"`
  Version      int        `yaml:"version"`
  Name         string     `yaml:"name"`
  Alias        string     `yaml:"alias"`
  Group        string     `yaml:"group"`
  Priority     int        `yaml:"priority"`
  PatternsConf []string   `yaml:"patterns"`
  Patterns     []*Pattern `yaml:"-"`
  Prepare      *Prepare   `yaml:"prepare"`
  Collector    *Collector `yaml:"collector"`
}

type Pattern struct {
  Content string
  Regex   *regexp.Regexp
}

type Prepare struct {
  Eval              string        `yaml:"eval"`
  WaitWhenReadyConf string        `yaml:"wait_when_ready"`
  WaitWhenReady     time.Duration `yaml:"-"`
}

type Collector struct {
  PageLoadTimeoutConf string           `yaml:"page_load_timeout"`
  PageLoadTimeout     time.Duration    `yaml:"-"`
  Once                []*OnceCollector `yaml:"once"`
  Loop                *LoopCollector   `yaml:"loop"`
}

type OnceCollector struct {
  Field    string        `yaml:"field"`
  Alias    string        `yaml:"alias"`
  Value    string        `yaml:"value"`
  Eval     string        `yaml:"eval"`
  Export   bool          `yaml:"export"`
  WaitConf string        `yaml:"wait"`
  Wait     time.Duration `yaml:"-"`
}

type LoopCollector struct {
  Field       string        `yaml:"field"`
  Alias       string        `yaml:"alias"`
  ExportCycle int           `yaml:"export_cycle"`
  Prepare     *Prepare      `yaml:"prepare"`
  Eval        string        `yaml:"eval"`
  Next        string        `yaml:"next"`
  WaitConf    string        `yaml:"wait"`
  Wait        time.Duration `yaml:"-"`
  Break       string        `yaml:"break"`
}
