package cmd

import (
	"github.com/daeuniverse/dae/config"
	"github.com/daeuniverse/dae/control"
	"github.com/sirupsen/logrus"
)

type ExternalControllerHandle interface {
	Close() error
	WebUIEnabled() bool
}

type ExternalControllerLogSink interface {
	Hook() logrus.Hook
}

type ExternalControllerRuntime struct {
	Version      string
	Config       *config.Config
	ControlPlane *control.ControlPlane
	ConfigPath   string
	Loggers      []*logrus.Logger
	LogSink      ExternalControllerLogSink
}

type ExternalControllerFactory func(runtime ExternalControllerRuntime) (ExternalControllerHandle, error)

type ExternalControllerLogSinkFactory func() ExternalControllerLogSink

var ControllerFactory ExternalControllerFactory
var ControllerLogSinkFactory ExternalControllerLogSinkFactory
