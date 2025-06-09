package supervisor

import (
	"fmt"
	"testing"

	"github.com/mattermost/awat/model"
	cloud "github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestGetPreImportPatch(t *testing.T) {
	importSize := model.Size1000String
	timeoutString := fmt.Sprintf("%d", model.S3ExtendedTimeout)

	var testCases = []struct {
		testName     string
		installation *cloud.Installation
		patch        *cloud.PatchInstallationRequest
	}{
		{
			"no updates",
			&cloud.Installation{
				Size: importSize,
				PriorityEnv: cloud.EnvVarMap{
					model.S3EnvKey:          cloud.EnvVar{Value: timeoutString},
					model.ExtractContentKey: cloud.EnvVar{Value: model.ExtractContentDisabled},
				},
			},
			nil,
		},
		{
			"all updates",
			&cloud.Installation{Size: model.SizeCloud10Users},
			&cloud.PatchInstallationRequest{
				Size: &importSize,
				PriorityEnv: cloud.EnvVarMap{
					model.S3EnvKey:          cloud.EnvVar{Value: timeoutString},
					model.ExtractContentKey: cloud.EnvVar{Value: model.ExtractContentDisabled},
				},
			},
		},
		{
			"only size",
			&cloud.Installation{
				Size: model.SizeCloud10Users,
				PriorityEnv: cloud.EnvVarMap{
					model.S3EnvKey:          cloud.EnvVar{Value: timeoutString},
					model.ExtractContentKey: cloud.EnvVar{Value: model.ExtractContentDisabled},
				},
			},
			&cloud.PatchInstallationRequest{
				Size: &importSize,
			},
		},
		{
			"only s3 timeout",
			&cloud.Installation{
				Size: importSize,
				PriorityEnv: cloud.EnvVarMap{
					model.ExtractContentKey: cloud.EnvVar{Value: model.ExtractContentDisabled},
				},
			},
			&cloud.PatchInstallationRequest{
				PriorityEnv: cloud.EnvVarMap{
					model.S3EnvKey: cloud.EnvVar{Value: timeoutString},
				},
			},
		},
		{
			"only file extract disable",
			&cloud.Installation{
				Size: importSize,
				PriorityEnv: cloud.EnvVarMap{
					model.S3EnvKey: cloud.EnvVar{Value: timeoutString},
				},
			},
			&cloud.PatchInstallationRequest{
				PriorityEnv: cloud.EnvVarMap{
					model.ExtractContentKey: cloud.EnvVar{Value: model.ExtractContentDisabled},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			logger := log.WithField("test", tc.testName)
			require.Equal(t, tc.patch, getPreImportPatch(tc.installation, logger))
		})
	}
}

func TestGetPostImportPatch(t *testing.T) {
	defaultSize := model.SizeCloud10Users
	timeoutString := fmt.Sprintf("%d", model.S3ExtendedTimeout)

	var testCases = []struct {
		testName     string
		installation *cloud.Installation
		patch        *cloud.PatchInstallationRequest
	}{
		{
			"no updates",
			&cloud.Installation{
				Size: defaultSize,
			},
			nil,
		},
		{
			"all updates",
			&cloud.Installation{
				Size: model.Size1000String,
				PriorityEnv: cloud.EnvVarMap{
					model.S3EnvKey:          cloud.EnvVar{Value: timeoutString},
					model.ExtractContentKey: cloud.EnvVar{Value: model.ExtractContentDisabled},
				},
			},
			&cloud.PatchInstallationRequest{
				Size:        &defaultSize,
				PriorityEnv: cloud.EnvVarMap{},
			},
		},
		{
			"only size",
			&cloud.Installation{
				Size: model.Size1000String,
			},
			&cloud.PatchInstallationRequest{
				Size: &defaultSize,
			},
		},
		{
			"only s3 timeout",
			&cloud.Installation{
				Size: defaultSize,
				PriorityEnv: cloud.EnvVarMap{
					model.S3EnvKey: cloud.EnvVar{Value: timeoutString},
				},
			},
			&cloud.PatchInstallationRequest{
				PriorityEnv: cloud.EnvVarMap{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			logger := log.WithField("test", tc.testName)
			require.Equal(t, tc.patch, getPostImportPatch(tc.installation, logger))
		})
	}
}
