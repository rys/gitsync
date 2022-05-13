package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

var BuildVersion string
var BuildDate string
var GitRevision string
var GitDate string
var BuildUser string

type GitsyncConfiguration struct {
	Sync []struct {
		Source   string   `json:"source_remote"`
		Target   string   `json:"target_remote"`
		Branches []string `json:"branches"`
	} `json:"sync"`
}

const gsStartupBanner string = "gitsync version %s built on %s by %s (git %s %s)\n"
const gsConfigFile string = ".gitsync.conf"
const gsConfigPathBanner string = "config path: %s\n"
const gsEndOfSync string = "gitsync has finished processing"

type GitsyncError string

const (
	gsFatalErrorCwd              GitsyncError = "can't get current working directory, Exiting..."
	gsFatalErrorDirNotExist      GitsyncError = "directory to work in does not exist. Exiting..."
	gsFatalErrorConfigNotExist   GitsyncError = "config file does not exist. Exiting..."
	gsFatalErrorConfigStat       GitsyncError = "could not stat config file. Exiting..."
	gsFatalErrorInsecureConfig   GitsyncError = "config file is not read only (r------). Exiting..."
	gsFatalErrorUnreadableConfig GitsyncError = "could not read config file records. Exiting..."
	gsFatalErrorInvalidJSON      GitsyncError = "could not process config file. Invalid JSON? Exiting..."
)

var gitsyncConfig GitsyncConfiguration

var repoRemotes = map[string]string{}
var repoBranches = map[string]string{}

var pathToRepo string = ""

var debug bool = false

func debugPrintln(msg string) {
	if debug {
		log.Println(msg)
	}
}

func debugPrintf(format string, args ...interface{}) {
	if debug {
		log.Printf(format, args...)
	}
}

// Utility functions taken from go-git and lightly modified

// CheckArgs should be used to ensure the right command line arguments are
// passed before executing an example.
func CheckArgs(arg ...string) {
	if len(os.Args) < len(arg)+1 {
		debugPrintf("Usage: %s %s", os.Args[0], strings.Join(arg, " "))
		os.Exit(1)
	}
}

// CheckIfError should be used to naively panics if an error is not nil.
func CheckIfError(err error) {
	if err == nil {
		return
	}

	debugPrintf("error: %s", err)
	os.Exit(1)
}

// End of utility functions taken from go-git and lightly modified

func checkSyncs() bool {
	for _, sync := range gitsyncConfig.Sync {
		if len(sync.Branches) >= 1 &&
			len(sync.Source) > 1 &&
			len(sync.Target) > 1 {
		} else {
			return false
		}
	}

	return true
}

func getCwd() string {
	cwd, err := os.Getwd()

	if err != nil {
		log.Fatal(gsFatalErrorCwd)
	}

	return cwd
}

func openRepoAtPath() *git.Repository {
	repo, err := git.PlainOpen(pathToRepo)
	CheckIfError(err)

	return repo
}

func collectRepoInfo() {
	repo := openRepoAtPath()

	branches, err := repo.Branches()
	CheckIfError(err)

	err = branches.ForEach(func(b *plumbing.Reference) error {
		repoBranches[b.Name().Short()] = b.Name().String()
		return nil
	})
	CheckIfError(err)

	remotes, err := repo.Remotes()
	CheckIfError(err)

	for _, remote := range remotes {
		repoRemotes[remote.Config().Name] = remote.Config().Name
	}

	if debug {
		log.Println("Repository branches:")
		log.Println(repoBranches)
		log.Println("Repository remotes:")
		log.Println(repoRemotes)
	}
}

func remoteExists(remote string) bool {
	_, exists := repoRemotes[remote]
	return exists
}

func branchExists(branch string) bool {
	_, exists := repoBranches[branch]
	return exists
}

func processSyncs() {
	for _, sync := range gitsyncConfig.Sync {
		var wouldFail = false
		debugPrintf("syncing %d branches between %s and %s\n", len(sync.Branches), sync.Source, sync.Target)

		if !remoteExists(sync.Source) {
			debugPrintf("%s source remote doesn't exist\n", sync.Source)
			wouldFail = true
		}

		if !remoteExists(sync.Target) {
			debugPrintf("%s target remote doesn't exist\n", sync.Source)
			wouldFail = true
		}

		for _, branch := range sync.Branches {
			if !branchExists(branch) {
				debugPrintf("%s branch doesn't exist\n", branch)
				wouldFail = true
			}
		}

		if wouldFail {
			debugPrintln("Attempting this sync would fail, skipping...")
			continue
		}

		debugPrintln("Processing sync")

		repo := openRepoAtPath()

		worktree, err := repo.Worktree()
		CheckIfError(err)

		for _, branch := range sync.Branches {
			var branchRef = plumbing.NewBranchReferenceName(branch)

			debugPrintf("checking out %s as %s\n", branch, branchRef)
			worktree.Checkout(&git.CheckoutOptions{Branch: branchRef})
			CheckIfError(err)

			debugPrintf("pulling changes on %s from %s\n", branch, sync.Source)
			worktree.Pull(&git.PullOptions{RemoteName: sync.Source, ReferenceName: branchRef, SingleBranch: true})
			CheckIfError(err)

			debugPrintf("pushing changes on %s to %s\n", branch, sync.Target)

			repo.Push(&git.PushOptions{
				RemoteName: sync.Target,
				RefSpecs:   []config.RefSpec{config.RefSpec(branchRef + ":" + branchRef)}})
			CheckIfError(err)
		}
	}
}

func main() {
	log.SetOutput(os.Stdout)

	var configFile string
	var printVersion bool
	var allowInsecureConfig bool

	flag.StringVar(&configFile, "config", gsConfigFile, "config file path")
	flag.BoolVar(&printVersion, "version", false, "print version and build information and exit")
	flag.BoolVar(&debug, "debug", false, "print debug information to stdout")
	flag.BoolVar(&allowInsecureConfig, "insecure", false, "allow reading an insecure config file")
	flag.StringVar(&pathToRepo, "repodir", getCwd(), "path to the git repository checkout you want to sync")
	flag.Parse()

	if printVersion {
		fmt.Printf(gsStartupBanner, BuildVersion, BuildDate, BuildUser, GitRevision, GitDate)
		os.Exit(0)
	}

	fmt.Printf(gsStartupBanner, BuildVersion, BuildDate, BuildUser, GitRevision, GitDate)
	log.Printf(gsConfigPathBanner, configFile)

	if _, err := os.ReadDir(pathToRepo); os.IsNotExist(err) {
		log.Fatal(gsFatalErrorDirNotExist)
	}

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		log.Fatal(gsFatalErrorConfigNotExist)
	}

	f, err := os.Lstat(configFile)

	if err != nil {
		log.Fatal(gsFatalErrorConfigStat)
	}

	if f.Mode() != 0400 {
		if !allowInsecureConfig {
			log.Fatal(gsFatalErrorInsecureConfig)
		}
	}

	tuples, err := ioutil.ReadFile(configFile)

	if err != nil {
		log.Fatal(gsFatalErrorUnreadableConfig)
	}

	err = json.Unmarshal(tuples, &gitsyncConfig)

	if err != nil {
		debugPrintln(err.Error())
		log.Fatal(gsFatalErrorInvalidJSON)
	}

	if checkSyncs() {
		collectRepoInfo()
		processSyncs()
		log.Println(gsEndOfSync)
		os.Exit(0)
	}

	os.Exit(1)
}
