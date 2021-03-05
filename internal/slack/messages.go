package slack

import (
	"archive/zip"
	"log"
	"os"
	"sort"

	mmetl "github.com/mattermost/mmetl/services/slack"
)

func addFileToPost(file *mmetl.SlackFile, uploads map[string]*zip.File, post *mmetl.IntermediatePost, attachmentsDir string) {
}

func TransformSlackMessages(inputFilePath, outputFilePath, team string) error {
	// input file
	fileReader, err := os.Open(inputFilePath)
	if err != nil {
		return err
	}
	defer fileReader.Close()

	zipFileInfo, err := fileReader.Stat()
	if err != nil {
		return err
	}

	zipReader, err := zip.NewReader(fileReader, zipFileInfo.Size())
	if err != nil || zipReader.File == nil {
		return err
	}

	slackExport, err := mmetl.ParseSlackExportFile(team, zipReader, true)
	if err != nil {
		return err
	}

	intermediate := new(mmetl.Intermediate)

	// ToDo: change log lines to something more meaningful
	log.Println("Transforming users")
	mmetl.TransformUsers(slackExport.Users, intermediate)

	log.Println("Transforming channels")
	if err := mmetl.TransformAllChannels(slackExport, intermediate); err != nil {
		return err
	}

	log.Println("Populating user memberships")
	mmetl.PopulateUserMemberships(intermediate)

	log.Println("Populating channel memberships")
	mmetl.PopulateChannelMemberships(intermediate)

	log.Println("Transforming posts")
	if err := transformPosts(slackExport, intermediate, ""); err != nil {
		return err
	}

	if err = mmetl.Export(team, intermediate, outputFilePath); err != nil {
		return err
	}

	log.Println("Transformation succeeded!!")
	return nil
}

