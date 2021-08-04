# The Automatic Workspace Archive Translator

The AWAT provides a REST API which allows consumers to translate foreign archives of chat workspaces into a Mattermost-native format, and for monitoring imports of those archives into Mattermost Cloud workspaces through the [Cloud Provisioner](https://github.com/mattermost/mattermost-cloud).

# Building and Getting Started

Clone this repository and run `go install ./...` from the project root directory to build and install the AWAT to the PATH. 

This binary provides the `awat` command. The `awat` command can be used to launch either the server, with `awat server` or it can be used as a REST client to query an existing server, with the other sub-commands.

Run `awat --help` for more information.

# End-to-End Tests

Running the end-to-end tests requires the following infrastructure be present before execution:
- The [Provisioner](https://github.com/mattermost/mattermost-cloud) and its dependencies
- A Postgres instance where a logical database can be created
- An empty S3 bucket

Set the following environment variables and run `make e2e`:
- `PROVISIONER_URL` to point to where the Provisioner is listening
- `AWAT_BUCKET` to the address of the S3 bucket
- `AWAT_DATABASE` to the address of the Postgres instance

**NOTE**: The database provided by `AWAT_DATABASE` must not exist yet and the bucket at `AWAT_BUCKET` must be empty
