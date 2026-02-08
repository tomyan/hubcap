package chrome

import (
	"encoding/json"
	"errors"
	"fmt"
)

// --- Errors ---

// Errors
var (
	ErrConnectionClosed = errors.New("connection closed")
	ErrProtocolError         = errors.New("protocol error")
)

// ProtocolError represents an error returned by the Chrome DevTools Protocol.
type ProtocolError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *ProtocolError) Error() string {
	return fmt.Sprintf("protocol error %d: %s", e.Code, e.Message)
}

func (e *ProtocolError) Unwrap() error {
	return ErrProtocolError
}

// --- Browser & Page Info ---

// VersionInfo contains browser version information.
type VersionInfo struct {
	Browser         string `json:"browser"`
	ProtocolVersion string `json:"protocol"`
	UserAgent       string `json:"userAgent,omitempty"`
	V8Version       string `json:"v8,omitempty"`
}

// TargetInfo contains information about a browser target (tab/page).
type TargetInfo struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

// PageInfo represents combined information about the current page.
type PageInfo struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	ReadyState  string `json:"readyState"`
	CharacterSet string `json:"characterSet"`
	ContentType string `json:"contentType"`
}

// --- Navigation ---

// NavigateResult contains the result of a navigation.
type NavigateResult struct {
	FrameID   string `json:"frameId"`
	LoaderID  string `json:"loaderId,omitempty"`
	URL       string `json:"url"`
	ErrorText string `json:"errorText,omitempty"`
}

// --- Screenshots & PDF ---

// ScreenshotOptions configures screenshot capture.
type ScreenshotOptions struct {
	Format  string // "png", "jpeg", "webp"
	Quality int    // 0-100, only for jpeg/webp
}

// ScreenshotResult contains metadata about a captured screenshot.
type ScreenshotResult struct {
	Format string `json:"format"`
	Size   int    `json:"size"`
}

// PDFOptions configures PDF generation.
type PDFOptions struct {
	Landscape           bool    `json:"landscape,omitempty"`
	PrintBackground     bool    `json:"printBackground,omitempty"`
	Scale               float64 `json:"scale,omitempty"`
	PaperWidth          float64 `json:"paperWidth,omitempty"`  // inches
	PaperHeight         float64 `json:"paperHeight,omitempty"` // inches
	MarginTop           float64 `json:"marginTop,omitempty"`   // inches
	MarginBottom        float64 `json:"marginBottom,omitempty"`
	MarginLeft          float64 `json:"marginLeft,omitempty"`
	MarginRight         float64 `json:"marginRight,omitempty"`
	PageRanges          string  `json:"pageRanges,omitempty"` // e.g. "1-5, 8"
	PreferCSSPageSize   bool    `json:"preferCSSPageSize,omitempty"`
}

// --- JavaScript Evaluation ---

// EvalResult contains the result of evaluating a JavaScript expression.
type EvalResult struct {
	Value interface{} `json:"value"`
	Type  string      `json:"type,omitempty"`
}

// ExceptionInfo represents a JavaScript exception.
type ExceptionInfo struct {
	Text        string `json:"text"`
	LineNumber  int    `json:"lineNumber,omitempty"`
	ColumnNumber int   `json:"columnNumber,omitempty"`
	URL         string `json:"url,omitempty"`
}

// --- Console ---

// ConsoleMessage represents a console message from the browser.
type ConsoleMessage struct {
	Type string `json:"type"` // "log", "warn", "error", "info", "debug"
	Text string `json:"text"`
}

// --- Network ---

// NetworkEvent represents a network request or response event.
type NetworkEvent struct {
	Type      string `json:"type"`      // "request" or "response"
	RequestID string `json:"requestId"` // unique identifier for matching request/response
	URL       string `json:"url"`
	Method    string `json:"method,omitempty"`    // HTTP method (requests only)
	Status    int    `json:"status,omitempty"`    // HTTP status code (responses only)
	MimeType  string `json:"mimeType,omitempty"`  // MIME type (responses only)
}

