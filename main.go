package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	isatty "github.com/mattn/go-isatty"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

var (
	flagRev    = flag.String("rev", "HEAD", "git revision")
	flagYear   = flag.String("year", "", "start year offset")
	flagPrefix = flag.String("prefix", "v", "prefix")
	flagSep    = flag.String("sep", ".", "field separator")
	flagShort  = flag.Bool("short", false, "trim last \"<sep>0\" from version")
)

func init() {
	time.Local = time.UTC
}

func main() {
	var err error

	flag.Parse()

	var wd string
	switch {
	case flag.NArg() == 0:
		wd, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case flag.NArg() == 1:
		wd = os.Args[len(os.Args)-1]
	case flag.NArg() > 1:
		fmt.Fprintln(os.Stderr, "error: cannot specify more than one git directory")
		os.Exit(1)
	}

	// format version
	var verstr string
	vers, err := getVersion(wd)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for i := 0; i < len(vers); i++ {
		if i != 0 {
			verstr += *flagSep
		}
		verstr += strconv.Itoa(vers[i])
	}
	if verstr == "" {
		verstr = strings.Join([]string{"0", "0", "0", "0"}, *flagSep)
	}
	if *flagShort {
		verstr = strings.TrimSuffix(verstr, *flagSep+"0")
	}

	// determine line end
	var extra string
	if isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		extra = "\n"
	}

	// output
	fmt.Fprintf(os.Stdout, "%s%s%s", *flagPrefix, verstr, extra)
}

// getVersion determines the version.
func getVersion(wd string) ([]int, error) {
	repo, err := git.PlainOpen(wd)
	if err != nil {
		return nil, err
	}
	hash, err := repo.ResolveRevision(plumbing.Revision(*flagRev))
	if err != nil {
		return nil, err
	}

	// get time of commit
	commit, err := repo.CommitObject(*hash)
	if err != nil {
		return nil, err
	}
	t := commit.Author.When.UTC()

	var year int
	// set default year based on initial commit
	if *flagYear != "" {
		// process year
		if year, err = strconv.Atoi(*flagYear); err != nil {
			return nil, err
		}
	} else {
		if year, err = getDefaultYear(commit); err != nil {
			return nil, err
		}
	}

	year = t.Year() - year
	if year < 0 {
		year = 0
	}

	// determine count (order) from the start of the day at UTC
	dayStart := commit.Committer.When.UTC().Truncate(24 * time.Hour)
	count, err := countCommits(dayStart, commit)
	if err != nil {
		return nil, err
	}
	count-- // 0-based
	return []int{year, int(t.Month()), t.Day(), count}, nil
}

// getDefaultYear gets the year of the first commit.
func getDefaultYear(commit *object.Commit) (int, error) {
	// The order doesn't matter, so do pre-order.
	iter := object.NewCommitPreorderIter(commit, nil, nil)
	defer iter.Close()
	oldestTime := time.Time{}
	err := iter.ForEach(func(commit *object.Commit) error {
		if len(commit.ParentHashes) > 0 {
			// not a root commit
			return nil
		}
		t := commit.Author.When
		if oldestTime == (time.Time{}) || t.Before(oldestTime) {
			oldestTime = t
		}
		return nil
	})
	return oldestTime.UTC().Year(), err
}

var stopIter = fmt.Errorf("stopped commit iter")

// countCommits counts the number of commits from a certain time until a
// certain commit.
func countCommits(from time.Time, until *object.Commit) (int, error) {
	// We want to visit the commit history like "git log", so do
	// post-order to see merged commits right after their merge
	// commit.
	iter := object.NewCommitPostorderIter(until, nil)
	defer iter.Close()
	count := 0
	err := iter.ForEach(func(commit *object.Commit) error {
		if commit.Committer.When.Before(from) {
			// Past the date; stop iterating.
			return stopIter
		}
		count++
		return nil
	})
	if err == stopIter {
		err = nil
	}
	return count, err
}
