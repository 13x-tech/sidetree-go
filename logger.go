package sidetree

type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{}) error
	Errorf(format string, args ...interface{}) error
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}