// Cookie represents a browser cookie.
type Cookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain,omitempty"`
	Path     string  `json:"path,omitempty"`
	Expires  float64 `json:"expires,omitempty"`
	HTTPOnly bool    `json:"httpOnly,omitempty"`
	Secure   bool    `json:"secure,omitempty"`
	SameSite string  `json:"sameSite,omitempty"`
}

// NetworkConditions represents network throttling settings.
type NetworkConditions struct {
	Offline            bool    `json:"offline"`
	Latency            float64 `json:"latency"`             // Milliseconds
	DownloadThroughput float64 `json:"downloadThroughput"`  // Bytes per second (-1 = disabled)
	UploadThroughput   float64 `json:"uploadThroughput"`    // Bytes per second (-1 = disabled)
}

// NetworkPresets contains common network condition presets.
var NetworkPresets = map[string]NetworkConditions{
	"slow3g": {
		Offline:            false,
		Latency:            2000,
		DownloadThroughput: 50000,  // 50 KB/s
		UploadThroughput:   25000,  // 25 KB/s
	},
	"fast3g": {
		Offline:            false,
		Latency:            563,
		DownloadThroughput: 180000, // 180 KB/s
		UploadThroughput:   84375,  // ~84 KB/s
	},
	"4g": {
		Offline:            false,
		Latency:            170,
		DownloadThroughput: 1500000, // 1.5 MB/s
		UploadThroughput:   750000,  // 750 KB/s
	},
	"wifi": {
		Offline:            false,
		Latency:            28,
		DownloadThroughput: 3750000, // 3.75 MB/s
		UploadThroughput:   1875000, // 1.875 MB/s
	},
}

// WaitRequestResult contains the result of waiting for a network request.
type WaitRequestResult struct {
	Found     bool   `json:"found"`
	URL       string `json:"url"`
	Method    string `json:"method"`
	RequestID string `json:"requestId"`
}

// WaitResponseResult contains the result of waiting for a network response.
type WaitResponseResult struct {
	Found     bool   `json:"found"`
	URL       string `json:"url"`
	Status    int    `json:"status"`
	MimeType  string `json:"mimeType,omitempty"`
	RequestID string `json:"requestId"`
}

// ResponseBodyResult contains the response body for a network request.
type ResponseBodyResult struct {
	Body          string `json:"body"`
	Base64Encoded bool   `json:"base64Encoded"`
}

// InterceptConfig configures request/response interception.
type InterceptConfig struct {
	URLPattern        string            // URL pattern to match (e.g., "*", "*.js", "https://example.com/*")
	InterceptResponse bool              // If true, intercept responses; if false, intercept requests
	Replacements      map[string]string // Text replacements to apply (old -> new)
	ResponseBody      string            // Override response body entirely (if set, Replacements ignored)
	StatusCode        int               // Override status code (0 = use original)
	Headers           map[string]string // Override/add headers
}

// --- HAR (HTTP Archive) ---

// HARLog represents an HTTP Archive log.
type HARLog struct {
	Log struct {
		Version string     `json:"version"`
		Creator HARCreator `json:"creator"`
		Entries []HAREntry `json:"entries"`
	} `json:"log"`
}

// HARCreator represents the creator of the HAR file.
type HARCreator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// HAREntry represents a single HTTP transaction.
type HAREntry struct {
	StartedDateTime string      `json:"startedDateTime"`
	Time            float64     `json:"time"`
	Request         HARRequest  `json:"request"`
	Response        HARResponse `json:"response"`
	Cache           struct{}    `json:"cache"`
	Timings         HARTimings  `json:"timings"`
}

// HARRequest represents an HTTP request.
type HARRequest struct {
	Method      string      `json:"method"`
	URL         string      `json:"url"`
	HTTPVersion string      `json:"httpVersion"`
	Headers     []HARHeader `json:"headers"`
	QueryString []HARQuery  `json:"queryString"`
	HeadersSize int         `json:"headersSize"`
	BodySize    int         `json:"bodySize"`
}

