// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//+build e2e

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

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/mattermost/awat/model"
	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	slackArchive = "../dummy-slack-workspace-archive.zip"
	slackTeam    = "slack-import"
)

type environment struct {
	awatURL     string
	bucket      string
	key         string
	testDomain  string
	provisioner *cloud.Client
}

func TestSlackTranslationAndImport(t *testing.T) {
	env := setupEnv(t)
	installation := getInstallation(t, env)
	client := model.NewClient(env.awatURL)
	ts := startTranslationAndWaitForItToSucceed(t, client, installation, env)
	waitForImportToSucceed(t, ts, client, env.provisioner, installation)

	t.Log("validate the import data was imported properly")

	clusterInstallations, err := env.provisioner.GetClusterInstallations(
		&cloud.GetClusterInstallationsRequest{
			Paging:         cloud.AllPagesNotDeleted(),
			InstallationID: installation.ID,
		})
	require.NoError(t, err)
	require.True(t, len(clusterInstallations) > 0)
	ci := clusterInstallations[0]

	checkTeams(t, env.provisioner, ci)
	checkChannels(t, env.provisioner, ci)
	checkUsers(t, env.provisioner, ci)
	checkPosts(t, env.provisioner, ci)
}

func setupEnv(t *testing.T) *environment {
	t.Log("validate the environment and gather variables")

	env, err := validatedEnvironment()
	require.NoError(t, err)

	err = ensureArtifactInBucket(env.bucket)
	t.Cleanup(func() {
		err = deleteS3Object(env.bucket, env.key)
		require.NoError(t, err)
	})
	require.NoError(t, err)

	env.key = strings.TrimPrefix(slackArchive, "../") // ok this could be more robust but really who cares
	t.Log("clean up the environment from any previously interrupted tests")

	cleanOldInstallations(t, env.provisioner)
	return env
}

func cleanOldInstallations(t *testing.T, provisioner *cloud.Client) {
	oldInstallations, err := provisioner.GetInstallations(
		&cloud.GetInstallationsRequest{
			Paging:                      cloud.AllPagesNotDeleted(),
			OwnerID:                     "awat-e2e",
			IncludeGroupConfig:          false,
			IncludeGroupConfigOverrides: false,
		})
	require.NoError(t, err)
	// this is just a best-effort attempt to clean up from old test runs
	// so just move on if it fails
	if err == nil {
		for _, i := range oldInstallations {
			err = provisioner.DeleteInstallation(i.ID)
			if err != nil {
				t.Log(err.Error())
			}
		}
	}
}

