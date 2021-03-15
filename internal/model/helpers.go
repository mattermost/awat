package model

import (
	"time"

	cloudModel "github.com/mattermost/mattermost-cloud/model"
)

func Timestamp() int64 {
	return time.Now().UnixNano() / 1000
}

func NewID() string {
	return cloudModel.NewID()
}
