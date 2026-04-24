package netpkg

type Addr struct { //nolint:govet // test type
	Host    string
	Port    int
	Network string
}

type Config struct {
	Timeout int
	Retries int
}