// HARResponse represents an HTTP response.
type HARResponse struct {
	Status      int         `json:"status"`
	StatusText  string      `json:"statusText"`
	HTTPVersion string      `json:"httpVersion"`
	Headers     []HARHeader `json:"headers"`
	Content     HARContent  `json:"content"`
	RedirectURL string      `json:"redirectURL"`
	HeadersSize int         `json:"headersSize"`
	BodySize    int         `json:"bodySize"`
}

// HARHeader represents an HTTP header.
type HARHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// HARQuery represents a query string parameter.
type HARQuery struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// HARContent represents the response body content.
type HARContent struct {
	Size     int    `json:"size"`
	MimeType string `json:"mimeType"`
}

// HARTimings represents the timing information.
type HARTimings struct {
	Send    float64 `json:"send"`
	Wait    float64 `json:"wait"`
	Receive float64 `json:"receive"`
}

// --- DOM & Elements ---

// QueryResult contains the result of querying for a DOM element.
type QueryResult struct {
	NodeID     int               `json:"nodeId"`
	TagName    string            `json:"tagName,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// BoundingBox represents an element's position and size.
type BoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// ElementLayout contains comprehensive layout information for an element.
type ElementLayout struct {
	Selector string                 `json:"selector"`
	TagName  string                 `json:"tagName"`
	Bounds   *BoundingBox           `json:"bounds"`
	Styles   map[string]string      `json:"styles,omitempty"`
	Children []ElementLayout        `json:"children,omitempty"`
}

// FindResult represents the result of finding text on the page.
type FindResult struct {
	Text  string `json:"text"`
	Count int    `json:"count"`
	Found bool   `json:"found"`
}

// ComputedStyleResult contains the computed style value for an element.
type ComputedStyleResult struct {
	Property string `json:"property"`
	Value    string `json:"value"`
}

// DOMSnapshotResult contains a DOM snapshot.
type DOMSnapshotResult struct {
	Documents []json.RawMessage `json:"documents"`
	Strings   []string          `json:"strings"`
}

// --- Input ---

// KeyModifiers represents keyboard modifier keys.
type KeyModifiers struct {
	Ctrl  bool
	Alt   bool
	Shift bool
	Meta  bool
}

// modifierBitmask returns the protocol modifier bitmask.
func (m KeyModifiers) modifierBitmask() int {
	mask := 0
	if m.Shift {
		mask |= 1
	}
	if m.Ctrl {
		mask |= 2
	}
	if m.Alt {
		mask |= 4
	}
	if m.Meta {
		mask |= 8
	}
	return mask
}

// SetValueResult represents the result of setting an input value.
type SetValueResult struct {
	Selector string `json:"selector"`
	Value    string `json:"value"`
}

// MouseMoveResult represents the result of moving the mouse.
type MouseMoveResult struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// DispatchEventResult contains the result of dispatching a custom event.
type DispatchEventResult struct {
	Dispatched bool   `json:"dispatched"`
	EventType  string `json:"eventType"`
	Selector   string `json:"selector"`
}

// SelectionResult contains the currently selected text on the page.
type SelectionResult struct {
	Text string `json:"text"`
}

// CaretPositionResult contains the caret position in an input element.
type CaretPositionResult struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// SwipeResult contains the result of a swipe gesture.
type SwipeResult struct {
	Swiped    bool   `json:"swiped"`
	Direction string `json:"direction"`
	Selector  string `json:"selector"`
}

// PinchResult contains the result of a pinch gesture.
type PinchResult struct {
	Pinched   bool   `json:"pinched"`
	Direction string `json:"direction"`
	Selector  string `json:"selector"`
}

// --- Device Emulation ---

// DeviceInfo contains device emulation parameters.
type DeviceInfo struct {
	Name              string  `json:"name"`
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	DeviceScaleFactor float64 `json:"deviceScaleFactor"`
	Mobile            bool    `json:"mobile"`
	UserAgent         string  `json:"userAgent"`
}

// CommonDevices is a map of common device names to their configurations.
var CommonDevices = map[string]DeviceInfo{
	"iPhone 12": {
		Name:              "iPhone 12",
		Width:             390,
		Height:            844,
		DeviceScaleFactor: 3,
		Mobile:            true,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 14_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Mobile/15E148 Safari/604.1",
	},
	"iPhone 12 Pro": {
		Name:              "iPhone 12 Pro",
		Width:             390,
		Height:            844,
		DeviceScaleFactor: 3,
		Mobile:            true,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 14_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Mobile/15E148 Safari/604.1",
	},
	"iPhone 12 Pro Max": {
		Name:              "iPhone 12 Pro Max",
		Width:             428,
		Height:            926,
		DeviceScaleFactor: 3,
		Mobile:            true,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 14_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Mobile/15E148 Safari/604.1",
	},
	"iPhone SE": {
		Name:              "iPhone SE",
		Width:             375,
		Height:            667,
		DeviceScaleFactor: 2,
		Mobile:            true,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 14_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Mobile/15E148 Safari/604.1",
	},
	"Pixel 5": {
		Name:              "Pixel 5",
		Width:             393,
		Height:            851,
		DeviceScaleFactor: 2.75,
		Mobile:            true,
		UserAgent:         "Mozilla/5.0 (Linux; Android 11; Pixel 5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.91 Mobile Safari/537.36",
	},
	"Galaxy S21": {
		Name:              "Galaxy S21",
		Width:             360,
		Height:            800,
		DeviceScaleFactor: 3,
		Mobile:            true,
		UserAgent:         "Mozilla/5.0 (Linux; Android 11; SM-G991B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.91 Mobile Safari/537.36",
	},
	"iPad": {
		Name:              "iPad",
		Width:             768,
		Height:            1024,
		DeviceScaleFactor: 2,
		Mobile:            true,
		UserAgent:         "Mozilla/5.0 (iPad; CPU OS 14_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Mobile/15E148 Safari/604.1",
	},
	"iPad Pro": {
		Name:              "iPad Pro",
		Width:             1024,
		Height:            1366,
		DeviceScaleFactor: 2,
		Mobile:            true,
		UserAgent:         "Mozilla/5.0 (iPad; CPU OS 14_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Mobile/15E148 Safari/604.1",
	},
}

// MediaFeatures represents CSS media features to emulate.
type MediaFeatures struct {
	ColorScheme   string // "light", "dark", or "" for no preference
	ReducedMotion string // "reduce", "no-preference", or "" for no preference
	ForcedColors  string // "active", "none", or "" for no preference
}

// --- Coverage ---

// CoverageResult represents JavaScript code coverage data.
type CoverageResult struct {
	Scripts []ScriptCoverage `json:"scripts"`
}

// ScriptCoverage represents coverage for a single script.
type ScriptCoverage struct {
	ScriptID string          `json:"scriptId"`
	URL      string          `json:"url"`
	Ranges   []CoverageRange `json:"ranges"`
}

// CoverageRange represents a covered range in the script.
type CoverageRange struct {
	StartOffset int `json:"startOffset"`
	EndOffset   int `json:"endOffset"`
	Count       int `json:"count"`
}

// CSSCoverageEntry represents a CSS rule usage entry.
type CSSCoverageEntry struct {
	StyleSheetID string `json:"styleSheetId"`
	StartOffset  int    `json:"startOffset"`
	EndOffset    int    `json:"endOffset"`
	Used         bool   `json:"used"`
}

// CSSCoverageResult contains CSS coverage data.
type CSSCoverageResult struct {
	Entries []CSSCoverageEntry `json:"entries"`
}

// --- Stylesheets ---

// StylesheetInfo represents information about a stylesheet.
type StylesheetInfo struct {
	StyleSheetID string `json:"styleSheetId"`
	SourceURL    string `json:"sourceURL"`
	Title        string `json:"title"`
	Disabled     bool   `json:"disabled"`
	IsInline     bool   `json:"isInline"`
	Length       int    `json:"length"`
}

// StylesheetsResult represents all stylesheets on the page.
type StylesheetsResult struct {
	Stylesheets []StylesheetInfo `json:"stylesheets"`
}

// --- Scripts ---

// ScriptInfo represents information about a script element.
type ScriptInfo struct {
	Src    string `json:"src"`
	Type   string `json:"type"`
	Async  bool   `json:"async"`
	Defer  bool   `json:"defer"`
	Inline bool   `json:"inline"`
}

// ScriptsResult represents all scripts on the page.
type ScriptsResult struct {
	Scripts []ScriptInfo `json:"scripts"`
}

// --- Forms ---

// FormInfo contains information about a form on the page.
type FormInfo struct {
	ID     string      `json:"id,omitempty"`
	Name   string      `json:"name,omitempty"`
	Action string      `json:"action,omitempty"`
	Method string      `json:"method,omitempty"`
	Inputs []InputInfo `json:"inputs"`
}

// InputInfo contains information about a form input.
type InputInfo struct {
	Name        string `json:"name,omitempty"`
	Type        string `json:"type"`
	ID          string `json:"id,omitempty"`
	Value       string `json:"value,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// --- Images ---

// ImageInfo contains information about an image element.
type ImageInfo struct {
	Src     string `json:"src"`
	Alt     string `json:"alt,omitempty"`
	Width   int    `json:"width,omitempty"`
	Height  int    `json:"height,omitempty"`
	Loading string `json:"loading,omitempty"`
}

// --- Frames ---

// FrameInfo contains information about a frame.
type FrameInfo struct {
	ID       string `json:"id"`
	ParentID string `json:"parentId,omitempty"`
	Name     string `json:"name,omitempty"`
	URL      string `json:"url"`
}

// frameTreeNode represents a node in the frame tree (used for recursive parsing).
type frameTreeNode struct {
	Frame struct {
		ID       string `json:"id"`
		ParentID string `json:"parentId"`
		Name     string `json:"name"`
		URL      string `json:"url"`
	} `json:"frame"`
	ChildFrames []frameTreeNode `json:"childFrames"`
}

// --- Accessibility ---

// AccessibilityNode represents a node in the accessibility tree.
type AccessibilityNode struct {
	NodeID      string                 `json:"nodeId"`
	Role        string                 `json:"role"`
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Value       string                 `json:"value,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	Children    []AccessibilityNode    `json:"children,omitempty"`
}

// --- Event Listeners ---

// EventListenerInfo contains information about an event listener.
type EventListenerInfo struct {
	Type       string `json:"type"`
	UseCapture bool   `json:"useCapture"`
	Passive    bool   `json:"passive"`
	Once       bool   `json:"once"`
	ScriptID   string `json:"scriptId,omitempty"`
	LineNumber int    `json:"lineNumber"`
	ColumnNumber int  `json:"columnNumber"`
}

// EventListenersResult contains the list of event listeners on an element.
type EventListenersResult struct {
	Listeners []EventListenerInfo `json:"listeners"`
}

// --- Profiling & Tracing ---

// HeapSnapshotResult contains metadata about a captured heap snapshot.
type HeapSnapshotResult struct {
	File string `json:"file"`
	Size int    `json:"size"`
}

// TraceResult contains metadata about a captured trace.
type TraceResult struct {
	File string `json:"file"`
	Size int    `json:"size"`
}

// --- Helper Functions ---

// isTruthy checks if a value is truthy in JavaScript terms.
func isTruthy(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case float64:
		return val != 0
	case string:
		return val != ""
	case []interface{}:
		return true
	case map[string]interface{}:
		return true
	default:
		return true
	}
}
