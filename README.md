# drivefs
A simple FUSE filesystem for Google Drive on Linux.

Currently supports:
* Mounting Google Drive to a directory in READONLY mode only.
* Opening non Google Apps files (i.e., Google Docs, Sheets etc will not open)
* It downloads files only when opened and has support for primitive 'caching'.

NOTE: Still in infancy mode, proper logging, doc and other features (sync, upload, etc.)
will come later. Pull requests welcome!

### Pre-reqs
* Golang version 1.16+
  * Built using [FUSE library by Bazil](https://github.com/bazil/fuse)
* A credentials json for a Google Cloud project
  * Easiest way to get one is to [go to this link](https://developers.google.com/drive/api/v3/quickstart/go#step_1_turn_on_the),
  click on "Enable the Drive API" button, follow the steps and download the file.

### Usage
* Download the source, cd to the drivefs dir and build it `go build drivefs.go`
* `$ ./drivefs -credsfile <path to credentials.json> -mntpoint <path to mount dir> -tokenfile <path to oauth token.json>`
* On first run, it will print a link to authorize the app and fetch an oauth refresh token.
  * NOTE: Since the authorization is for your own app created in the pre-reqs step, it should be fine to proceed,
  however, make sure the file is stored in a safe location on the machine after download.
* It will fetch basic file/dir information (not the actual contents), and the mounted directory can be browsed using
a regular file manager/shell.
