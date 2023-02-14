// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//go:build e2e
// +build e2e

package e2e

/*
   The following line makes this package work with LSP in Emacs

	 (setq lsp-go-build-flags ["-tags=e2e"])

   To make this file work properly with LSP in VSCode, set the following in settings.json:
	 "gopls.env": {
				"GOFLAGS": "-tags=e2e"
		},
*/

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/mattermost/awat/internal/common"
	"github.com/mattermost/awat/model"
	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	slackArchive      = "../dummy-slack-workspace-archive.zip"
	mattermostArchive = "../dummy-mattermost-workspace-archive.zip"
	slackTeam         = "slack-import"
	mattermostTeam    = "ad-1"
)

type setting struct {
	bucket      string
	file        string
	key         string
	testDomain  string
	provisioner *cloud.Client
	awat        *model.Client
}

type completedEnvironment struct {
	t           *testing.T
	provisioner *cloud.Client
	target      *cloud.ClusterInstallation
	archiveType model.BackupType
}

func TestTwoInQuickSuccession(t *testing.T) {
	settings := setupEnvironment(t, model.SlackWorkspaceBackupType)
	installations := make([]*cloud.InstallationDTO, 2)
	for i := range installations {
		installations[i] = getInstallation(t, settings)
	}

	translationsChannel := make(chan *model.TranslationStatus, 2)
	translations := make([]*model.TranslationStatus, 2)
	for i := range installations {
		go func(i int) {
			translationsChannel <- startTranslation(t, settings.awat, installations[i], settings, model.SlackWorkspaceBackupType)
		}(i)
	}
	for i := 0; i < 2; i++ {
		translations[i] = <-translationsChannel
	}

	translationsChannel = make(chan *model.TranslationStatus, 2)
	for i := range translations {
		go func(i int) {
			translationsChannel <- waitForTranslationToSucceed(t, settings.awat, translations[i])
		}(i)
	}
	for i := 0; i < 2; i++ { // TODO write a helper function or two
		translations[i] = <-translationsChannel
	}

	done := make(chan bool, 2)
	for i := range translations {
		go func(i int) {
			waitForImportToSucceed(t, translations[i], settings.awat, settings.provisioner, installations[i])
			done <- true
		}(i)
	}
	for i := 0; i < 2; i++ {
		<-done
	}

	t.Log("validate the import data was imported properly")

	for _, installation := range installations {
		t.Logf("checking import into installation %s", installation.ID)
		clusterInstallations, err := settings.provisioner.GetClusterInstallations(
			&cloud.GetClusterInstallationsRequest{
				Paging:         cloud.AllPagesNotDeleted(),
				InstallationID: installation.ID,
			})
		require.NoError(t, err)
		require.True(t, len(clusterInstallations) > 0)
		ci := clusterInstallations[0]

		completed := &completedEnvironment{
			t:           t,
			provisioner: settings.provisioner,
			target:      ci,
			archiveType: model.SlackWorkspaceBackupType,
		}
		checkTeams(completed)
		checkChannels(completed)
		checkUsers(completed)
		checkPosts(completed)
	}
}

func TestSlackTranslationAndImport(t *testing.T) {
	settings := setupEnvironment(t, model.SlackWorkspaceBackupType)
	installation := getInstallation(t, settings)

	ts := startTranslationAndWaitForItToSucceed(
		t, settings.awat, installation,
		settings, model.SlackWorkspaceBackupType)

	waitForImportToSucceed(t, ts, settings.awat, settings.provisioner, installation)

	clusterInstallations, err := settings.provisioner.GetClusterInstallations(
		&cloud.GetClusterInstallationsRequest{
			Paging:         cloud.AllPagesNotDeleted(),
			InstallationID: installation.ID,
		})
	require.NoError(t, err)
	require.True(t, len(clusterInstallations) > 0)
	ci := clusterInstallations[0]

	completed := &completedEnvironment{
		t:           t,
		provisioner: settings.provisioner,
		target:      ci,
		archiveType: model.SlackWorkspaceBackupType,
	}
	checkTeams(completed)
	checkChannels(completed)
	checkUsers(completed)
	checkPosts(completed)
}

