package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
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
	flagRev     = flag.String("rev", "HEAD", "git revision")
	flagYear    = flag.String("year", "", "start year offset")
	flagPrefix  = flag.String("prefix", "v", "prefix")
	flagSep     = flag.String("sep", ".", "field separator")
	flagShort   = flag.Bool("short", false, "trim last \"<sep>0\" from version")
	flagInverse = flag.String("inverse", "", "string to inverse")
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	flag.Parse()

	var err error
	var wd string
	switch {
	case flag.NArg() == 0:
		wd, err = os.Getwd()
		if err != nil {
			return err
		}
	case flag.NArg() == 1:
		wd = os.Args[len(os.Args)-1]
	case flag.NArg() > 1:
		return errors.New("cannot specify more than one git directory")
	}

	// determine line end
	var extra string
	if isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		extra = "\n"
	}

	if *flagInverse == "" {
		var vers []string
		if vers, err = getVersion(wd); err != nil {
			return err
		}
		if n := len(vers) - 1; *flagShort && vers[n] == "0" {
			vers = vers[:n]
		}
		fmt.Fprint(os.Stdout, *flagPrefix, strings.Join(vers, *flagSep), extra)
	} else {
		var hash string
		if hash, err = getInverse(wd); err != nil {
			return err
		}
		fmt.Fprint(os.Stdout, hash, extra)
	}

	return nil
}

// getVersion determines the version.
func getVersion(wd string) ([]string, error) {
	var err error

	// open repository
	repo, err := git.PlainOpen(wd)
	if err != nil {
		return nil, err
	}

	// find commit
	var hash *plumbing.Hash
	var commit *object.Commit
	if hash, err = repo.ResolveRevision(plumbing.Revision(*flagRev)); err != nil {
		// could not resolve rev, so search for associated object
		if h := plumbing.NewHash(*flagRev); h != plumbing.ZeroHash {
			var obj object.Object
			if obj, err = repo.Object(plumbing.AnyObject, h); err != nil {
				return nil, errors.New("invalid ref")
			}
			var ok bool
			if commit, ok = obj.(*object.Commit); !ok {
				return nil, errors.New("ref is not a commit ref")
			}
		}
	} else {
		// rev flag was blank or was valid gitrev
		commit, err = repo.CommitObject(*hash)
		if err != nil {
			return nil, err
		}
	}

	// empty repository
	if commit == nil {
		return []string{"0", "0", "0", "0"}, nil
	}

	// determine year offset
	year, err := determineYearOffset(commit)
	if err != nil {
		return nil, err
	}

	// clamp date to UTC and year to 0
	date := commit.Committer.When.UTC()
	year = date.Year() - year
	if year < 0 {
		year = 0
	}

	// determine order
	order, err := commitOrder(commit, date.Truncate(24*time.Hour))
	if err != nil {
		return nil, err
	}

	return []string{
		strconv.Itoa(year),
		strconv.Itoa(int(date.Month())),
		strconv.Itoa(date.Day()),
		strconv.Itoa(order),
	}, nil
}

// getInverse determines the hash based on the supplied flags.
func getInverse(wd string) (string, error) {
	var err error

	// open repository
	repo, err := git.PlainOpen(wd)
	if err != nil {
		return "", err
	}

	// get head
	head, err := repo.Head()
	if err != nil {
		return "", err
	}

	// get commit
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return "", err
	}

	// parse inverse string
	year, month, day, order, err := parseInverse(commit)
	if err != nil {
		return "", err
	}

	iter := object.NewCommitPostorderIter(commit, nil)
	defer iter.Close()

	// find closest matching commit
	var c *object.Commit
	for c, err = iter.Next(); err == nil; c, err = iter.Next() {
		d := c.Committer.When.UTC()
		var n int
		n, err = commitOrder(c, d.Truncate(24*time.Hour))
		if err != nil {
			return "", err
		}
		if d.Year() == year && d.Month() == month && d.Day() == day && order == n {
			return c.Hash.String(), nil
		}
	}
	if err != nil && err != io.EOF {
		return "", err
	}

	return "", errors.New("could not find matching version")
}

// parseInverse parses the inverse flag.
func parseInverse(commit *object.Commit) (int, time.Month, int, int, error) {
	var err error
	vers := strings.Split(strings.TrimPrefix(*flagInverse, *flagPrefix), *flagSep)
	if len(vers) == 3 {
		vers = append(vers, "0")
	}
	ver := make([]int, 4)
	for i := range vers {
		ver[i], err = strconv.Atoi(vers[i])
		if err != nil {
			return 0, 0, 0, 0, errors.New("invalid inverse version")
		}
	}

	// determine year offset
	var year int
	year, err = determineYearOffset(commit)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return ver[0] + year, time.Month(ver[1]), ver[2], ver[3], nil
}

// determineYearOffset determines the offset for the given commit.
func determineYearOffset(commit *object.Commit) (int, error) {
	var err error
	var year int
	if *flagYear != "" {
		if year, err = strconv.Atoi(*flagYear); err != nil {
			return 0, err
		}
	} else {
		var oldest *object.Commit
		if oldest, err = oldestParent(commit); err != nil {
			return 0, err
		}
		year = oldest.Committer.When.UTC().Year()
	}
	return year, nil
}

// oldestParent retrieves the oldest parent of commit.
func oldestParent(commit *object.Commit) (*object.Commit, error) {
	iter := object.NewCommitPostorderIter(commit, nil)
	defer iter.Close()

	oldest, date := commit, commit.Committer.When.UTC()
	var c *object.Commit
	var err error
	for c, err = iter.Next(); err == nil; c, err = iter.Next() {
		if len(c.ParentHashes) > 0 {
			continue
		}
		if d := c.Committer.When.UTC(); d.Before(date) {
			oldest, date = c, d
		}
	}
	if err != nil && err != io.EOF {
		return nil, err
	}
	return oldest, nil
}

// commitOrder determines the order (ie, number) of commits made before the
// commit on the same date (based on UTC time).
//
// Note: zero ordered.
func commitOrder(commit *object.Commit, date time.Time) (int, error) {
	iter := object.NewCommitPostorderIter(commit, nil)
	defer iter.Close()

	var count int
	var c *object.Commit
	var err error
	for c, err = iter.Next(); err == nil; c, err = iter.Next() {
		if d := c.Committer.When.UTC(); d.Equal(date) || d.After(date) {
			count++
		}
	}
	if err != nil && err != io.EOF {
		return -1, err
	}
	return count - 1, nil
}
