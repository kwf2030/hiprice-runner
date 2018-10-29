package main

import (
  "errors"
  "sort"
  "time"

  "gopkg.in/yaml.v2"
)

var allRules map[string][]*Rule

type Rule struct {
  Id       string
  Version  int
  Name     string
  Alias    string
  Group    string
  Priority int
  Output   *Output
  Match    []string
  Prepare  string
  Collect  *Collect
  Loop     *Loop
}

type Output struct {
  Format     string
  File       string
  DBHost     string
  DBPort     int
  DBUser     string
  DBPassword string
  DNName     string
  DBTable    string
  Endpoint   string
}

type Collect struct {
  Field   string
  Alias   string
  Value   string
  Eval    string
  Async   bool
  WaitStr string
  Wait    time.Duration
}

type Loop struct {
  Field            string
  Alias            string
  OutputCycle      int
  Prepare          string
  WaitWhenReadyStr string
  WaitWhenReady    time.Duration
  Eval             string
  Next             string
  WaitStr          string
  Wait             time.Duration
  Break            string
}

func UpdateRules(group string, data []byte) error {
  if group == "" || len(data) == 0 {
    return errors.New("empty arguments")
  }
  capacity := 16
  if allRules == nil {
    allRules = make(map[string][]*Rule, capacity)
  }
  rules := make([]*Rule, 0, capacity)
  e := yaml.Unmarshal(data, &rules)
  if e != nil {
    return e
  }
  if _, ok := allRules[group]; !ok {
    allRules[group] = make([]*Rule, 0, capacity)
  }
  for _, r := range rules {
    if r.Group != group {
      continue
    }
    index := -1
    for i, old := range allRules[group] {
      if old.Id == r.Id {
        index = i
        break
      }
    }
    if index == -1 {
      allRules[group] = append(allRules[group], r)
    } else {
      old := allRules[group][index]
      if old.Version < r.Version {
        allRules[group][index] = r
      }
    }
  }
  sort.SliceStable(allRules[group], func(i, j int) bool {
    return allRules[group][i].Priority < allRules[group][j].Priority
  })
  return nil
}

func removeRules(group string, ids ...string) error {
  if group == "" || len(ids) == 0 {
    return errors.New("empty arguments")
  }
  if _, ok := allRules[group]; !ok {
    return errors.New("no such group")
  }
  for _, id := range ids {
    index := -1
    for i, r := range allRules[group] {
      if id == r.Id {
        index = i
        break
      }
    }
    if index != -1 {
      allRules[group] = append(allRules[group][:index], allRules[group][index+1:]...)
    }
  }
  return nil
}