func TestMattermostImport(t *testing.T) {
	settings := setupEnvironment(t, model.MattermostWorkspaceBackupType)
	installation := getInstallation(t, settings)

	ts := startTranslationAndWaitForItToSucceed(
		t, settings.awat, installation,
		settings, model.MattermostWorkspaceBackupType)

	waitForImportToSucceed(t, ts, settings.awat, settings.provisioner, installation)

	clusterInstallations, err := settings.provisioner.GetClusterInstallations(
		&cloud.GetClusterInstallationsRequest{
			Paging:         cloud.AllPagesNotDeleted(),
			InstallationID: installation.ID,
		})
	require.NoError(t, err)
	require.True(t, len(clusterInstallations) > 0)
	ci := clusterInstallations[0]

	completed := &completedEnvironment{
		t:           t,
		provisioner: settings.provisioner,
		target:      ci,
		archiveType: model.MattermostWorkspaceBackupType,
	}
	checkTeams(completed)
	checkChannels(completed)
	checkUsers(completed)
	checkPosts(completed)
}

func setupEnvironment(t *testing.T, importType model.BackupType) *setting {
	t.Log("validate the environment and gather variables")

	settings, err := getSettings()
	require.NoError(t, err)

	switch importType {
	case model.MattermostWorkspaceBackupType:
		settings.file = mattermostArchive
	case model.SlackWorkspaceBackupType:
		settings.file = slackArchive
	default:
		t.FailNow()
	}

	t.Log("Upload the archive for translation")
	archiveName, err := settings.awat.UploadArchiveForTranslation(settings.file)
	require.NoError(t, err)
	uploadID := strings.TrimSuffix(archiveName, ".zip")
	err = settings.awat.WaitForUploadToComplete(uploadID)
	require.NoError(t, err)

	settings.key = archiveName

	t.Cleanup(func() {
		err = deleteS3Object(settings.bucket, settings.key)
		require.NoError(t, err)
	})

	t.Log("clean up the environment from any previously interrupted tests")

	cleanOldInstallations(t, settings.provisioner)
	return settings
}

func cleanOldInstallations(t *testing.T, provisioner *cloud.Client) {
	oldInstallations, err := provisioner.GetInstallations(
		&cloud.GetInstallationsRequest{
			Paging:                      cloud.AllPagesNotDeleted(),
			OwnerID:                     "awat-e2e",
			IncludeGroupConfig:          false,
			IncludeGroupConfigOverrides: false,
		})
	// this is just a best-effort attempt to clean up from old test runs
	// so just move on if it fails
	if err != nil {
		return
	}
	for _, i := range oldInstallations {
		err = provisioner.DeleteInstallation(i.ID)
		if err != nil {
			t.Log(err.Error())
		}
	}
}

func getInstallation(t *testing.T, env *setting) *cloud.InstallationDTO {
	t.Log("create an Installation into which to run an import")

	installation, err := env.provisioner.CreateInstallation(
		&cloud.CreateInstallationRequest{
			OwnerID:   "awat-e2e",
			DNS:       fmt.Sprintf("awat-e2e-%s%s", model.NewID(), env.testDomain),
			Filestore: cloud.InstallationFilestoreBifrost,
			Version:   "stable",
			Image:     "mattermost/mattermost-enterprise-edition",
			Affinity:  cloud.InstallationAffinityMultiTenant,
		})

	t.Cleanup(
		func() {
			retryFor(t, time.Minute*5, func() bool {
				err := env.provisioner.DeleteInstallation(installation.ID)
				if err != nil {
					t.Log("nonfatal error deleting an installation during cleanup: " + err.Error())
					return false
				}
				return true
			})
		},
	)

	require.NoError(t, err)
	require.NotNil(t, installation)

	t.Log("wait for the Installation to become stable")

	retryFor(t, time.Minute*10, func() bool {
		var err error
		installation, err = env.provisioner.GetInstallation(installation.ID,
			&cloud.GetInstallationRequest{
				IncludeGroupConfig:          false,
				IncludeGroupConfigOverrides: false,
			})
		require.NoError(t, err)
		if installation.State == cloud.InstallationStateStable {
			return true
		}
		if installation.State == cloud.InstallationStateCreationNoCompatibleClusters {
			t.Log("No compatible clusters on which to run. Did cleanup fail?")
			t.FailNow()
		}
		return false
	})
	require.Equal(t, installation.State, cloud.InstallationStateStable)

	return installation
}

func startTranslationAndWaitForItToSucceed(
	t *testing.T,
	client *model.Client,
	installation *cloud.InstallationDTO,
	settings *setting,
	archiveType model.BackupType,
) *model.TranslationStatus {

	translation := startTranslation(t, client, installation, settings, archiveType)
	return waitForTranslationToSucceed(t, client, translation)
}

