module github.com/myjupyter/errgen/example/loggers/logrus

go 1.22

require github.com/sirupsen/logrus v1.9.3

require (
	github.com/myjupyter/errgen v0.2.0
	golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8 // indirect
)

replace github.com/myjupyter/errgen => ../../..
