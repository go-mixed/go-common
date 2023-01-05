package badger

import (
	"github.com/dgraph-io/badger/v3"
	"gopkg.in/go-mixed/go-common.v1/utils"
)

type iLogger struct {
	logger utils.ILogger
}

var _ badger.Logger = (*iLogger)(nil)

func (i iLogger) Errorf(s string, i2 ...interface{}) {
	i.logger.Errorf(s, i2...)
}

func (i iLogger) Warningf(s string, i2 ...interface{}) {
	i.logger.Warnf(s, i2...)
}

func (i iLogger) Infof(s string, i2 ...interface{}) {
	i.logger.Infof(s, i2...)
}

func (i iLogger) Debugf(s string, i2 ...interface{}) {
	i.logger.Debugf(s, i2...)
}
