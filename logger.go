package goen

type Logger interface {
	Print(...interface{})
	Printf(string, ...interface{})
}

type leveledLogger interface {
	Debug(...interface{})
	Debugf(string, ...interface{})
}
