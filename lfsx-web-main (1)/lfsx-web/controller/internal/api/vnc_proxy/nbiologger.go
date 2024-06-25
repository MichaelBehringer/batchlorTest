package vnc

import "gitea.hama.de/LFS/go-logger"

// nbioLogger is a logger adapter for the nbio engine to the RPJosh go-logger
type nbioLogger struct {
	*logger.Logger
}

func (l nbioLogger) SetLevel(level int) {
	// Do nothing
}

func (l nbioLogger) Warn(message string, parameters ...any) {
	l.Logger.Log(logger.LevelWarning, message, parameters...)
}
func (l nbioLogger) Debug(message string, parameters ...any) {
	l.Logger.Log(logger.LevelDebug, message, parameters...)
}
func (l nbioLogger) Error(message string, parameters ...any) {
	l.Logger.Log(logger.LevelError, message, parameters...)
}
func (l nbioLogger) Info(message string, parameters ...any) {
	l.Logger.Log(logger.LevelInfo, message, parameters...)
}

func newNbioLogger() nbioLogger {
	// All messages of nio should be shifted by 1 because it does not have a trace level
	printLevel := logger.GetGlobalLogger().PrintLevel + 1
	logLevel := logger.GetGlobalLogger().LogLevel + 1

	log := logger.NewLoggerWithFile(
		&logger.Logger{
			LogLevel:      logLevel,
			PrintLevel:    printLevel,
			ColoredOutput: logger.GetGlobalLogger().ColoredOutput,
			PrintSource:   true,
		}, logger.GetGlobalLogger(),
	)

	return nbioLogger{Logger: log}
}
