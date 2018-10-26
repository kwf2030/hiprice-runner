package main

import (
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
  Match    []string
  Fields   []Field
}

type Field struct {
  Name   string
  Alias  string
  Value  string
  Script string
  Async  bool
  Cmd    string
}

func UpdateRules(data []byte) error {
  if len(data) == 0 {
    return nil
  }
  rules := make([]*Rule, 0, 16)
  e := yaml.Unmarshal(data, &rules)
  if e != nil {
    return e
  }
  if allRules == nil {
    allRules = make(map[string][]*Rule, 16)
  }
  for _, r := range rules {
    if _, ok := allRules[r.Group]; !ok {
      allRules[r.Group] = make([]*Rule, 0, 16)
      allRules[r.Group] = append(allRules[r.Group], r)
    } else {
      index := -1
      for i, v := range allRules[r.Group] {
        if v.Id == r.Id {
          index = i
          break
        }
      }
      if index == -1 {
        allRules[r.Group] = append(allRules[r.Group], r)
      } else {
        old := allRules[r.Group][index]
        if old.Version < r.Version {
          allRules[r.Group][index] = r
        }
      }
    }
  }
  return nil
}

func removeRules(group string, ids ...string) {
  if group == "" || len(ids) == 0 {
    return
  }
  if _, ok := allRules[group]; !ok {
    return
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
}

func main() {
  removeRules("1", "2")
}
