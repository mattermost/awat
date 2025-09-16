// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"os"

	log "github.com/sirupsen/logrus"
)

var logger *log.Logger

func init() {
	logger = log.New()
	logger.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	// Output to stdout instead of the default stderr.
	log.SetOutput(os.Stdout)
}
