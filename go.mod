module github.com/panjf2000/gnet/v2

require (
	github.com/panjf2000/ants/v2 v2.12.1
	github.com/stretchr/testify v1.11.1
	github.com/valyala/bytebufferpool v1.0.0
	go.uber.org/zap v1.28.0
	golang.org/x/sync v0.11.0 // don't upgrade beyond v0.11.0, v0.12.0+ requires Go 1.23+
	golang.org/x/sys v0.30.0 // don't upgrade beyond v0.30.0, v0.31.0+ requires Go 1.23+
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

go 1.20