func transformPosts(slackExport *mmetl.SlackExport, intermediate *mmetl.Intermediate, attachmentsDir string) error {
	newGroupChannels := []*mmetl.IntermediateChannel{}
	newDirectChannels := []*mmetl.IntermediateChannel{}
	channelsByOriginalName := buildChannelsByOriginalNameMap(intermediate)

	resultPosts := []*mmetl.IntermediatePost{}
	for originalChannelName, channelPosts := range slackExport.Posts {
		channel, ok := channelsByOriginalName[originalChannelName]
		if !ok {
			log.Printf("--- Couldn't find channel %s referenced by posts", originalChannelName)
			continue
		}

		timestamps := make(map[int64]bool)
		sort.Slice(channelPosts, func(i, j int) bool {
			return mmetl.SlackConvertTimeStamp(channelPosts[i].TimeStamp) < mmetl.SlackConvertTimeStamp(channelPosts[j].TimeStamp)
		})
		threads := map[string]*mmetl.IntermediatePost{}

		for _, post := range channelPosts {
			switch {
			// plain message that can have files attached
			case post.IsPlainMessage():
				if post.User == "" {
					log.Println("Slack Import: Unable to import the message as the user field is missing.")
					continue
				}
				author := intermediate.UsersById[post.User]
				if author == nil {
					log.Println("Slack Import: Unable to add the message as the Slack user does not exist in Mattermost. user=" + post.User)
					continue
				}
				newPost := &mmetl.IntermediatePost{
					User:     author.Username,
					Channel:  channel.Name,
					Message:  post.Text,
					CreateAt: mmetl.SlackConvertTimeStamp(post.TimeStamp),
				}
				if post.File != nil {
					addFileToPost(post.File, slackExport.Uploads, newPost, attachmentsDir)
				} else if post.Files != nil {
					for _, file := range post.Files {
						addFileToPost(file, slackExport.Uploads, newPost, attachmentsDir)
					}
				} else {
					log.Println("post.File and post.Files were nil")
				}

				mmetl.AddPostToThreads(post, newPost, threads, channel, timestamps)

			// file comment
			case post.IsFileComment():
				if post.Comment == nil {
					log.Println("Slack Import: Unable to import the message as it has no comments.")
					continue
				}
				if post.Comment.User == "" {
					log.Println("Slack Import: Unable to import the message as the user field is missing.")
					continue
				}
				author := intermediate.UsersById[post.Comment.User]
				if author == nil {
					log.Println("Slack Import: Unable to add the message as the Slack user does not exist in Mattermost. user=" + post.Comment.User)
					continue
				}
				newPost := &mmetl.IntermediatePost{
					User:     author.Username,
					Channel:  channel.Name,
					Message:  post.Comment.Comment,
					CreateAt: mmetl.SlackConvertTimeStamp(post.TimeStamp),
				}

				mmetl.AddPostToThreads(post, newPost, threads, channel, timestamps)

			// bot message
			case post.IsBotMessage():
				// log.Println("Slack Import: bot messages are not yet supported")
				break

			// channel join/leave messages
			case post.IsJoinLeaveMessage():
				// log.Println("Slack Import: Join/Leave messages are not yet supported")
				break

			// me message
			case post.IsMeMessage():
				// log.Println("Slack Import: me messages are not yet supported")
				break

			// change topic message
			case post.IsChannelTopicMessage():
				if post.User == "" {
					log.Println("Slack Import: Unable to import the message as the user field is missing.")
					continue
				}
				author := intermediate.UsersById[post.User]
				if author == nil {
					log.Println("Slack Import: Unable to add the message as the Slack user does not exist in Mattermost. user=" + post.User)
					continue
				}

				newPost := &mmetl.IntermediatePost{
					User:     author.Username,
					Channel:  channel.Name,
					Message:  post.Text,
					CreateAt: mmetl.SlackConvertTimeStamp(post.TimeStamp),
					// Type:     model.POST_HEADER_CHANGE,
				}

				mmetl.AddPostToThreads(post, newPost, threads, channel, timestamps)

			// change channel purpose message
			case post.IsChannelPurposeMessage():
				if post.User == "" {
					log.Println("Slack Import: Unable to import the message as the user field is missing.")
					continue
				}
				author := intermediate.UsersById[post.User]
				if author == nil {
					log.Println("Slack Import: Unable to add the message as the Slack user does not exist in Mattermost. user=" + post.User)
					continue
				}

				newPost := &mmetl.IntermediatePost{
					User:     author.Username,
					Channel:  channel.Name,
					Message:  post.Text,
					CreateAt: mmetl.SlackConvertTimeStamp(post.TimeStamp),
					// Type:     model.POST_HEADER_CHANGE,
				}

				mmetl.AddPostToThreads(post, newPost, threads, channel, timestamps)

			// change channel name message
			case post.IsChannelNameMessage():
				if post.User == "" {
					log.Println("Slack Import: Unable to import the message as the user field is missing.")
					continue
				}
				author := intermediate.UsersById[post.User]
				if author == nil {
					log.Println("Slack Import: Unable to add the message as the Slack user does not exist in Mattermost. user=" + post.User)
					continue
				}

				newPost := &mmetl.IntermediatePost{
					User:     author.Username,
					Channel:  channel.Name,
					Message:  post.Text,
					CreateAt: mmetl.SlackConvertTimeStamp(post.TimeStamp),
					// Type:     model.POST_DISPLAYNAME_CHANGE,
				}

				mmetl.AddPostToThreads(post, newPost, threads, channel, timestamps)

			default:
				log.Println("Slack Import: Unable to import the message as its type is not supported. post_type=" + post.Type + " post_subtype=" + post.SubType)
			}
		}

		channelPosts := []*mmetl.IntermediatePost{}
		for _, post := range threads {
			channelPosts = append(channelPosts, post)
		}
		resultPosts = append(resultPosts, channelPosts...)
	}

	intermediate.Posts = resultPosts
	intermediate.GroupChannels = append(intermediate.GroupChannels, newGroupChannels...)
	intermediate.DirectChannels = append(intermediate.DirectChannels, newDirectChannels...)

	return nil
}

func buildChannelsByOriginalNameMap(intermediate *mmetl.Intermediate) map[string]*mmetl.IntermediateChannel {
	channelsByName := map[string]*mmetl.IntermediateChannel{}
	for _, channel := range intermediate.PublicChannels {
		channelsByName[channel.OriginalName] = channel
	}
	for _, channel := range intermediate.PrivateChannels {
		channelsByName[channel.OriginalName] = channel
	}
	for _, channel := range intermediate.GroupChannels {
		channelsByName[channel.OriginalName] = channel
	}
	for _, channel := range intermediate.DirectChannels {
		channelsByName[channel.OriginalName] = channel
	}
	return channelsByName
}
