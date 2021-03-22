# The Automatic Workspace Archive Translator

The AWAT provides a REST API which allows consumers to translate foreign archives of chat workspaces into a Mattermost-native format, and for monitoring imports of those archives into Mattermost Cloud workspaces through the [Cloud Provisioner](https://github.com/mattermost/cloud).

# Building and Getting Started

Clone this repository and run `go install ./...` from the project root directory to build and install the AWAT to the PATH. 

This binary provides the `awat` command. The `awat` command can be used to launch either the server, with `awat server` or it can be used as a REST client to query an existing server, with the other sub-commands.

Run `awat --help` for more information.
