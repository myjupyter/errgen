module github.com/myjupyter/errgen/example/loggers/zerolog

go 1.22

require github.com/rs/zerolog v1.33.0

require (
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/myjupyter/errgen v0.2.0
	golang.org/x/sys v0.12.0 // indirect
)

replace github.com/myjupyter/errgen => ../../..
