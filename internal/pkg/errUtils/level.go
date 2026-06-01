package errUtils

type ErrorLevel string

const (
	LevelDebug ErrorLevel = "DEBUG"
	LevelInfo  ErrorLevel = "INFO"
	LevelError ErrorLevel = "ERROR"
	LevelWarn  ErrorLevel = "WARN"
	LevelFatal ErrorLevel = "FATAL"
)

func (l ErrorLevel) String() string {
	return string(l)
}
