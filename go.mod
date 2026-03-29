module github.com/tomyan/hubcap

go 1.25.5

require (
	github.com/gorilla/websocket v1.5.3
	github.com/tomyan/sumi v0.0.0
	github.com/tomyan/sumi-ui v0.0.0-00010101000000-000000000000
)

require (
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/term v0.39.0 // indirect
)

replace github.com/tomyan/sumi => ../sumi

replace github.com/tomyan/sumi-ui => ../sumi-ui
