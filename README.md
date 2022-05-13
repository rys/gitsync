# gitsync

`gitsync` is a tool that allows you to sync a git repository's remotes. It was designed to allow me to sync public 
 repositories to a private CI system without having to manually check for changes and mirror them to the internal CI system.

It can sync multiple branches per remote pair, and can be run inside and outside of a current working directory that contains a git repository.

It is designed to be defensive at startup, checking as much state as it can to make sure the sync operation can proceed.

It is _NOT_ designed to be run inside a working tree that you're hacking on. Setup the repo on disk somewhere with your remote pairs and then leave it alone, so it can cleanly fast forward from the source and push that cleanly into the target.

It checks out each branch before syncing it, in order to pull any changes.

Look at [`gitsync.conf`](gitsync.conf) for an example configuration.

# Libraries

`gitsync` uses the _excellent_ [go-git](https://github.com/go-git/go-git) golang git library. 

# Usage

- `-help` print usage help
- `-config` config file path (defaults to `.gitsync.conf`)
- `-debug` print debug information to stdout
- `-insecure` allow reading an insecure config file
- `-repodir` path to the git repository checkout you want to sync (defaults to `$CWD`)
- `-version` print version and build information and exit

# License

[MIT licensed](LICENSE)