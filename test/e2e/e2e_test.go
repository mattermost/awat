// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

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
	awatURL        string
	bucket         string
	provisionerURL string
	testDomain     string
}

func TestAWAT(t *testing.T) {
	env, err := validatedEnvironment()
	require.NoError(t, err)

	err = ensureArtifactInBucket(env.bucket)
	require.NoError(t, err)

	key := strings.TrimPrefix(slackArchive, "../") // ok this could be more robust but really who cares
	defer func() {
		err = deleteS3Object(env.bucket, key)
		require.NoError(t, err)
	}()

	provisioner := cloud.NewClient(env.provisionerURL)
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
	installation, err := provisioner.CreateInstallation(
		&cloud.CreateInstallationRequest{
			OwnerID:   "awat-e2e",
			DNS:       fmt.Sprintf("awat-e2e-%s%s", model.NewID(), env.testDomain),
			Filestore: cloud.InstallationFilestoreBifrost,
			Version:   "793e006",
			// TODO change this to EE
			// and a stable version not a random commit on my branch
			Image: "mattermost/mattermost-team-edition",
		})
	require.NoError(t, err)
	require.NotNil(t, installation)
	defer func() {
		for i := 0; i < 600; i++ {
			err := provisioner.DeleteInstallation(installation.ID)
			if err != nil {
				t.Log("nonfatal error deleting an installation during cleanup: " + err.Error())
				time.Sleep(time.Second)
			}
			return
		}
	}()
	for i := 0; i < 600; i++ {
		installation, err = provisioner.GetInstallation(installation.ID,
			&cloud.GetInstallationRequest{
				IncludeGroupConfig:          false,
				IncludeGroupConfigOverrides: false,
			})
		require.NoError(t, err)
		if installation.State == cloud.InstallationStateStable {
			break
		}
		if installation.State == cloud.InstallationStateCreationNoCompatibleClusters {
			t.Log("No compatible clusters on which to run. Did cleanup fail?")
			t.FailNow()
		}
		time.Sleep(time.Second)
	}
	require.Equal(t, installation.State, cloud.InstallationStateStable)

	client := model.NewClient(env.awatURL)

	ts, err := client.CreateTranslation(
		&model.TranslationRequest{
			Type:           "slack",
			InstallationID: installation.ID,
			Archive:        key,
			Team:           slackTeam,
		})
	require.NoError(t, err)
	require.Equal(t, model.TranslationStateRequested, ts.State)
	for i := 0; i < 300; i++ { // lazy retry loop w/ timeout
		ts, err = client.GetTranslationStatus(ts.ID)
		require.NoError(t, err)
		if ts.State != model.TranslationStateRequested {
			require.Equal(t, model.TranslationStateInProgress, ts.State)
			break
		}
		time.Sleep(time.Second)
	}

	for i := 0; i < 300; i++ {
		ts, err = client.GetTranslationStatus(ts.ID)
		require.NoError(t, err)
		if ts.State != model.TranslationStateInProgress {
			require.Equal(t, model.TranslationStateComplete, ts.State)
			break
		}
		time.Sleep(time.Second)
	}

	require.NotZero(t, ts.CompleteAt)

	ts, err = client.GetTranslationStatus(ts.ID)
	require.NoError(t, err)
	require.Equal(t, model.TranslationStateComplete, ts.State)

	importStatusList, err := client.GetImportStatusesByTranslation(ts.ID)
	require.Equal(t, 1, len(importStatusList))
	importStatus := importStatusList[0]
	if importStatus.StartAt == 0 {
		require.Equal(t, model.ImportStateRequested, importStatus.State)
	}

	for i := 0; i < 600; i++ {
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
				continue
			}
			require.NoError(t, err)
			if installation == nil {
				t.Log("wtf? the installation should exist")
				t.Fail()
			}

			require.Equal(t, cloud.InstallationStateImportInProgress, installation.State)
			break
		}
		time.Sleep(time.Second)
	}
	installation, err = provisioner.GetInstallation(
		importStatus.InstallationID,
		&cloud.GetInstallationRequest{
			IncludeGroupConfig:          false,
			IncludeGroupConfigOverrides: false,
		})
	require.NoError(t, err)
	require.Equal(t, cloud.InstallationStateImportInProgress, installation.State)

	for i := 0; i < 600; i++ {
		importStatus, err = client.GetImportStatus(importStatus.ID)
		if importStatus.State != model.ImportStateInProgress {
			require.NotZero(t, importStatus.CompleteAt)
			require.NotZero(t, importStatus.CreateAt)
			require.NotZero(t, importStatus.StartAt)
			require.Equal(t, model.ImportStateComplete, importStatus.State)
			if model.ImportStateComplete != importStatus.State {
				t.FailNow()
			}
			break
		}
		time.Sleep(time.Second)
	}
	require.Equal(t, model.ImportStateComplete, importStatus.State)

	installation, err = provisioner.GetInstallation(
		importStatus.InstallationID,
		&cloud.GetInstallationRequest{
			IncludeGroupConfig:          false,
			IncludeGroupConfigOverrides: false,
		})
	require.NoError(t, err)
	require.Equal(t, cloud.InstallationStateStable, installation.State)

	clusterInstallations, err := provisioner.GetClusterInstallations(
		&cloud.GetClusterInstallationsRequest{
			Paging:         cloud.AllPagesNotDeleted(),
			InstallationID: installation.ID,
		})
	require.Equal(t, 1, len(clusterInstallations))
	ci := clusterInstallations[0]

	output, err := provisioner.ExecClusterInstallationCLI(ci.ID, "mmctl",
		[]string{
			"--format", "json",
			"--local",
			"team", "list", `""`,
		})
	require.NoError(t, err)

	teamSearch := []*team{}
	t.Logf("team search result:\n%s", output)
	_ = json.Unmarshal(output, &teamSearch)
	require.NotEmpty(t, teamSearch)
	assert.Equal(t, slackTeam, teamSearch[0].Name)

	output, err = provisioner.ExecClusterInstallationCLI(ci.ID, "mmctl",
		[]string{
			"--format", "json",
			"--local",
			"user", "list",
		})

	userSearchResult := []*user{}
	err = json.Unmarshal(output, &userSearchResult)
	require.NoError(t, err)
	require.NotEmpty(t, userSearchResult)
	t.Logf("user search result:\n%+v", userSearchResult)
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
		t.Logf("%+v", u)
		assert.Equal(t, correctUser, u.Username)
	}

	output, err = provisioner.ExecClusterInstallationCLI(ci.ID, "mmctl",
		[]string{
			"--local",
			"--format", "json",
			"post", "list", fmt.Sprintf("%s:testing", slackTeam),
		})

	t.Logf("post list output:\n%s", output)
	postListResult, err := extractPosts(output)
	t.Logf("postListResult:\n%#v", postListResult)
	require.NoError(t, err)
	assert.NotEmpty(t, postListResult)
	assert.Equal(t, 12, len(postListResult))
	assert.Equal(t, "short message", postListResult[0].Message)
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
		awatURL:        awat,
		bucket:         s3URL,
		provisionerURL: provisionerURL,
		testDomain:     domain,
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
