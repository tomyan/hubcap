package main

import (
	"flag"
	"fmt"
	"sort"
)

// CommandInfo describes a CLI command.
type CommandInfo struct {
	Name     string
	Desc     string
	Category string
	Run      func(cfg *Config, args []string) int
}

// commands is the registry of all available commands.
//
// Commands fall into two shapes:
//   - "simple" commands have no flags of their own; they go through `simple()`
//     so unknown --flags are rejected uniformly.
//   - commands with their own flags handle parsing internally (e.g. cmdGoto).
//
// All commands automatically support `<cmd> help`, `<cmd> --help`, and
// `<cmd> -h` via dispatchCommand — those short-circuit before Run is called.
var commands = map[string]CommandInfo{
	// Navigation
	"goto":    {Name: "goto", Desc: "Navigate to a URL", Category: "Navigate & manage tabs", Run: func(cfg *Config, args []string) int { return cmdGoto(cfg, args) }},
	"back":    {Name: "back", Desc: "Go back in history", Category: "Navigate & manage tabs", Run: simple("back", "usage: hubcap back", 0, func(cfg *Config, _ []string) int { return cmdBack(cfg) })},
	"forward": {Name: "forward", Desc: "Go forward in history", Category: "Navigate & manage tabs", Run: simple("forward", "usage: hubcap forward", 0, func(cfg *Config, _ []string) int { return cmdForward(cfg) })},
	"reload":  {Name: "reload", Desc: "Reload the page", Category: "Navigate & manage tabs", Run: func(cfg *Config, args []string) int { return cmdReload(cfg, args) }},
	"new":     {Name: "new", Desc: "Open a new tab", Category: "Navigate & manage tabs", Run: func(cfg *Config, args []string) int { return cmdNew(cfg, args) }},
	"close":   {Name: "close", Desc: "Close the current tab", Category: "Navigate & manage tabs", Run: simple("close", "usage: hubcap close", 0, func(cfg *Config, _ []string) int { return cmdClose(cfg) })},
	"tabs":    {Name: "tabs", Desc: "List open tabs", Category: "Navigate & manage tabs", Run: simple("tabs", "usage: hubcap tabs", 0, func(cfg *Config, _ []string) int { return cmdTabs(cfg) })},
	"version": {Name: "version", Desc: "Show browser version", Category: "Navigate & manage tabs", Run: simple("version", "usage: hubcap version", 0, func(cfg *Config, _ []string) int { return cmdVersion(cfg) })},

	// Page info
	"title":   {Name: "title", Desc: "Get page title", Category: "Read page info", Run: simple("title", "usage: hubcap title", 0, func(cfg *Config, _ []string) int { return cmdTitle(cfg) })},
	"url":     {Name: "url", Desc: "Get page URL", Category: "Read page info", Run: simple("url", "usage: hubcap url", 0, func(cfg *Config, _ []string) int { return cmdURL(cfg) })},
	"info":    {Name: "info", Desc: "Get page info", Category: "Read page info", Run: simple("info", "usage: hubcap info", 0, func(cfg *Config, _ []string) int { return cmdInfo(cfg) })},
	"source":  {Name: "source", Desc: "Get page source HTML", Category: "Read page info", Run: simple("source", "usage: hubcap source", 0, func(cfg *Config, _ []string) int { return cmdSource(cfg) })},
	"meta":    {Name: "meta", Desc: "Get meta tags", Category: "Read page info", Run: simple("meta", "usage: hubcap meta", 0, func(cfg *Config, _ []string) int { return cmdMeta(cfg) })},
	"links":   {Name: "links", Desc: "Get all links", Category: "Read page info", Run: simple("links", "usage: hubcap links", 0, func(cfg *Config, _ []string) int { return cmdLinks(cfg) })},
	"scripts": {Name: "scripts", Desc: "Get script elements", Category: "Read page info", Run: simple("scripts", "usage: hubcap scripts", 0, func(cfg *Config, _ []string) int { return cmdScripts(cfg) })},
	"images":  {Name: "images", Desc: "Get image elements", Category: "Read page info", Run: simple("images", "usage: hubcap images", 0, func(cfg *Config, _ []string) int { return cmdImages(cfg) })},
	"tables":  {Name: "tables", Desc: "Get table data", Category: "Read page info", Run: simple("tables", "usage: hubcap tables", 0, func(cfg *Config, _ []string) int { return cmdTables(cfg) })},
	"forms":   {Name: "forms", Desc: "Get form elements", Category: "Read page info", Run: simple("forms", "usage: hubcap forms", 0, func(cfg *Config, _ []string) int { return cmdForms(cfg) })},
	"frames":  {Name: "frames", Desc: "Get page frames", Category: "Read page info", Run: simple("frames", "usage: hubcap frames", 0, func(cfg *Config, _ []string) int { return cmdFrames(cfg) })},

	// DOM
	"query":     {Name: "query", Desc: "Query a DOM element", Category: "Query DOM", Run: simple("query", "usage: hubcap query <selector>", 1, func(cfg *Config, pos []string) int { return cmdQuery(cfg, pos[0]) })},
	"html":      {Name: "html", Desc: "Get element outer HTML", Category: "Query DOM", Run: simple("html", "usage: hubcap html <selector>", 1, func(cfg *Config, pos []string) int { return cmdHTML(cfg, pos[0]) })},
	"text":      {Name: "text", Desc: "Get element text content", Category: "Query DOM", Run: simple("text", "usage: hubcap text <selector>", 1, func(cfg *Config, pos []string) int { return cmdText(cfg, pos[0]) })},
	"attr":      {Name: "attr", Desc: "Get element attribute", Category: "Query DOM", Run: simple("attr", "usage: hubcap attr <selector> <attribute>", 2, func(cfg *Config, pos []string) int { return cmdAttr(cfg, pos[0], pos[1]) })},
	"value":     {Name: "value", Desc: "Get input value", Category: "Query DOM", Run: simple("value", "usage: hubcap value <selector>", 1, func(cfg *Config, pos []string) int { return cmdValue(cfg, pos[0]) })},
	"count":     {Name: "count", Desc: "Count matching elements", Category: "Query DOM", Run: simple("count", "usage: hubcap count <selector>", 1, func(cfg *Config, pos []string) int { return cmdCount(cfg, pos[0]) })},
	"visible":   {Name: "visible", Desc: "Check if element is visible", Category: "Query DOM", Run: simple("visible", "usage: hubcap visible <selector>", 1, func(cfg *Config, pos []string) int { return cmdVisible(cfg, pos[0]) })},
	"exists":    {Name: "exists", Desc: "Check if element exists", Category: "Query DOM", Run: simple("exists", "usage: hubcap exists <selector>", 1, func(cfg *Config, pos []string) int { return cmdExists(cfg, pos[0]) })},
	"bounds":    {Name: "bounds", Desc: "Get element bounding box", Category: "Query DOM", Run: simple("bounds", "usage: hubcap bounds <selector>", 1, func(cfg *Config, pos []string) int { return cmdBounds(cfg, pos[0]) })},
	"styles":    {Name: "styles", Desc: "Get computed styles", Category: "Query DOM", Run: simple("styles", "usage: hubcap styles <selector>", 1, func(cfg *Config, pos []string) int { return cmdStyles(cfg, pos[0]) })},
	"computed":  {Name: "computed", Desc: "Get a computed style property", Category: "Query DOM", Run: simple("computed", "usage: hubcap computed <selector> <property>", 2, func(cfg *Config, pos []string) int { return cmdComputed(cfg, pos[0], pos[1]) })},
	"layout":    {Name: "layout", Desc: "Get element layout info", Category: "Query DOM", Run: func(cfg *Config, args []string) int { return cmdLayout(cfg, args) }},
	"shadow":    {Name: "shadow", Desc: "Query shadow DOM", Category: "Query DOM", Run: simple("shadow", "usage: hubcap shadow <host-selector> <inner-selector>", 2, func(cfg *Config, pos []string) int { return cmdShadow(cfg, pos[0], pos[1]) })},
	"find":      {Name: "find", Desc: "Find text in page", Category: "Query DOM", Run: simple("find", "usage: hubcap find <text>", 1, func(cfg *Config, pos []string) int { return cmdFind(cfg, pos[0]) })},
	"selection": {Name: "selection", Desc: "Get text selection", Category: "Query DOM", Run: simple("selection", "usage: hubcap selection", 0, func(cfg *Config, _ []string) int { return cmdSelection(cfg) })},
	"caret":     {Name: "caret", Desc: "Get caret position", Category: "Query DOM", Run: simple("caret", "usage: hubcap caret <selector>", 1, func(cfg *Config, pos []string) int { return cmdCaret(cfg, pos[0]) })},

	// Input
	"click":       {Name: "click", Desc: "Click an element", Category: "Click & interact", Run: simple("click", "usage: hubcap click <selector>", 1, func(cfg *Config, pos []string) int { return cmdClick(cfg, pos[0]) })},
	"dblclick":    {Name: "dblclick", Desc: "Double-click an element", Category: "Click & interact", Run: simple("dblclick", "usage: hubcap dblclick <selector>", 1, func(cfg *Config, pos []string) int { return cmdDblClick(cfg, pos[0]) })},
	"rightclick":  {Name: "rightclick", Desc: "Right-click an element", Category: "Click & interact", Run: simple("rightclick", "usage: hubcap rightclick <selector>", 1, func(cfg *Config, pos []string) int { return cmdRightClick(cfg, pos[0]) })},
	"tripleclick": {Name: "tripleclick", Desc: "Triple-click an element", Category: "Click & interact", Run: simple("tripleclick", "usage: hubcap tripleclick <selector>", 1, func(cfg *Config, pos []string) int { return cmdTripleClick(cfg, pos[0]) })},
	"clickat":     {Name: "clickat", Desc: "Click at coordinates", Category: "Click & interact", Run: simple("clickat", "usage: hubcap clickat <x> <y>", 2, func(cfg *Config, pos []string) int { return cmdClickAt(cfg, pos[0], pos[1]) })},
	"hover":       {Name: "hover", Desc: "Hover over an element", Category: "Click & interact", Run: simple("hover", "usage: hubcap hover <selector>", 1, func(cfg *Config, pos []string) int { return cmdHover(cfg, pos[0]) })},
	"tap":         {Name: "tap", Desc: "Tap an element (touch)", Category: "Click & interact", Run: simple("tap", "usage: hubcap tap <selector>", 1, func(cfg *Config, pos []string) int { return cmdTap(cfg, pos[0]) })},
	"focus":       {Name: "focus", Desc: "Focus an element", Category: "Click & interact", Run: simple("focus", "usage: hubcap focus <selector>", 1, func(cfg *Config, pos []string) int { return cmdFocus(cfg, pos[0]) })},
	"fill":        {Name: "fill", Desc: "Fill an input field", Category: "Click & interact", Run: simple("fill", "usage: hubcap fill <selector> <text>", 2, func(cfg *Config, pos []string) int { return cmdFill(cfg, pos[0], pos[1]) })},
	"clear":       {Name: "clear", Desc: "Clear an input field", Category: "Click & interact", Run: simple("clear", "usage: hubcap clear <selector>", 1, func(cfg *Config, pos []string) int { return cmdClear(cfg, pos[0]) })},
	"type":        {Name: "type", Desc: "Type text (keystrokes)", Category: "Click & interact", Run: simple("type", "usage: hubcap type <text>", 1, func(cfg *Config, pos []string) int { return cmdType(cfg, pos[0]) })},
	"press":       {Name: "press", Desc: "Press a key", Category: "Click & interact", Run: simple("press", "usage: hubcap press <key>", 1, func(cfg *Config, pos []string) int { return cmdPress(cfg, pos[0]) })},
	"select":      {Name: "select", Desc: "Select a dropdown option", Category: "Click & interact", Run: simple("select", "usage: hubcap select <selector> <value>", 2, func(cfg *Config, pos []string) int { return cmdSelect(cfg, pos[0], pos[1]) })},
	"check":       {Name: "check", Desc: "Check a checkbox", Category: "Click & interact", Run: simple("check", "usage: hubcap check <selector>", 1, func(cfg *Config, pos []string) int { return cmdCheck(cfg, pos[0]) })},
	"uncheck":     {Name: "uncheck", Desc: "Uncheck a checkbox", Category: "Click & interact", Run: simple("uncheck", "usage: hubcap uncheck <selector>", 1, func(cfg *Config, pos []string) int { return cmdUncheck(cfg, pos[0]) })},
	"setvalue":    {Name: "setvalue", Desc: "Set element value property", Category: "Click & interact", Run: simple("setvalue", "usage: hubcap setvalue <selector> <value>", 2, func(cfg *Config, pos []string) int { return cmdSetValue(cfg, pos[0], pos[1]) })},
	"upload":      {Name: "upload", Desc: "Upload files to input", Category: "Click & interact", Run: simple("upload", "usage: hubcap upload <selector> <file>...", 2, func(cfg *Config, pos []string) int { return cmdUpload(cfg, pos[0], pos[1:]) })},
	"dispatch":    {Name: "dispatch", Desc: "Dispatch a DOM event", Category: "Click & interact", Run: simple("dispatch", "usage: hubcap dispatch <selector> <eventType>", 2, func(cfg *Config, pos []string) int { return cmdDispatch(cfg, pos[0], pos[1]) })},
	"drag":        {Name: "drag", Desc: "Drag from one element to another", Category: "Click & interact", Run: simple("drag", "usage: hubcap drag <source-selector> <dest-selector>", 2, func(cfg *Config, pos []string) int { return cmdDrag(cfg, pos[0], pos[1]) })},
	"mouse":       {Name: "mouse", Desc: "Move mouse to coordinates", Category: "Click & interact", Run: simple("mouse", "usage: hubcap mouse <x> <y>", 2, func(cfg *Config, pos []string) int { return cmdMouse(cfg, pos[0], pos[1]) })},
	"swipe":       {Name: "swipe", Desc: "Swipe gesture on element", Category: "Click & interact", Run: simple("swipe", "usage: hubcap swipe <selector> <left|right|up|down>", 2, func(cfg *Config, pos []string) int { return cmdSwipe(cfg, pos[0], pos[1]) })},
	"pinch":       {Name: "pinch", Desc: "Pinch gesture on element", Category: "Click & interact", Run: simple("pinch", "usage: hubcap pinch <selector> <in|out>", 2, func(cfg *Config, pos []string) int { return cmdPinch(cfg, pos[0], pos[1]) })},

	// Scroll
	"scroll":       {Name: "scroll", Desc: "Scroll by offset", Category: "Scroll", Run: simple("scroll", "usage: hubcap scroll <x> <y>", 2, func(cfg *Config, pos []string) int { return cmdScroll(cfg, pos[0], pos[1]) })},
	"scrollto":     {Name: "scrollto", Desc: "Scroll element into view", Category: "Scroll", Run: simple("scrollto", "usage: hubcap scrollto <selector>", 1, func(cfg *Config, pos []string) int { return cmdScrollTo(cfg, pos[0]) })},
	"scrolltop":    {Name: "scrolltop", Desc: "Scroll to top of page", Category: "Scroll", Run: simple("scrolltop", "usage: hubcap scrolltop", 0, func(cfg *Config, _ []string) int { return cmdScrollTop(cfg) })},
	"scrollbottom": {Name: "scrollbottom", Desc: "Scroll to bottom of page", Category: "Scroll", Run: simple("scrollbottom", "usage: hubcap scrollbottom", 0, func(cfg *Config, _ []string) int { return cmdScrollBottom(cfg) })},

	// Wait
	"wait":         {Name: "wait", Desc: "Wait for element to appear", Category: "Wait", Run: func(cfg *Config, args []string) int { return cmdWait(cfg, args) }},
	"waittext":     {Name: "waittext", Desc: "Wait for text to appear", Category: "Wait", Run: func(cfg *Config, args []string) int { return cmdWaitText(cfg, args) }},
	"waitgone":     {Name: "waitgone", Desc: "Wait for element to disappear", Category: "Wait", Run: func(cfg *Config, args []string) int { return cmdWaitGone(cfg, args) }},
	"waitfn":       {Name: "waitfn", Desc: "Wait for JS expression to be truthy", Category: "Wait", Run: func(cfg *Config, args []string) int { return cmdWaitFn(cfg, args) }},
	"waitidle":     {Name: "waitidle", Desc: "Wait for network idle", Category: "Wait", Run: func(cfg *Config, args []string) int { return cmdWaitIdle(cfg, args) }},
	"waitnav":      {Name: "waitnav", Desc: "Wait for navigation", Category: "Wait", Run: func(cfg *Config, args []string) int { return cmdWaitNav(cfg, args) }},
	"waitload":     {Name: "waitload", Desc: "Wait for page load", Category: "Wait", Run: func(cfg *Config, args []string) int { return cmdWaitLoad(cfg, args) }},
	"waiturl":      {Name: "waiturl", Desc: "Wait for URL to match pattern", Category: "Wait", Run: func(cfg *Config, args []string) int { return cmdWaitURL(cfg, args) }},
	"waitrequest":  {Name: "waitrequest", Desc: "Wait for a network request", Category: "Wait", Run: func(cfg *Config, args []string) int { return cmdWaitRequest(cfg, args) }},
	"waitresponse": {Name: "waitresponse", Desc: "Wait for a network response", Category: "Wait", Run: func(cfg *Config, args []string) int { return cmdWaitResponse(cfg, args) }},

	// Capture
	"screenshot": {Name: "screenshot", Desc: "Take a screenshot", Category: "Capture", Run: func(cfg *Config, args []string) int { return cmdScreenshot(cfg, args) }},
	"pdf":        {Name: "pdf", Desc: "Print page to PDF", Category: "Capture", Run: func(cfg *Config, args []string) int { return cmdPDF(cfg, args) }},

	// Network & monitoring
	"network":      {Name: "network", Desc: "Capture network events", Category: "Network & monitor", Run: func(cfg *Config, args []string) int { return cmdNetwork(cfg, args) }},
	"har":          {Name: "har", Desc: "Capture HAR log", Category: "Network & monitor", Run: func(cfg *Config, args []string) int { return cmdHar(cfg, args) }},
	"intercept":    {Name: "intercept", Desc: "Intercept requests/responses", Category: "Network & monitor", Run: func(cfg *Config, args []string) int { return cmdIntercept(cfg, args) }},
	"block":        {Name: "block", Desc: "Block URL patterns", Category: "Network & monitor", Run: func(cfg *Config, args []string) int { return cmdBlock(cfg, args) }},
	"throttle":     {Name: "throttle", Desc: "Throttle network speed", Category: "Network & monitor", Run: func(cfg *Config, args []string) int { return cmdThrottle(cfg, args) }},
	"responsebody": {Name: "responsebody", Desc: "Get response body", Category: "Network & monitor", Run: simple("responsebody", "usage: hubcap responsebody <requestId>", 1, func(cfg *Config, pos []string) int { return cmdResponseBody(cfg, pos[0]) })},
	"console":      {Name: "console", Desc: "Capture console messages", Category: "Network & monitor", Run: func(cfg *Config, args []string) int { return cmdConsole(cfg, args) }},
	"errors":       {Name: "errors", Desc: "Capture JavaScript errors", Category: "Network & monitor", Run: func(cfg *Config, args []string) int { return cmdErrors(cfg, args) }},

	// Emulation
	"emulate":     {Name: "emulate", Desc: "Emulate a device", Category: "Emulate", Run: simple("emulate", "usage: hubcap emulate <device>", 1, func(cfg *Config, pos []string) int { return cmdEmulate(cfg, pos[0]) })},
	"useragent":   {Name: "useragent", Desc: "Set user agent", Category: "Emulate", Run: simple("useragent", "usage: hubcap useragent <string>", 1, func(cfg *Config, pos []string) int { return cmdUserAgent(cfg, pos[0]) })},
	"geolocation": {Name: "geolocation", Desc: "Set geolocation", Category: "Emulate", Run: simple("geolocation", "usage: hubcap geolocation <latitude> <longitude>", 2, func(cfg *Config, pos []string) int { return cmdGeolocation(cfg, pos[0], pos[1]) })},
	"offline":     {Name: "offline", Desc: "Toggle offline mode", Category: "Emulate", Run: simple("offline", "usage: hubcap offline <true|false>", 1, func(cfg *Config, pos []string) int { return cmdOffline(cfg, pos[0]) })},
	"media":       {Name: "media", Desc: "Set emulated media features", Category: "Emulate", Run: func(cfg *Config, args []string) int { return cmdMedia(cfg, args) }},
	"viewport":    {Name: "viewport", Desc: "Set viewport size", Category: "Emulate", Run: simple("viewport", "usage: hubcap viewport <width> <height>", 2, func(cfg *Config, pos []string) int { return cmdViewport(cfg, pos[0], pos[1]) })},
	"permission":  {Name: "permission", Desc: "Set browser permission", Category: "Emulate", Run: simple("permission", "usage: hubcap permission <name> <granted|denied|prompt>", 2, func(cfg *Config, pos []string) int { return cmdPermission(cfg, pos[0], pos[1]) })},

	// Storage
	"cookies":   {Name: "cookies", Desc: "Manage cookies", Category: "Storage", Run: func(cfg *Config, args []string) int { return cmdCookies(cfg, args) }},
	"storage":   {Name: "storage", Desc: "Manage localStorage", Category: "Storage", Run: func(cfg *Config, args []string) int { return cmdStorage(cfg, args) }},
	"session":   {Name: "session", Desc: "Manage sessionStorage", Category: "Storage", Run: func(cfg *Config, args []string) int { return cmdSession(cfg, args) }},
	"clipboard": {Name: "clipboard", Desc: "Read/write clipboard", Category: "Storage", Run: func(cfg *Config, args []string) int { return cmdClipboard(cfg, args) }},

	// Analysis
	"metrics":     {Name: "metrics", Desc: "Get performance metrics", Category: "Analyze", Run: simple("metrics", "usage: hubcap metrics", 0, func(cfg *Config, _ []string) int { return cmdMetrics(cfg) })},
	"a11y":        {Name: "a11y", Desc: "Get accessibility tree", Category: "Analyze", Run: simple("a11y", "usage: hubcap a11y", 0, func(cfg *Config, _ []string) int { return cmdA11y(cfg) })},
	"coverage":    {Name: "coverage", Desc: "Get JavaScript coverage", Category: "Analyze", Run: simple("coverage", "usage: hubcap coverage", 0, func(cfg *Config, _ []string) int { return cmdCoverage(cfg) })},
	"csscoverage": {Name: "csscoverage", Desc: "Get CSS coverage", Category: "Analyze", Run: simple("csscoverage", "usage: hubcap csscoverage", 0, func(cfg *Config, _ []string) int { return cmdCSSCoverage(cfg) })},
	"stylesheets": {Name: "stylesheets", Desc: "Get stylesheets", Category: "Analyze", Run: simple("stylesheets", "usage: hubcap stylesheets", 0, func(cfg *Config, _ []string) int { return cmdStylesheets(cfg) })},
	"listeners":   {Name: "listeners", Desc: "Get event listeners", Category: "Analyze", Run: simple("listeners", "usage: hubcap listeners <selector>", 1, func(cfg *Config, pos []string) int { return cmdListeners(cfg, pos[0]) })},
	"domsnapshot": {Name: "domsnapshot", Desc: "Get DOM snapshot", Category: "Analyze", Run: simple("domsnapshot", "usage: hubcap domsnapshot", 0, func(cfg *Config, _ []string) int { return cmdDOMSnapshot(cfg) })},

	// Profiling
	"heapsnapshot": {Name: "heapsnapshot", Desc: "Take heap snapshot", Category: "Profile", Run: func(cfg *Config, args []string) int { return cmdHeapSnapshot(cfg, args) }},
	"trace":        {Name: "trace", Desc: "Capture trace", Category: "Profile", Run: func(cfg *Config, args []string) int { return cmdTrace(cfg, args) }},

	// Advanced
	"eval":      {Name: "eval", Desc: "Evaluate JavaScript", Category: "Advanced", Run: simple("eval", "usage: hubcap eval <expression>", 1, func(cfg *Config, pos []string) int { return cmdEval(cfg, pos[0]) })},
	"evalframe": {Name: "evalframe", Desc: "Evaluate JS in a frame", Category: "Advanced", Run: simple("evalframe", "usage: hubcap evalframe <frame-id> <expression>", 2, func(cfg *Config, pos []string) int { return cmdEvalFrame(cfg, pos[0], pos[1]) })},
	"run":       {Name: "run", Desc: "Run a JavaScript file", Category: "Advanced", Run: simple("run", "usage: hubcap run <file.js>", 1, func(cfg *Config, pos []string) int { return cmdRun(cfg, pos[0]) })},
	"raw":       {Name: "raw", Desc: "Send raw CDP command", Category: "Advanced", Run: func(cfg *Config, args []string) int { return cmdRaw(cfg, args) }},
	"dialog":    {Name: "dialog", Desc: "Handle JavaScript dialog", Category: "Advanced", Run: func(cfg *Config, args []string) int { return cmdDialog(cfg, args) }},
	"highlight": {Name: "highlight", Desc: "Highlight an element", Category: "Advanced", Run: func(cfg *Config, args []string) int { return cmdHighlight(cfg, args) }},

	// Assert
	"assert": {Name: "assert", Desc: "Assert page state", Category: "Assert", Run: func(cfg *Config, args []string) int { return cmdAssert(cfg, args) }},

	// Record
	"record": {Name: "record", Desc: "Record browser interactions", Category: "Utility", Run: func(cfg *Config, args []string) int { return cmdRecord(cfg, args) }},

	// Bridge
	"bridge": {Name: "bridge", Desc: "Bidirectional JS message channel", Category: "Advanced", Run: func(cfg *Config, args []string) int { return cmdBridge(cfg, args) }},
}

