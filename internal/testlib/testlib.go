// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package testlib

import (
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
)

// MakeLogger creates a log.FieldLogger that routes to tb.Log.
func MakeLogger(tb testing.TB) log.FieldLogger {
	logger := log.New()
	logger.SetOutput(&testingWriter{tb})
	logger.SetLevel(log.TraceLevel)

	return logger
}

// testingWriter is an io.Writer that writes through t.Log.
type testingWriter struct {
	tb testing.TB
}

func (tw *testingWriter) Write(b []byte) (int, error) {
	tw.tb.Log(strings.TrimSpace(string(b)))
	return len(b), nil
}
