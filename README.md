# The Automatic Workspace Archive Translator

The AWAT provides a REST API which allows consumers to translate foreign archives of chat workspaces into a Mattermost-native format, and for monitoring imports of those archives into Mattermost Cloud workspaces through the [Cloud Provisioner](https://github.com/mattermost/mattermost-cloud).

# Building and Getting Started

Clone this repository and run `go install ./...` from the project root directory to build and install the AWAT to the PATH. 

This binary provides the `awat` command. The `awat` command can be used to launch either the server, with `awat server` or it can be used as a REST client to query an existing server, with the other sub-commands.

Run `awat --help` for more information.

# Usage

The `awat` binary provides both a server and a CLI client.

## Server

Start the `awat` server with the `awat server` subcommand. Run the command with no arguments to see the help text:

```shell
Usage:
  awat server [flags]

Flags:
      --bucket string        S3 URI where the input can be found
      --database string      Location of a Postgres database for the server to use (default "postgres://localhost:5435")
  -h, --help                 help for server
      --listen string        Local interface and port to listen on (default "localhost:8077")
      --provisioner string   Address of the Provisioner (default "http://localhost:8075")
      --workdir string       The directory to which attachments can be fetched and where the input can be extracted. In production, this will contain the location where the EBS volume is mounted. (default "/tmp/awat/workdir")
```

Running the AWAT Server requires an S3 bucket (`--bucket`), a large volume for unpacking archives (`--workdir`), a Postgres database (`--server`), and a Cloud Proivisioner to communicate with (`--provisioner`).

Example usage:

```shell
$ awat server --bucket cloud-awat-dev  --database 'postgres://postgres@localhost:5435/awat?sslmode=disable' --provisioner http://localhost:8075 --workdir /tmp/whatever
INFO[2021-12-13T15:53:50-06:00] Translation supervisor started                translation-supervisor=nsjz6hfkzjng3et918irejfz7o
INFO[2021-12-13T15:53:50-06:00] Import supervisor started                    
INFO[2021-12-13T15:53:50-06:00] Listening                                     addr="localhost:8077"
```

**N.B.** that some objects (translation outputs, etc) are retained in S3 for manual inspection / auditing during the course of normal operation (as this may be desirable for any number of reasons) and that the S3 bucket should therefore be occasionally emptied of old objects, as the AWAT will otherwise consume a lot of space.

## Client

Communicate with the AWAT using the AWAT CLI tool. 

### How to Import a Workspace into a Mattermost Cloud Installation from Slack or Mattermost On-Premise Installation
Obtain an archive of the source Workspace and start a translation with a command like the following, and the workspace will be imported into the destination Installation immediately after the translation (if necessary) is complete:

```shell
$ awat translation start --installation-id  qgbrng6mubfu5jfiwjtgt1rqmh --type mattermost --filename ./test/dummy-mattermost-workspace-archive.zip --upload --server http://localhost:8077
```
If the `--upload` flag is specified, the file to be translated to Mattermost format and imported into the Workspace will be uploaded to S3 from the local filesystem. 

Otherwise, upload the file yourself to S3 using the `aws` cli tool or web interface, and then provide a path relative to the root of the S3 bucket:
```shell
$ awat translation start --installation-id 39edz9g15b8858u8uybdm9kyco --filename 'dummy-slack-workspace-archive.zip' --type slack --team myTeam
```

The first example shows a Mattermost export being imported into a destination Installation with ID qgbrng6mubfu5jfiwjtgt1rqmh, and the file is local and being uploaded. The AWAT Server is specified to be running on `localhost:8077`.

The second example shows a Slack type translation being imported into a destination Installation with ID 39edz9g15b8858u8uybdm9kyco and the filename is presumed to already be in S3 at the given location. The AWAT Server is assumed to be running on `localhost:8077`. A destination team name is specified because Slack workspaces do not have a concept of teams, so the user must provide one before the translation is performed.

The translation process can be monitored using `awat translation list` and `awat translation get`. Run `list` with no arguments to see all running translations. Run `get` with no arguments to see the help text.

When the translation is complete, if it is successful, an import job will be created and performed. 

`awat import list` will show all imports. 
`awat import get` will show detailed information about a single import.

### Restart an Import or Import an Existing Archive Into A New Workspace

Use `awat import get` to discover the `Resource` that was being imported into the new Workspace.
In order to import that file into a new Workspace, use e.g. `awat translation start --type mattermost --filename value-from-Resource-field-above`. Don't forget to specify the additional required arguments for the previous command; only the ones relevant to the example are shown here.

### Manually Remediate a Failed Import

If an Import fails for a reason that can be remediated by manually editing the backup archive being imported, download the file in the `Resource` field of the output of `awat import get --import-id` for the failed import, manually apply the remediations to the downloaded archive, and start a new import with `awat translation start --type mattermost --filename ./remediated-filename --upload`. Don't forget to specify the rest of the arguments for `awat translation start` here as necessary.

## Troubleshooting

### Failed Translations

The AWAT's logs will hold the most information regarding failed translations. These are most likely to occur if a third-party changes the format of their Workspace, e.g. if the Slack export changes, a translation may fail at this stage and the AWAT logs will be informative.
To restart a Translation, simply create a new one.

### Failed Imports

After translation, an import may fail for any number of reasons. The AWAT will receive some error message from the Provisioner, but the Provisioner's logs may be informative and the Mattermost Workspace with the failed import will also have important information in its logs.

# End-to-End Tests

Running the end-to-end tests requires the following infrastructure be present before execution:
- The [Provisioner](https://github.com/mattermost/mattermost-cloud) and its dependencies
- A Postgres instance where a logical database can be created
- An empty S3 bucket

Set the following environment variables and run `make e2e`:
- `AWAT_E2E_INSTALLATION_DOMAIN` set to the domain to use for testing installations, e.g. ".dev.cloud.mattermost.com"
- `AWAT_E2E_URL` set to the AWAT endpoint
- `AWAT_E2E_PROVISIONER_URL` set to the Provisioner endpoint
- `AWAT_E2E_BUCKET` set to the address of the S3 bucket