func startTranslation(
	t *testing.T,
	client *model.Client,
	installation *cloud.InstallationDTO,
	settings *setting,
	archiveType model.BackupType) *model.TranslationStatus {
	t.Logf("start a new translation into installation %s", installation.ID)

	var teamName string
	if archiveType == model.SlackWorkspaceBackupType {
		teamName = slackTeam
	} else {
		teamName = ""
	}

	t.Log("create a new translation with key " + settings.key)

	ts, err := client.CreateTranslation(
		&model.TranslationRequest{
			Type:           archiveType,
			InstallationID: installation.ID,
			Archive:        settings.key,
			Team:           teamName,
		})
	require.NoError(t, err)
	require.Equal(t, model.TranslationStateRequested, ts.State)

	t.Logf("wait for translation %s to start", ts.ID)

	retryFor(t, time.Minute*5, func() bool {
		ts, err = client.GetTranslationStatus(ts.ID)
		require.NoError(t, err)
		if ts.State != model.TranslationStateRequested {
			if ts.Type == model.MattermostWorkspaceBackupType {
				// Mattermost type backups have a no-op translation step which
				// occurs very quickly. Due to this, it's difficult to time
				// the InProgress state, which can exist for a very brief
				// window, so return here, now that we know the Translation
				// was started, and we'll move on to checking if the
				// Translation completed
				return true
			}

			// Slack backups will have to be translated, however, so we
			// should be able to observe the Translation in the InProgress
			// state
			require.Equal(t, model.TranslationStateInProgress, ts.State)
			return true
		}
		return false
	})

	return ts
}

func waitForTranslationToSucceed(
	t *testing.T,
	client *model.Client,
	ts *model.TranslationStatus) *model.TranslationStatus {
	var err error

	t.Logf("wait for translation %s to complete", ts.ID)

	retryFor(t, time.Minute*5, func() bool {
		ts, err = client.GetTranslationStatus(ts.ID)
		require.NoError(t, err)
		if ts.State != model.TranslationStateInProgress {
			require.Equal(t, model.TranslationStateComplete, ts.State)
			return true
		}
		return false
	})

	require.NotZero(t, ts.CompleteAt)

	ts, err = client.GetTranslationStatus(ts.ID)
	require.NoError(t, err)
	require.Equal(t, model.TranslationStateComplete, ts.State)
	return ts
}

func waitForImportStatusChange(t *testing.T, client *model.Client, importID string, stateFrom, statusTo string) {
	t.Logf("wait for import to move from %s state to %s", stateFrom, statusTo)

	retryFor(t, time.Minute*5, func() bool {
		importStatus, err := client.GetImportStatus(importID)
		require.NoError(t, err)
		return importStatus.State != stateFrom
	})

	importStatus, err := client.GetImportStatus(importID)
	require.NoError(t, err)
	require.Equal(t, statusTo, importStatus.State)
}

func waitForInstallationStatus(t *testing.T, provisioner *cloud.Client, installationID string, state string) {
	t.Logf("wait for installation to be in %s state", state)

	retryFor(t, time.Minute*5, func() bool {
		installation, err := provisioner.GetInstallation(
			installationID,
			&cloud.GetInstallationRequest{
				IncludeGroupConfig:          false,
				IncludeGroupConfigOverrides: false,
			})
		require.NoError(t, err)
		return installation.State == state
	})
}

