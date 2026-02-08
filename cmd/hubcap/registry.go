package main

import (
	"flag"
	"fmt"
	"sort"
	"strings"
)

// CommandInfo describes a CLI command.
type CommandInfo struct {
	Name     string
	Desc     string
	Category string
	Run      func(cfg *Config, args []string) int
}

// commands is the registry of all available commands.
var commands = map[string]CommandInfo{
	// Navigation
	"goto":    {Name: "goto", Desc: "Navigate to a URL", Category: "Navigate & manage tabs", Run: func(cfg *Config, args []string) int { return cmdGoto(cfg, args) }},
	"back":    {Name: "back", Desc: "Go back in history", Category: "Navigate & manage tabs", Run: func(cfg *Config, args []string) int { return cmdBack(cfg) }},
	"forward": {Name: "forward", Desc: "Go forward in history", Category: "Navigate & manage tabs", Run: func(cfg *Config, args []string) int { return cmdForward(cfg) }},
	"reload":  {Name: "reload", Desc: "Reload the page", Category: "Navigate & manage tabs", Run: func(cfg *Config, args []string) int { return cmdReload(cfg, args) }},
	"new":     {Name: "new", Desc: "Open a new tab", Category: "Navigate & manage tabs", Run: func(cfg *Config, args []string) int {
		url := ""
		if len(args) > 0 {
			url = args[0]
		}
		return cmdNew(cfg, url)
	}},
	"close":   {Name: "close", Desc: "Close the current tab", Category: "Navigate & manage tabs", Run: func(cfg *Config, args []string) int { return cmdClose(cfg) }},
	"tabs":    {Name: "tabs", Desc: "List open tabs", Category: "Navigate & manage tabs", Run: func(cfg *Config, args []string) int { return cmdTabs(cfg) }},
	"version": {Name: "version", Desc: "Show browser version", Category: "Navigate & manage tabs", Run: func(cfg *Config, args []string) int { return cmdVersion(cfg) }},

	// Page info
	"title":  {Name: "title", Desc: "Get page title", Category: "Read page info", Run: func(cfg *Config, args []string) int { return cmdTitle(cfg) }},
	"url":    {Name: "url", Desc: "Get page URL", Category: "Read page info", Run: func(cfg *Config, args []string) int { return cmdURL(cfg) }},
	"info":   {Name: "info", Desc: "Get page info", Category: "Read page info", Run: func(cfg *Config, args []string) int { return cmdInfo(cfg) }},
	"source": {Name: "source", Desc: "Get page source HTML", Category: "Read page info", Run: func(cfg *Config, args []string) int { return cmdSource(cfg) }},
	"meta":   {Name: "meta", Desc: "Get meta tags", Category: "Read page info", Run: func(cfg *Config, args []string) int { return cmdMeta(cfg) }},
	"links":  {Name: "links", Desc: "Get all links", Category: "Read page info", Run: func(cfg *Config, args []string) int { return cmdLinks(cfg) }},
	"scripts": {Name: "scripts", Desc: "Get script elements", Category: "Read page info", Run: func(cfg *Config, args []string) int { return cmdScripts(cfg) }},
	"images": {Name: "images", Desc: "Get image elements", Category: "Read page info", Run: func(cfg *Config, args []string) int { return cmdImages(cfg) }},
	"tables": {Name: "tables", Desc: "Get table data", Category: "Read page info", Run: func(cfg *Config, args []string) int { return cmdTables(cfg) }},
	"forms":  {Name: "forms", Desc: "Get form elements", Category: "Read page info", Run: func(cfg *Config, args []string) int { return cmdForms(cfg) }},
	"frames": {Name: "frames", Desc: "Get page frames", Category: "Read page info", Run: func(cfg *Config, args []string) int { return cmdFrames(cfg) }},

	// DOM
	"query": {Name: "query", Desc: "Query a DOM element", Category: "Query DOM", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap query <selector>")
		}
		return cmdQuery(cfg, args[0])
	}},
	"html": {Name: "html", Desc: "Get element outer HTML", Category: "Query DOM", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap html <selector>")
		}
		return cmdHTML(cfg, args[0])
	}},
	"text": {Name: "text", Desc: "Get element text content", Category: "Query DOM", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap text <selector>")
		}
		return cmdText(cfg, args[0])
	}},
	"attr": {Name: "attr", Desc: "Get element attribute", Category: "Query DOM", Run: func(cfg *Config, args []string) int {
		if len(args) < 2 {
			return cmdMissingArg(cfg, "usage: hubcap attr <selector> <attribute>")
		}
		return cmdAttr(cfg, args[0], args[1])
	}},
	"value": {Name: "value", Desc: "Get input value", Category: "Query DOM", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap value <selector>")
		}
		return cmdValue(cfg, args[0])
	}},
	"count": {Name: "count", Desc: "Count matching elements", Category: "Query DOM", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap count <selector>")
		}
		return cmdCount(cfg, args[0])
	}},
	"visible": {Name: "visible", Desc: "Check if element is visible", Category: "Query DOM", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap visible <selector>")
		}
		return cmdVisible(cfg, args[0])
	}},
	"exists": {Name: "exists", Desc: "Check if element exists", Category: "Query DOM", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap exists <selector>")
		}
		return cmdExists(cfg, args[0])
	}},
	"bounds": {Name: "bounds", Desc: "Get element bounding box", Category: "Query DOM", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap bounds <selector>")
		}
		return cmdBounds(cfg, args[0])
	}},
	"styles": {Name: "styles", Desc: "Get computed styles", Category: "Query DOM", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap styles <selector>")
		}
		return cmdStyles(cfg, args[0])
	}},
	"computed": {Name: "computed", Desc: "Get a computed style property", Category: "Query DOM", Run: func(cfg *Config, args []string) int {
		if len(args) < 2 {
			return cmdMissingArg(cfg, "usage: hubcap computed <selector> <property>")
		}
		return cmdComputed(cfg, args[0], args[1])
	}},
	"layout": {Name: "layout", Desc: "Get element layout info", Category: "Query DOM", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap layout <selector> [--depth <n>]")
		}
		return cmdLayout(cfg, args)
	}},
	"shadow": {Name: "shadow", Desc: "Query shadow DOM", Category: "Query DOM", Run: func(cfg *Config, args []string) int {
		if len(args) < 2 {
			return cmdMissingArg(cfg, "usage: hubcap shadow <host-selector> <inner-selector>")
		}
		return cmdShadow(cfg, args[0], args[1])
	}},
	"find": {Name: "find", Desc: "Find text in page", Category: "Query DOM", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap find <text>")
		}
		return cmdFind(cfg, args[0])
	}},
	"selection": {Name: "selection", Desc: "Get text selection", Category: "Query DOM", Run: func(cfg *Config, args []string) int { return cmdSelection(cfg) }},
	"caret": {Name: "caret", Desc: "Get caret position", Category: "Query DOM", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap caret <selector>")
		}
		return cmdCaret(cfg, args[0])
	}},

	// Input
	"click": {Name: "click", Desc: "Click an element", Category: "Click & interact", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap click <selector>")
		}
		return cmdClick(cfg, args[0])
	}},
	"dblclick": {Name: "dblclick", Desc: "Double-click an element", Category: "Click & interact", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap dblclick <selector>")
		}
		return cmdDblClick(cfg, args[0])
	}},
	"rightclick": {Name: "rightclick", Desc: "Right-click an element", Category: "Click & interact", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap rightclick <selector>")
		}
		return cmdRightClick(cfg, args[0])
	}},
	"tripleclick": {Name: "tripleclick", Desc: "Triple-click an element", Category: "Click & interact", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap tripleclick <selector>")
		}
		return cmdTripleClick(cfg, args[0])
	}},
	"clickat": {Name: "clickat", Desc: "Click at coordinates", Category: "Click & interact", Run: func(cfg *Config, args []string) int {
		if len(args) < 2 {
			return cmdMissingArg(cfg, "usage: hubcap clickat <x> <y>")
		}
		return cmdClickAt(cfg, args[0], args[1])
	}},
	"hover": {Name: "hover", Desc: "Hover over an element", Category: "Click & interact", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap hover <selector>")
		}
		return cmdHover(cfg, args[0])
	}},
	"tap": {Name: "tap", Desc: "Tap an element (touch)", Category: "Click & interact", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap tap <selector>")
		}
		return cmdTap(cfg, args[0])
	}},
	"focus": {Name: "focus", Desc: "Focus an element", Category: "Click & interact", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap focus <selector>")
		}
		return cmdFocus(cfg, args[0])
	}},
	"fill": {Name: "fill", Desc: "Fill an input field", Category: "Click & interact", Run: func(cfg *Config, args []string) int {
		if len(args) < 2 {
			return cmdMissingArg(cfg, "usage: hubcap fill <selector> <text>")
		}
		return cmdFill(cfg, args[0], args[1])
	}},
	"clear": {Name: "clear", Desc: "Clear an input field", Category: "Click & interact", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap clear <selector>")
		}
		return cmdClear(cfg, args[0])
	}},
	"type": {Name: "type", Desc: "Type text (keystrokes)", Category: "Click & interact", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap type <text>")
		}
		return cmdType(cfg, args[0])
	}},
	"press": {Name: "press", Desc: "Press a key", Category: "Click & interact", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap press <key>")
		}
		return cmdPress(cfg, args[0])
	}},
	"select": {Name: "select", Desc: "Select a dropdown option", Category: "Click & interact", Run: func(cfg *Config, args []string) int {
		if len(args) < 2 {
			return cmdMissingArg(cfg, "usage: hubcap select <selector> <value>")
		}
		return cmdSelect(cfg, args[0], args[1])
	}},
	"check": {Name: "check", Desc: "Check a checkbox", Category: "Click & interact", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap check <selector>")
		}
		return cmdCheck(cfg, args[0])
	}},
	"uncheck": {Name: "uncheck", Desc: "Uncheck a checkbox", Category: "Click & interact", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap uncheck <selector>")
		}
		return cmdUncheck(cfg, args[0])
	}},
	"setvalue": {Name: "setvalue", Desc: "Set element value property", Category: "Click & interact", Run: func(cfg *Config, args []string) int {
		if len(args) < 2 {
			return cmdMissingArg(cfg, "usage: hubcap setvalue <selector> <value>")
		}
		return cmdSetValue(cfg, args[0], args[1])
	}},
	"upload": {Name: "upload", Desc: "Upload files to input", Category: "Click & interact", Run: func(cfg *Config, args []string) int {
		if len(args) < 2 {
			return cmdMissingArg(cfg, "usage: hubcap upload <selector> <file>...")
		}
		return cmdUpload(cfg, args[0], args[1:])
	}},
	"dispatch": {Name: "dispatch", Desc: "Dispatch a DOM event", Category: "Click & interact", Run: func(cfg *Config, args []string) int {
		if len(args) < 2 {
			return cmdMissingArg(cfg, "usage: hubcap dispatch <selector> <eventType>")
		}
		return cmdDispatch(cfg, args[0], args[1])
	}},
	"drag": {Name: "drag", Desc: "Drag from one element to another", Category: "Click & interact", Run: func(cfg *Config, args []string) int {
		if len(args) < 2 {
			return cmdMissingArg(cfg, "usage: hubcap drag <source-selector> <dest-selector>")
		}
		return cmdDrag(cfg, args[0], args[1])
	}},
	"mouse": {Name: "mouse", Desc: "Move mouse to coordinates", Category: "Click & interact", Run: func(cfg *Config, args []string) int {
		if len(args) < 2 {
			return cmdMissingArg(cfg, "usage: hubcap mouse <x> <y>")
		}
		return cmdMouse(cfg, args[0], args[1])
	}},
	"swipe": {Name: "swipe", Desc: "Swipe gesture on element", Category: "Click & interact", Run: func(cfg *Config, args []string) int {
		if len(args) < 2 {
			return cmdMissingArg(cfg, "usage: hubcap swipe <selector> <left|right|up|down>")
		}
		return cmdSwipe(cfg, args[0], args[1])
	}},
	"pinch": {Name: "pinch", Desc: "Pinch gesture on element", Category: "Click & interact", Run: func(cfg *Config, args []string) int {
		if len(args) < 2 {
			return cmdMissingArg(cfg, "usage: hubcap pinch <selector> <in|out>")
		}
		return cmdPinch(cfg, args[0], args[1])
	}},

	// Scroll
	"scroll":       {Name: "scroll", Desc: "Scroll by offset", Category: "Scroll", Run: func(cfg *Config, args []string) int {
		if len(args) < 2 {
			return cmdMissingArg(cfg, "usage: hubcap scroll <x> <y>")
		}
		return cmdScroll(cfg, args[0], args[1])
	}},
	"scrollto":     {Name: "scrollto", Desc: "Scroll element into view", Category: "Scroll", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap scrollto <selector>")
		}
		return cmdScrollTo(cfg, args[0])
	}},
	"scrolltop":    {Name: "scrolltop", Desc: "Scroll to top of page", Category: "Scroll", Run: func(cfg *Config, args []string) int { return cmdScrollTop(cfg) }},
	"scrollbottom": {Name: "scrollbottom", Desc: "Scroll to bottom of page", Category: "Scroll", Run: func(cfg *Config, args []string) int { return cmdScrollBottom(cfg) }},

	// Wait
	"wait": {Name: "wait", Desc: "Wait for element to appear", Category: "Wait", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap wait <selector> [--timeout <duration>]")
		}
		return cmdWait(cfg, args)
	}},
	"waittext": {Name: "waittext", Desc: "Wait for text to appear", Category: "Wait", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap waittext <text> [--timeout <duration>]")
		}
		return cmdWaitText(cfg, args[0], args[1:])
	}},
	"waitgone": {Name: "waitgone", Desc: "Wait for element to disappear", Category: "Wait", Run: func(cfg *Config, args []string) int { return cmdWaitGone(cfg, args) }},
	"waitfn": {Name: "waitfn", Desc: "Wait for JS expression to be truthy", Category: "Wait", Run: func(cfg *Config, args []string) int { return cmdWaitFn(cfg, args) }},
	"waitidle": {Name: "waitidle", Desc: "Wait for network idle", Category: "Wait", Run: func(cfg *Config, args []string) int { return cmdWaitIdle(cfg, args) }},
	"waitnav": {Name: "waitnav", Desc: "Wait for navigation", Category: "Wait", Run: func(cfg *Config, args []string) int { return cmdWaitNav(cfg, args) }},
	"waitload": {Name: "waitload", Desc: "Wait for page load", Category: "Wait", Run: func(cfg *Config, args []string) int { return cmdWaitLoad(cfg, args) }},
	"waiturl": {Name: "waiturl", Desc: "Wait for URL to match pattern", Category: "Wait", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap waiturl <pattern> [--timeout <duration>]")
		}
		return cmdWaitURL(cfg, args[0], args[1:])
	}},
	"waitrequest": {Name: "waitrequest", Desc: "Wait for a network request", Category: "Wait", Run: func(cfg *Config, args []string) int { return cmdWaitRequest(cfg, args) }},
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
	"responsebody": {Name: "responsebody", Desc: "Get response body", Category: "Network & monitor", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap responsebody <requestId>")
		}
		return cmdResponseBody(cfg, args[0])
	}},
	"console": {Name: "console", Desc: "Capture console messages", Category: "Network & monitor", Run: func(cfg *Config, args []string) int { return cmdConsole(cfg, args) }},
	"errors":  {Name: "errors", Desc: "Capture JavaScript errors", Category: "Network & monitor", Run: func(cfg *Config, args []string) int { return cmdErrors(cfg, args) }},

	// Emulation
	"emulate":     {Name: "emulate", Desc: "Emulate a device", Category: "Emulate", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap emulate <device>")
		}
		return cmdEmulate(cfg, args[0])
	}},
	"useragent":   {Name: "useragent", Desc: "Set user agent", Category: "Emulate", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap useragent <string>")
		}
		return cmdUserAgent(cfg, args[0])
	}},
	"geolocation": {Name: "geolocation", Desc: "Set geolocation", Category: "Emulate", Run: func(cfg *Config, args []string) int {
		if len(args) < 2 {
			return cmdMissingArg(cfg, "usage: hubcap geolocation <latitude> <longitude>")
		}
		return cmdGeolocation(cfg, args[0], args[1])
	}},
	"offline":     {Name: "offline", Desc: "Toggle offline mode", Category: "Emulate", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap offline <true|false>")
		}
		return cmdOffline(cfg, args[0])
	}},
	"media":       {Name: "media", Desc: "Set emulated media features", Category: "Emulate", Run: func(cfg *Config, args []string) int { return cmdMedia(cfg, args) }},
	"viewport":    {Name: "viewport", Desc: "Set viewport size", Category: "Emulate", Run: func(cfg *Config, args []string) int {
		if len(args) < 2 {
			return cmdMissingArg(cfg, "usage: hubcap viewport <width> <height>")
		}
		return cmdViewport(cfg, args[0], args[1])
	}},
	"permission": {Name: "permission", Desc: "Set browser permission", Category: "Emulate", Run: func(cfg *Config, args []string) int {
		if len(args) < 2 {
			return cmdMissingArg(cfg, "usage: hubcap permission <name> <granted|denied|prompt>")
		}
		return cmdPermission(cfg, args[0], args[1])
	}},

	// Storage
	"cookies":   {Name: "cookies", Desc: "Manage cookies", Category: "Storage", Run: func(cfg *Config, args []string) int { return cmdCookies(cfg, args) }},
	"storage":   {Name: "storage", Desc: "Manage localStorage", Category: "Storage", Run: func(cfg *Config, args []string) int { return cmdStorage(cfg, args) }},
	"session":   {Name: "session", Desc: "Manage sessionStorage", Category: "Storage", Run: func(cfg *Config, args []string) int { return cmdSession(cfg, args) }},
	"clipboard": {Name: "clipboard", Desc: "Read/write clipboard", Category: "Storage", Run: func(cfg *Config, args []string) int { return cmdClipboard(cfg, args) }},

	// Analysis
	"metrics":      {Name: "metrics", Desc: "Get performance metrics", Category: "Analyze", Run: func(cfg *Config, args []string) int { return cmdMetrics(cfg) }},
	"a11y":         {Name: "a11y", Desc: "Get accessibility tree", Category: "Analyze", Run: func(cfg *Config, args []string) int { return cmdA11y(cfg) }},
	"coverage":     {Name: "coverage", Desc: "Get JavaScript coverage", Category: "Analyze", Run: func(cfg *Config, args []string) int { return cmdCoverage(cfg) }},
	"csscoverage":  {Name: "csscoverage", Desc: "Get CSS coverage", Category: "Analyze", Run: func(cfg *Config, args []string) int { return cmdCSSCoverage(cfg) }},
	"stylesheets":  {Name: "stylesheets", Desc: "Get stylesheets", Category: "Analyze", Run: func(cfg *Config, args []string) int { return cmdStylesheets(cfg) }},
	"listeners":    {Name: "listeners", Desc: "Get event listeners", Category: "Analyze", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap listeners <selector>")
		}
		return cmdListeners(cfg, args[0])
	}},
	"domsnapshot": {Name: "domsnapshot", Desc: "Get DOM snapshot", Category: "Analyze", Run: func(cfg *Config, args []string) int { return cmdDOMSnapshot(cfg) }},

	// Profiling
	"heapsnapshot": {Name: "heapsnapshot", Desc: "Take heap snapshot", Category: "Profile", Run: func(cfg *Config, args []string) int { return cmdHeapSnapshot(cfg, args) }},
	"trace":        {Name: "trace", Desc: "Capture trace", Category: "Profile", Run: func(cfg *Config, args []string) int { return cmdTrace(cfg, args) }},

	// Advanced
	"eval":      {Name: "eval", Desc: "Evaluate JavaScript", Category: "Advanced", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap eval <expression>")
		}
		return cmdEval(cfg, args[0])
	}},
	"evalframe": {Name: "evalframe", Desc: "Evaluate JS in a frame", Category: "Advanced", Run: func(cfg *Config, args []string) int {
		if len(args) < 2 {
			return cmdMissingArg(cfg, "usage: hubcap evalframe <frame-id> <expression>")
		}
		return cmdEvalFrame(cfg, args[0], args[1])
	}},
	"run":       {Name: "run", Desc: "Run a JavaScript file", Category: "Advanced", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap run <file.js>")
		}
		return cmdRun(cfg, args[0])
	}},
	"raw":       {Name: "raw", Desc: "Send raw CDP command", Category: "Advanced", Run: func(cfg *Config, args []string) int { return cmdRaw(cfg, args) }},
	"dialog":    {Name: "dialog", Desc: "Handle JavaScript dialog", Category: "Advanced", Run: func(cfg *Config, args []string) int { return cmdDialog(cfg, args) }},
	"highlight": {Name: "highlight", Desc: "Highlight an element", Category: "Advanced", Run: func(cfg *Config, args []string) int {
		if len(args) < 1 {
			return cmdMissingArg(cfg, "usage: hubcap highlight <selector> [--hide]")
		}
		return cmdHighlight(cfg, args)
	}},

	// Assert
	"assert": {Name: "assert", Desc: "Assert page state", Category: "Assert", Run: func(cfg *Config, args []string) int { return cmdAssert(cfg, args) }},

}

func init() {
	commands["help"] = CommandInfo{Name: "help", Desc: "Show help for a command", Category: "Advanced", Run: func(cfg *Config, args []string) int { return cmdHelp(cfg, args) }}
	commands["retry"] = CommandInfo{Name: "retry", Desc: "Retry a command on failure", Category: "Utility", Run: func(cfg *Config, args []string) int { return cmdRetry(cfg, args) }}
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

// printUsage prints the usage message with commands grouped by category.
func printUsage(cfg *Config, fs *flag.FlagSet) {
	fmt.Fprintln(cfg.Stderr, "usage: hubcap [flags] <command>")
	fmt.Fprintln(cfg.Stderr)

	for _, group := range commandsByCategory() {
		fmt.Fprintf(cfg.Stderr, "  %s:\n", group.Category)
		names := make([]string, len(group.Commands))
		for i, cmd := range group.Commands {
			names[i] = cmd.Name
		}
		fmt.Fprintf(cfg.Stderr, "    %s\n", strings.Join(names, ", "))
		fmt.Fprintln(cfg.Stderr)
	}

	fmt.Fprintln(cfg.Stderr, "flags:")
	fs.PrintDefaults()
}
