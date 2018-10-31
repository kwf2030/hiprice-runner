// Domain的方法/事件都是字符串，
// 为方便使用，这里仅集成了较为常用的，
// 其它的请参考官方文档（https://chromedevtools.github.io/devtools-protocol/tot）

package main

var Browser = struct {
  Close      string
  GetVersion string
}{
  "Browser.close",
  "Browser.getVersion",
}

var DOM = struct {
  Enable           string
  Disable          string
  DescribeNode     string
  GetDocument      string
  QuerySelector    string
  QuerySelectorAll string
  RequestNode      string
  ResolveNode      string
}{
  "DOM.enable",
  "DOM.disable",
  "DOM.describeNode",
  "DOM.getDocument",
  "DOM.querySelector",
  "DOM.querySelectorAll",
  "DOM.requestNode",
  "DOM.resolveNode",
}

var Input = struct {
  DispatchKeyEvent   string
  DispatchMouseEvent string
  DispatchTouchEvent string
}{
  "Input.dispatchKeyEvent",
  "Input.dispatchMouseEvent",
  "Input.dispatchTouchEvent",
}

var Page = struct {
  Enable            string
  Disable           string
  BringToFront      string
  CaptureScreenshot string
  Close             string
  Navigate          string
  Reload            string
  StopLoading       string

  DomContentEventFired string
  FrameAttached        string
  FrameDetached        string
  FrameNavigated       string
  LifecycleEvent       string
  LoadEventFired       string
  WindowOpen           string
}{
  "Page.enable",
  "Page.disable",
  "Page.bringToFront",
  "Page.captureScreenshot",
  "Page.close",
  "Page.navigate",
  "Page.reload",
  "Page.stopLoading",

  "Page.domContentEventFired",
  "Page.frameAttached",
  "Page.frameDetached",
  "Page.frameNavigated",
  "Page.lifecycleEvent",
  "Page.loadEventFired",
  "Page.windowOpen",
}

var Runtime = struct {
  Enable        string
  Disable       string
  CompileScript string
  Evaluate      string
  QueryObjects  string
  RunScript     string
}{
  "Runtime.enable",
  "Runtime.disable",
  "Runtime.compileScript",
  "Runtime.evaluate",
  "Runtime.queryObjects",
  "Runtime.runScript",
}

var Target = struct {
  ActivateTarget        string
  CloseTarget           string
  CreateTarget          string
  GetTargetInfo         string
  GetTargets            string
  CreateBrowserContext  string
  DisposeBrowserContext string
  GetBrowserContexts    string

  TargetCreated     string
  TargetDestroyed   string
  TargetInfoChanged string
}{
  "Runtime.activateTarget",
  "Runtime.closeTarget",
  "Runtime.createTarget",
  "Runtime.getTargetInfo",
  "Runtime.getTargets",
  "Runtime.createBrowserContext",
  "Runtime.disposeBrowserContext",
  "Runtime.getBrowserContexts",

  "Runtime.targetCreated",
  "Runtime.targetDestroyed",
  "Runtime.targetInfoChanged",
}
