package log

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/TwilyName/website/config"
)

type Channels struct {
	Access *log.Logger
	Error  *log.Logger
	Stdout *log.Logger
	Stderr *log.Logger
}

var logger = Channels{
	Stdout: log.New(os.Stdout, "INFO: ", log.LstdFlags),
	Stderr: log.New(os.Stderr, "ERR:", log.LstdFlags),
}

func Initialize() {
	logout := os.Stdout
	logerr := os.Stderr
	var err error

	if config.Get().Log.Access != "" {
		logout, err = os.OpenFile(config.Get().Log.Access, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0660)
		if err != nil {
			log.Fatalf("cannot open file: %v", err)
		}
	}

	if config.Get().Log.Error != "" {
		logerr, err = os.OpenFile(config.Get().Log.Error, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0660)
		if err != nil {
			log.Fatalf("cannot open file: %v", err)
		}
	}

	logger.Access = log.New(logout, "", 0)
	logger.Error = log.New(logerr, "", 0)
}

func InitializeSignalHandler() {
	sighup := make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGHUP)

	go func() {
		for {
			_ = <-sighup
			Stdout().Print("SIGHUP received, reopening log files")
			Initialize()
		}
	}()
}

func Access() *log.Logger {
	return logger.Access
}
func Error() *log.Logger {
	return logger.Error
}
func Stdout() *log.Logger {
	return logger.Stdout
}
func Stderr() *log.Logger {
	return logger.Stderr
}