func waitForImportToSucceed(
	t *testing.T,
	ts *model.TranslationStatus,
	client *model.Client,
	provisioner *cloud.Client,
	installation *cloud.InstallationDTO) {
	t.Log("make sure an import is created and wait for it to start")

	importStatusList, err := client.GetImportStatusesByTranslation(ts.ID)
	require.Equal(t, 1, len(importStatusList))
	importStatus := importStatusList[0]

	// Step 1
	// It should be in the requested state adn then move to the pre-adjustment state
	waitForImportStatusChange(t, client, importStatus.ID, model.ImportStateRequested, model.ImportStateInstallationPreAdjustment)
	// Installation should move to the stable state
	waitForInstallationStatus(t, provisioner, installation.ID, cloud.InstallationStateStable)

	installation, err = provisioner.GetInstallation(
		installation.ID,
		&cloud.GetInstallationRequest{
			IncludeGroupConfig:          false,
			IncludeGroupConfigOverrides: false,
		})
	require.NoError(t, err)
	require.Equal(t, model.Size1000String, installation.Size)

	// Step 2
	// It should be in the pre-adjustment state and then move to the in-progress state
	waitForImportStatusChange(t, client, importStatus.ID, model.ImportStateInstallationPreAdjustment, model.ImportStateInProgress)
	// Installation should move to the import-in-progress state
	waitForInstallationStatus(t, provisioner, installation.ID, cloud.InstallationStateImportInProgress)
	// Installation should move to the import-complete state
	waitForInstallationStatus(t, provisioner, installation.ID, cloud.InstallationStateImportComplete)

	// Step 3
	// It should be in the in-progress state and then move to the complete state
	waitForImportStatusChange(t, client, importStatus.ID, model.ImportStateInProgress, model.ImportStateComplete)
	// Installation should move to the stable state
	waitForInstallationStatus(t, provisioner, installation.ID, cloud.InstallationStateStable)

	t.Log("Checking import status")
	importStatus, err = client.GetImportStatus(importStatus.ID)
	require.NoError(t, err)
	require.NotZero(t, importStatus.CompleteAt)
	require.NotZero(t, importStatus.CreateAt)
	require.NotZero(t, importStatus.StartAt)
	require.Empty(t, importStatus.Error)

	// Step 4
	// It should be in the complete state and then move to the post-adjustment state
	waitForImportStatusChange(t, client, importStatus.ID, model.ImportStateComplete, model.ImportStateInstallationPostAdjustment)
	// Installation should move to the stable state
	waitForInstallationStatus(t, provisioner, installation.ID, cloud.InstallationStateStable)

	installation, err = provisioner.GetInstallation(
		installation.ID,
		&cloud.GetInstallationRequest{
			IncludeGroupConfig:          false,
			IncludeGroupConfigOverrides: false,
		})
	require.NoError(t, err)
	require.Equal(t, model.SizeCloud10Users, installation.Size)

	// Step 5
	// It should be in the post-adjustment state and then move to the complete state
	waitForImportStatusChange(t, client, importStatus.ID, model.ImportStateInstallationPostAdjustment, model.ImportStateSucceeded)
}

type post struct {
	Message  string        `json:"message"`
	Metadata *postMetadata `json:"metadata"`
}

type postMetadata struct {
	Files []*postFiles `json:"files"`
}

type postFiles struct {
	Name string `json:"name"`
}

type user struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	IsBot    bool   `json:"is_bot"`
}

type team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type channel struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func (c *channel) String() string {
	if c.Type == "P" {
		// match the output from mmctl
		return fmt.Sprintf("%s (private)", c.Name)
	}

	return c.Name
}

func checkTeams(env *completedEnvironment) {
	env.t.Log("check teams")
	output, err := env.provisioner.ExecClusterInstallationCLI(env.target.ID, "mmctl",
		[]string{
			"--format", "json",
			"--local",
			"team", "list", `""`,
		})
	require.NoError(env.t, err)

	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	lines = lines[:len(lines)-1]
	output = []byte(strings.Join(lines, "\n"))

	var teamSearch []*team
	err = json.Unmarshal(output, &teamSearch)
	require.NoError(env.t, err)

	require.NotEmpty(env.t, teamSearch)

	switch env.archiveType {
	case model.MattermostWorkspaceBackupType:
	case model.SlackWorkspaceBackupType:
		assert.Equal(env.t, slackTeam, teamSearch[0].Name)
	default:
		env.t.FailNow()
	}
}

func checkChannels(env *completedEnvironment) {
	env.t.Log("check channels")
	var (
		wantedChannels []string
		wantedTeam     string
	)

	if env.archiveType == model.SlackWorkspaceBackupType {
		wantedChannels = slackChannels
		wantedTeam = slackTeam
	} else if env.archiveType == model.MattermostWorkspaceBackupType {
		wantedChannels = mattermostChannels
		wantedTeam = mattermostTeam
	} else {
		env.t.FailNow()
	}

	output, err := env.provisioner.ExecClusterInstallationCLI(env.target.ID, "mmctl",
		[]string{
			"--format", "json",
			"--local",
			"channel", "list", wantedTeam,
		})
	require.NoError(env.t, err)

	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	lines = lines[:len(lines)-1]
	output = []byte(strings.Join(lines, "\n"))

	var channelSearch []*channel
	err = json.Unmarshal(output, &channelSearch)
	require.NoError(env.t, err)

	for _, wantedChannel := range wantedChannels {
		found := false // find channels we know are in the backup
		for _, channel := range channelSearch {
			if channel.String() == wantedChannel {
				found = true
				break
			}
		}
		if !found {
			env.t.Logf("Not found: %s", wantedChannel)
			env.t.Logf("All channels found: %v", channelSearch)
		}
		require.True(env.t, found)
	}
}