func init() {
	commands["help"] = CommandInfo{Name: "help", Desc: "Show help for a command", Category: "Advanced", Run: func(cfg *Config, args []string) int { return cmdHelp(cfg, args) }}
	commands["retry"] = CommandInfo{Name: "retry", Desc: "Retry a command on failure", Category: "Utility", Run: func(cfg *Config, args []string) int { return cmdRetry(cfg, args) }}
	commands["pipe"] = CommandInfo{Name: "pipe", Desc: "Read commands from stdin", Category: "Utility", Run: func(cfg *Config, args []string) int { return cmdPipe(cfg, args) }}
	commands["shell"] = CommandInfo{Name: "shell", Desc: "Interactive REPL", Category: "Utility", Run: func(cfg *Config, args []string) int { return cmdShell(cfg, args) }}
	commands["inspect"] = CommandInfo{Name: "inspect", Desc: "Terminal web inspector", Category: "Utility", Run: func(cfg *Config, args []string) int { return cmdInspect(cfg, args) }}
}

// cmdMissingArg prints a usage message and returns ExitError.
func cmdMissingArg(cfg *Config, usage string) int {
	fmt.Fprintln(cfg.Stderr, usage)
	return ExitError
}

// categoryOrder defines the display order for command categories.
var categoryOrder = []string{
	"Navigate & manage tabs",
	"Read page info",
	"Query DOM",
	"Click & interact",
	"Scroll",
	"Wait",
	"Capture",
	"Network & monitor",
	"Emulate",
	"Storage",
	"Analyze",
	"Profile",
	"Advanced",
	"Assert",
	"Utility",
}