func getInstallation(t *testing.T, env *environment) *cloud.InstallationDTO {
	t.Log("create an Installation into which to run an import")

	installation, err := env.provisioner.CreateInstallation(
		&cloud.CreateInstallationRequest{
			OwnerID:   "awat-e2e",
			DNS:       fmt.Sprintf("awat-e2e-%s%s", model.NewID(), env.testDomain),
			Filestore: cloud.InstallationFilestoreBifrost,
			Version:   "793e006",
			// TODO change this to EE
			// and a stable version not a random commit on my branch
			Image: "mattermost/mattermost-team-edition",
		})

	t.Cleanup(
		func() {
			retryFor(time.Minute*5, func() bool {
				err := env.provisioner.DeleteInstallation(installation.ID)
				if err != nil {
					t.Log("nonfatal error deleting an installation during cleanup: " + err.Error())
					return false
				}
				return true
			})
		})

	require.NoError(t, err)
	require.NotNil(t, installation)

	t.Log("wait for the Installation to become stable")

	retryFor(time.Minute*10, func() bool {
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
	env *environment) *model.TranslationStatus {
	t.Logf("start a new translation into installation %s", installation.ID)

	ts, err := client.CreateTranslation(
		&model.TranslationRequest{
			Type:           "slack",
			InstallationID: installation.ID,
			Archive:        env.key,
			Team:           slackTeam,
		})
	require.NoError(t, err)
	require.Equal(t, model.TranslationStateRequested, ts.State)

	t.Logf("wait for translation %s to start", ts.ID)

	retryFor(time.Minute*5, func() bool {
		ts, err = client.GetTranslationStatus(ts.ID)
		require.NoError(t, err)
		if ts.State != model.TranslationStateRequested {
			require.Equal(t, model.TranslationStateInProgress, ts.State)
			return true
		}
		return false
	})

	t.Logf("wait for translation %s to complete", ts.ID)

	retryFor(time.Minute*5, func() bool {
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
	if importStatus.StartAt == 0 {
		require.Equal(t, model.ImportStateRequested, importStatus.State)
	}

	retryFor(time.Minute*10, func() bool {
		importStatus, err = client.GetImportStatus(importStatus.ID)
		if importStatus.State == model.ImportStateInProgress {
			installation, err := provisioner.GetInstallation(
				importStatus.InstallationID,
				&cloud.GetInstallationRequest{
					IncludeGroupConfig:          false,
					IncludeGroupConfigOverrides: false,
				})
			if err != nil && err.Error() == "failed with status code 409" {
				// the Installation is locked, probably by one of the
				// operations that is being tested, ha!
				return false
			}
			require.NoError(t, err)
			if installation == nil {
				t.Log("wtf? the installation should exist")
				t.Fail()
			}

			require.Equal(t, cloud.InstallationStateImportInProgress, installation.State)
			return true
		}
		return false
	})

	t.Logf("wait for import %s to complete", importStatus.ID)

	installation, err = provisioner.GetInstallation(
		importStatus.InstallationID,
		&cloud.GetInstallationRequest{
			IncludeGroupConfig:          false,
			IncludeGroupConfigOverrides: false,
		})
	require.NoError(t, err)
	require.Equal(t, cloud.InstallationStateImportInProgress, installation.State)

	retryFor(time.Minute*10, func() bool {
		importStatus, err = client.GetImportStatus(importStatus.ID)
		if importStatus.State != model.ImportStateInProgress {
			require.NotZero(t, importStatus.CompleteAt)
			require.NotZero(t, importStatus.CreateAt)
			require.NotZero(t, importStatus.StartAt)
			require.Equal(t, model.ImportStateComplete, importStatus.State)
			if model.ImportStateComplete != importStatus.State {
				t.FailNow()
			}
			return true
		}
		return false
	})
	require.Equal(t, model.ImportStateComplete, importStatus.State)

	installation, err = provisioner.GetInstallation(
		importStatus.InstallationID,
		&cloud.GetInstallationRequest{
			IncludeGroupConfig:          false,
			IncludeGroupConfigOverrides: false,
		})
	require.NoError(t, err)
	require.Equal(t, cloud.InstallationStateStable, installation.State)
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

func checkTeams(t *testing.T, provisioner *cloud.Client, ci *cloud.ClusterInstallation) {
	t.Log("check teams")

	output, err := provisioner.ExecClusterInstallationCLI(ci.ID, "mmctl",
		[]string{
			"--format", "json",
			"--local",
			"team", "list", `""`,
		})
	require.NoError(t, err)

	teamSearch := []*team{}
	_ = json.Unmarshal(output, &teamSearch)
	require.NotEmpty(t, teamSearch)
	assert.Equal(t, slackTeam, teamSearch[0].Name)
}

func checkChannels(t *testing.T, provisioner *cloud.Client, ci *cloud.ClusterInstallation) {
	t.Log("check channels")

	channelSearch := []*channel{}
	output, err := provisioner.ExecClusterInstallationCLI(ci.ID, "mmctl",
		[]string{
			"--format", "json",
			"--local",
			"channel", "list", slackTeam,
		})

	require.NoError(t, err)
	err = json.Unmarshal(output, &channelSearch)
	require.NoError(t, err)

	found := false // find a channel we know is in the backup
	for _, channel := range channelSearch {
		if channel.Name == "testing" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func checkPosts(t *testing.T, provisioner *cloud.Client, ci *cloud.ClusterInstallation) {

	t.Log("check posts")

	output, err := provisioner.ExecClusterInstallationCLI(ci.ID, "mmctl",
		[]string{
			"--local",
			"--format", "json",
			"post", "list", fmt.Sprintf("%s:testing", slackTeam),
		})

	postListResult, err := extractPosts(output)
	require.NoError(t, err)
	assert.NotEmpty(t, postListResult)
	assert.Equal(t, 12, len(postListResult))
	assert.Equal(t, "short message", postListResult[0].Message)
}

func checkUsers(t *testing.T, provisioner *cloud.Client, ci *cloud.ClusterInstallation) {
	t.Log("check users")
	output, err := provisioner.ExecClusterInstallationCLI(ci.ID, "mmctl",
		[]string{
			"--format", "json",
			"--local",
			"user", "list",
		})

	userSearchResult := []*user{}
	err = json.Unmarshal(output, &userSearchResult)
	require.NoError(t, err)
	require.NotEmpty(t, userSearchResult)
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
		assert.Equal(t, correctUser, u.Username)
	}
}

// if the doer returns true, consider it done, and stop retrying
func retryFor(d time.Duration, doer func() bool) {
	for i := float64(0); i < d.Seconds(); i++ {
		if doer() {
			break
		}
		time.Sleep(time.Second)
	}
}

func extractPosts(input []byte) ([]post, error) {
	input = []byte(strings.TrimSpace(string(input)))
	posts := []post{}
	for len(input) > 0 {
		originalLength := len(input)
		for i := 1; i <= len(input); i++ { // this is really brute force but it'll do
			var post post
			err := json.Unmarshal(input[:i], &post)
			if err == nil {
				posts = append(posts, post)
				input = input[i:]
				break
			}
		}
		if len(input) == originalLength {
			return nil, errors.Errorf("couldn't parse full input, %d characters left, %d originally", len(input), originalLength)
		}
	}
	return posts, nil
}

func deleteS3Object(bucket, key string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return err
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*5)
	defer cancelFunc()

	client := s3.NewFromConfig(cfg)
	_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})

	return err
}

func validatedEnvironment() (*environment, error) {
	s3URL := os.Getenv("AWAT_E2E_BUCKET")
	if s3URL == "" {
		return nil, errors.New("provided bucket name must not be empty; set AWAT_E2E_BUCKET")
	}

	awat := os.Getenv("AWAT_E2E_URL")
	if awat == "" {
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

	err := ensureArtifactInBucket(s3URL)
	if err != nil {
		return nil, errors.Wrap(err, "provided bucket was not valid")
	}

	return &environment{
		awatURL:     awat,
		bucket:      s3URL,
		provisioner: cloud.NewClient(provisionerURL),
		testDomain:  domain,
	}, nil
}

func uploadTestArtifact(bucketName string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return err
	}

	client := s3.NewFromConfig(cfg)
	archive, err := os.Open(slackArchive)
	defer archive.Close()
	if err != nil {
		return err
	}
	keyName := strings.TrimPrefix(archive.Name(), "../") // forgive me oh Lord for the sins I have committed

	params := &s3.PutObjectInput{
		Bucket: &bucketName,
		Key:    &keyName,
		Body:   archive,
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Minute)
	defer cancelFunc()

	_, err = client.PutObject(ctx, params)
	return err
}

func ensureArtifactInBucket(bucketName string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return err
	}

	client := s3.NewFromConfig(cfg)

	params := &s3.ListObjectsV2Input{
		Bucket: &bucketName,
	}

	p := s3.NewListObjectsV2Paginator(client, params,
		func(o *s3.ListObjectsV2PaginatorOptions) {
			if v := int32(1); v != 0 {
				o.Limit = v
			}
		})

	if !p.HasMorePages() {
		return errors.Errorf("bucket %s was empty", bucketName)
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Minute)
	defer cancelFunc()

	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return err
		}

		for _, item := range page.Contents {
			if *item.Key == bucketName {
				return nil
			}
		}
	}

	return uploadTestArtifact(bucketName)
}
