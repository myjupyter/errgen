package auth

type Token struct { //nolint:govet // test type
	Value     string
	ExpiresAt int64
	Scope     string
}