// commandsByCategory returns commands grouped by category, with sorted names within each category.
func commandsByCategory() []struct {
	Category string
	Commands []CommandInfo
} {
	grouped := make(map[string][]CommandInfo)
	for _, cmd := range commands {
		grouped[cmd.Category] = append(grouped[cmd.Category], cmd)
	}

	var result []struct {
		Category string
		Commands []CommandInfo
	}

	for _, cat := range categoryOrder {
		cmds := grouped[cat]
		if len(cmds) == 0 {
			continue
		}
		sort.Slice(cmds, func(i, j int) bool { return cmds[i].Name < cmds[j].Name })
		result = append(result, struct {
			Category string
			Commands []CommandInfo
		}{Category: cat, Commands: cmds})
	}

	return result
}

// sortedCommandNames returns all command names sorted alphabetically.
func sortedCommandNames() []string {
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// printBriefUsage prints a short usage message with flags and hints.
func printBriefUsage(cfg *Config, fs *flag.FlagSet) {
	fmt.Fprintln(cfg.Stderr, "usage: hubcap [flags] <command>")
	fmt.Fprintln(cfg.Stderr)
	fmt.Fprintln(cfg.Stderr, "Quick start:")
	fmt.Fprintln(cfg.Stderr, "  hubcap setup launch          Start Chrome")
	fmt.Fprintln(cfg.Stderr, "  hubcap tabs                  List open tabs")
	fmt.Fprintln(cfg.Stderr, "  hubcap goto https://example   Navigate to a URL")
	fmt.Fprintln(cfg.Stderr, "  hubcap title                 Get page title")
	fmt.Fprintln(cfg.Stderr, "  hubcap click 'button#submit' Click an element")
	fmt.Fprintln(cfg.Stderr)
	fmt.Fprintln(cfg.Stderr, "flags:")
	fs.PrintDefaults()
	fmt.Fprintln(cfg.Stderr)
	fmt.Fprintln(cfg.Stderr, "Run 'hubcap --help-commands' to list all commands.")
	fmt.Fprintln(cfg.Stderr, "Run 'hubcap help <command>' for detailed help on a command.")
}

// printFullCommandList prints all commands grouped by category, one per line with descriptions.
func printFullCommandList(cfg *Config) {
	fmt.Fprintln(cfg.Stderr, "usage: hubcap [flags] <command>")
	fmt.Fprintln(cfg.Stderr)
	for _, group := range commandsByCategory() {
		fmt.Fprintf(cfg.Stderr, "  %s:\n", group.Category)
		for _, cmd := range group.Commands {
			fmt.Fprintf(cfg.Stderr, "    %-14s %s\n", cmd.Name, cmd.Desc)
		}
		fmt.Fprintln(cfg.Stderr)
	}
	fmt.Fprintln(cfg.Stderr, "Run 'hubcap help <command>' for detailed help on a command.")
}