func checkPosts(env *completedEnvironment) {
	env.t.Log("check posts")
	var channelName string
	var teamName string
	switch env.archiveType {
	case model.MattermostWorkspaceBackupType:
		channelName = "saepe-5"
		teamName = mattermostTeam
	case model.SlackWorkspaceBackupType:
		channelName = "testing"
		teamName = slackTeam
	default:
		env.t.FailNow()
	}

	output, err := env.provisioner.ExecClusterInstallationCLI(env.target.ID, "mmctl",
		[]string{
			"--local",
			"--format", "json",
			"post", "list", fmt.Sprintf("%s:%s", teamName, channelName),
		})
	require.NoError(env.t, err)

	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	lines = lines[:len(lines)-1]
	output = []byte(strings.Join(lines, "\n"))

	var postListResult []post
	err = json.Unmarshal(output, &postListResult)
	require.NoError(env.t, err)
	require.NotEmpty(env.t, postListResult)

	switch env.archiveType {
	case model.MattermostWorkspaceBackupType:
		assert.Equal(env.t, 20, len(postListResult))
		assert.Equal(env.t, "iusto nisi quos qui architecto tempore.\nut et fuga neque ducimus accusamus sit est sed.", postListResult[0].Message)
	case model.SlackWorkspaceBackupType:
		assert.Equal(env.t, 12, len(postListResult))
		assert.Equal(env.t, "short message", postListResult[0].Message)
	default:
		env.t.FailNow()
	}
}

func checkUsers(env *completedEnvironment) {
	env.t.Log("check users")
	var correctUsers map[string]string
	switch env.archiveType {
	case model.MattermostWorkspaceBackupType:
		correctUsers = map[string]string{}
	case model.SlackWorkspaceBackupType:
		correctUsers = correctSlackUsers
	default:
		env.t.FailNow()
	}

	output, err := env.provisioner.ExecClusterInstallationCLI(env.target.ID, "mmctl",
		[]string{
			"--format", "json",
			"--local",
			"user", "list",
		})
	require.NoError(env.t, err)

	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	lines = lines[:len(lines)-1]
	output = []byte(strings.Join(lines, "\n"))

	var userSearchResult []*user
	err = json.Unmarshal(output, &userSearchResult)
	require.NoError(env.t, err)
	require.NotEmpty(env.t, userSearchResult)

	for _, u := range userSearchResult {
		if u.IsBot {
			continue
		}
		correctUser, ok := correctUsers[u.Email]
		if !ok {
			// it is expected for the workspace to have some extra users in
			// it that aren't in the hardcoded "correct" list
			continue
		}
		assert.Equal(env.t, correctUser, u.Username)
	}
}

// if the doer returns true, consider it done, and stop retrying
func retryFor(t *testing.T, d time.Duration, doer func() bool) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if doer() {
				return
			}
		case <-timer.C:
			t.Error("failed waiting for condition")
			t.FailNow()
			return
		}
	}
}

func deleteS3Object(bucket, key string) error {

	awsConfig, err := common.NewAWSConfig()
	if err != nil {
		return err
	}

	s3client := s3.NewFromConfig(awsConfig)

	_, err = s3client.DeleteObject(
		context.TODO(),
		&s3.DeleteObjectInput{
			Bucket: &bucket,
			Key:    &key,
		})

	return err
}

func getSettings() (*setting, error) {
	s3URL := os.Getenv("AWAT_E2E_BUCKET")
	if s3URL == "" {
		return nil, errors.New("provided bucket name must not be empty; set AWAT_E2E_BUCKET")
	}

	awatURL := os.Getenv("AWAT_E2E_URL")
	if awatURL == "" {
		return nil, errors.New("provided AWAT URL must not be empty; set AWAT_E2E_URL")
	}

	domain := os.Getenv("AWAT_E2E_INSTALLATION_DOMAIN")

	if domain == "" {
		return nil, errors.New("provided target Installation DNS space must not be empty; set AWAT_E2E_INSTALLATION_DOMAIN to e.g. .dev.cloud.mattermost.com")
	}

	provisionerURL := os.Getenv("AWAT_E2E_PROVISIONER_URL")
	if provisionerURL == "" {
		return nil, errors.New("provided Provisioner URL must not be empty; set AWAT_E2E_PROVISIONER_URL")
	}

	return &setting{
		bucket:      s3URL,
		testDomain:  domain,
		provisioner: cloud.NewClient(provisionerURL),
		awat:        model.NewClient(awatURL),
	}, nil
}
