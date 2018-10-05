package cdp

var Browser = struct {
  Close      string
  GetVersion string
}{
  "Browser.close",
  "Browser.getVersion",
}

var DOM = struct {
  Enable               string
  Disable              string
  DescribeNode         string
  GetAttributes        string
  GetDocument          string
  GetFlattenedDocument string
  GetNodeForLocation   string
  GetOuterHTML         string
  PerformSearch        string
  QuerySelector        string
  QuerySelectorAll     string
  RemoveAttribute      string
  RemoveNode           string
  RequestChildNodes    string
  RequestNode          string
  ResolveNode          string
  SetAttributeValue    string
  SetAttributesAsText  string
  SetFileInputFiles    string
  SetNodeName          string
  SetNodeValue         string
  SetOuterHTML         string

  DocumentUpdated string
}{
  "DOM.enable",
  "DOM.disable",
  "DOM.describeNode",
  "DOM.getAttributes",
  "DOM.getDocument",
  "DOM.getFlattenedDocument",
  "DOM.getNodeForLocation",
  "DOM.getOuterHTML",
  "DOM.performSearch",
  "DOM.querySelector",
  "DOM.querySelectorAll",
  "DOM.removeAttribute",
  "DOM.removeNode",
  "DOM.requestChildNodes",
  "DOM.requestNode",
  "DOM.resolveNode",
  "DOM.setAttributeValue",
  "DOM.setAttributesAsText",
  "DOM.setFileInputFiles",
  "DOM.setNodeName",
  "DOM.setNodeValue",
  "DOM.setOuterHTML",

  "DOM.documentUpdated",
}

var Input = struct {
  DispatchKeyEvent           string
  DispatchMouseEvent         string
  DispatchTouchEvent         string
  EmulateTouchFromMouseEvent string
  SetIgnoreInputEvents       string
  SynthesizePinchGesture     string
  SynthesizeScrollGesture    string
  SynthesizeTapGesture       string
}{
  "Input.dispatchKeyEvent",
  "Input.dispatchMouseEvent",
  "Input.dispatchTouchEvent",
  "Input.emulateTouchFromMouseEvent",
  "Input.setIgnoreInputEvents",
  "Input.synthesizePinchGesture",
  "Input.synthesizeScrollGesture",
  "Input.synthesizeTapGesture",
}

var Page = struct {
  Enable            string
  Disable           string
  BringToFront      string
  CaptureScreenshot string
  Close             string
  DeleteCookie      string
  GetCookies        string
  Navigate          string
  PrintToPDF        string
  Reload            string
  StopLoading       string

  DomContentEventFired string
  FrameAttached        string
  FrameDetached        string
  FrameNavigated       string
  FrameStartedLoading  string
  FrameStoppedLoading  string
  LifecycleEvent       string
  LoadEventFired       string
}{
  "Page.enable",
  "Page.disable",
  "Page.bringToFront",
  "Page.captureScreenshot",
  "Page.close",
  "Page.deleteCookie",
  "Page.getCookies",
  "Page.navigate",
  "Page.printToPDF",
  "Page.reload",
  "Page.stopLoading",

  "Page.domContentEventFired",
  "Page.frameAttached",
  "Page.frameDetached",
  "Page.frameNavigated",
  "Page.frameStartedLoading",
  "Page.frameStoppedLoading",
  "Page.lifecycleEvent",
  "Page.loadEventFired",
}

var Runtime = struct {
  Enable        string
  Disable       string
  CompileScript string
  Evaluate      string
  RunScript     string
}{
  "Runtime.enable",
  "Runtime.disable",
  "Runtime.compileScript",
  "Runtime.evaluate",
  "Runtime.runScript",
}

/*var Target = struct {
  ActivateTarget      string
  AttachToTarget      string
  CloseTarget         string
  CreateTarget        string
  DetachFromTarget    string
  GetTargetInfo       string
  GetTargets          string
  SendMessageToTarget string
  SetDiscoverTargets  string

  AttachedToTarget          string
  DetachedFromTarget        string
  ReceivedMessageFromTarget string
  TargetCreated             string
  TargetDestroyed           string
  TargetCrashed             string
  TargetInfoChanged         string
}{
  "Target.activateTarget",
  "Target.attachToTarget",
  "Target.closeTarget",
  "Target.createTarget",
  "Target.detachFromTarget",
  "Target.getTargetInfo",
  "Target.getTargets",
  "Target.sendMessageToTarget",
  "Target.setDiscoverTargets",

  "Target.attachedToTarget",
  "Target.detachedFromTarget",
  "Target.receivedMessageFromTarget",
  "Target.targetCreated",
  "Target.targetDestroyed",
  "Target.targetCrashed",
  "Target.targetInfoChanged",
}*/
