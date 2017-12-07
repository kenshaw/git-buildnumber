package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	isatty "github.com/mattn/go-isatty"
)

var (
	flagRev    = flag.String("rev", "HEAD", "git revision")
	flagYear   = flag.String("year", "", "start year")
	flagPrefix = flag.String("prefix", "v", "prefix")
	flagSep    = flag.String("sep", ".", "field separator")
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
	vers := getVersion(wd)
	for i := 0; i < len(vers); i++ {
		if i != 0 {
			verstr += *flagSep
		}
		verstr += strconv.Itoa(vers[i])
	}
	if verstr == "" {
		verstr = strings.Join([]string{"0", "0", "0", "0"}, *flagSep)
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
func getVersion(wd string) []int {
	// get time of commit
	at, err := git(wd, params("show", "-s", "--format=%at")...)
	if err != nil {
		return nil
	}
	z, err := strconv.ParseInt(at, 10, 64)
	if err != nil {
		return nil
	}
	t := time.Unix(z, 0)

	// set default year based on initial commit
	if *flagYear == "" {
		*flagYear = strconv.Itoa(getDefaultYear(wd))
	}

	// process year
	year, err := strconv.Atoi(*flagYear)
	if err != nil {
		return nil
	}
	year = t.Year() - year
	if year < 0 {
		year = 0
	}

	// determine count (order)
	commits, err := git(
		wd, "log", "-s", "--format=%h",
		"--since="+t.Format("2006-01-02 00:00:00 +0000"),
		"--until="+t.Format("2006-01-02 15:04:05 +0000"),
	)
	var count int
	if err == nil {
		count = len(strings.Split(commits, "\n")) - 1
	}
	return []int{year, int(t.Month()), int(t.Day()), count}
}

// getDefaultYear gets the year of the first commit.
func getDefaultYear(dir string) int {
	// get first revision
	firstRev, err := git(dir, "rev-list", "--max-parents=0", "HEAD")
	if err != nil {
		return time.Now().Year()
	}
	firstTimestamp, err := git(dir, "show", "-s", "--format=%at", firstRev)
	if err != nil {
		return time.Now().Year()
	}
	firstT, err := strconv.ParseInt(firstTimestamp, 10, 64)
	if err != nil {
		return time.Now().Year()
	}
	return time.Unix(firstT, 0).Year()
}

// params conditionally appends flagRev to v.
func params(v ...string) []string {
	if *flagRev != "" {
		return append(v, *flagRev)
	}
	return v
}

// git runs git with the passed parameters, and returns the output.
func git(dir string, params ...string) (string, error) {
	cmd := exec.Command("git", params...)
	cmd.Dir = dir
	buf, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(buf)), nil
}
