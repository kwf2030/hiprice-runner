package cdp

import (
  "context"
  "time"
)

const defaultEvent = "DEFAULT"

type Action interface {
  Method() string

  Params() Params
}

type EvaluateAction interface {
  Action

  Expressions() []string

  Handle(Result) error
}

type SimpleAction struct {
  method string
  params Params
}

func NewAction(method string, params ...Params) *SimpleAction {
  ret := &SimpleAction{method: method}
  if len(params) > 0 {
    ret.params = params[0]
  }
  return ret
}

func (sa *SimpleAction) Method() string {
  return sa.method
}

func (sa *SimpleAction) Params() Params {
  return sa.params
}

type sleepAction time.Duration

func (sa sleepAction) Method() string {
  return ""
}

func (sa sleepAction) Params() Params {
  return nil
}

type SimpleEvaluateAction struct {
  expression string
  handler    func(Result) error
}

func NewEvaluateAction(expression string, handler func(Result) error) *SimpleEvaluateAction {
  return &SimpleEvaluateAction{expression: expression, handler: handler}
}

func (sea *SimpleEvaluateAction) Method() string {
  return Runtime.Evaluate
}

func (sea *SimpleEvaluateAction) Params() Params {
  return Params{"objectGroup": "console", "includeCommandLineAPI": true}
}

func (sea *SimpleEvaluateAction) Expressions() []string {
  return []string{sea.expression}
}

func (sea *SimpleEvaluateAction) Handle(result Result) error {
  if sea.handler != nil {
    return sea.handler(result)
  }
  return nil
}

type Task struct {
  chrome Chrome

  // 一个事件对应N个Action，
  // 没有事件的Action的key为DEFAULT，会优先执行
  actions map[string][]Action

  // 当前事件
  evt string
}

func NewTask(c Chrome) *Task {
  t := &Task{
    chrome:  c,
    actions: make(map[string][]Action, 4),
    evt:     defaultEvent,
  }
  t.actions[defaultEvent] = make([]Action, 0, 2)
  return t
}

func (t *Task) Action(action Action) *Task {
  if action != nil {
    t.actions[t.evt] = append(t.actions[t.evt], action)
  }
  return t
}

func (t *Task) WaitFor(event string) *Task {
  if event != "" {
    if _, ok := t.actions[event]; !ok {
      t.evt = event
      t.actions[event] = make([]Action, 0, 16)
    }
  }
  return t
}

func (t *Task) Sleep(duration time.Duration) *Task {
  return t.Action(sleepAction(duration))
}

func (t *Task) Run(ctx context.Context) {
  tab, e := t.chrome.NewTab()
  if e != nil {
    return
  }
  for event := range t.actions {
    if event != defaultEvent {
      tab.Subscribe(event)
    }
  }
  for _, action := range t.actions[defaultEvent] {
    switch a := action.(type) {
    case sleepAction:
      time.Sleep(time.Duration(a))
    case EvaluateAction:
      t.runEvaluateAction(tab, a)
    default:
      tab.CallAsync(action.Method(), action.Params())
    }
  }
  // 默认加了一个DEFAULT事件，所以Task.actions长度至少为1，
  // 如果大于1，说明有事件要监听（执行对应事件的Action）
  if len(t.actions) > 1 {
    go t.runActions(ctx, tab)
  }
}

func (t *Task) runActions(ctx context.Context, tab *Tab) {
  for {
    select {
    case <-ctx.Done():
      tab.Close()
      return
    case msg := <-tab.C:
      if actions, ok := t.actions[msg.Method]; ok {
        for _, action := range actions {
          switch a := action.(type) {
          case sleepAction:
            time.Sleep(time.Duration(a))
          case EvaluateAction:
            t.runEvaluateAction(tab, a)
          default:
            tab.CallAsync(action.Method(), action.Params())
          }
        }
      }
    }
  }
}

func (t *Task) runEvaluateAction(tab *Tab, action EvaluateAction) {
  method := action.Method()
  params := action.Params()
  if params == nil {
    params = Params{}
  }
  arr := action.Expressions()
  for _, expr := range arr {
    params["expression"] = expr
    msg := tab.Call(method, params)
    var e error
    if msg == nil {
      e = action.Handle(Result{})
    } else {
      e = action.Handle(msg.Result)
    }
    if e != nil {
      break
    }
  }
}
