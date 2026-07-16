module github.com/issueye/icoo_proxy/plugins/mock

go 1.23

require github.com/issueye/icoo_proxy/common v0.0.0

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	golang.org/x/sys v0.30.0 // indirect
)

replace github.com/issueye/icoo_proxy/common => ../../common
