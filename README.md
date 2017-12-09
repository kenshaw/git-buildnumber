# About git-buildnumber

`git-buildnumber` produces standardized build numbers.

The build numbers generated here are designed to be auto-incrementing based on
the git revision/commit number, for use in API versioning, and in other places
where a numeric / sortable build number is expected to be used.

These are not meant to be used in place of standard Git short / long commit IDs
(as those are far superior), however these numbers have a very specific use
case.

The build number is formatted in a standard `W.X.Y.Z` format, such that:

```text
W - The number of years since start the start of the git repository
X - Month of year of a git commit
Y - Day of the month of a git commit commit
Z - The number (order) of the git commit of the day
```

## Installing

Install in the usual Go way:

```sh
$ go get -u github.com/brankas/git-buildnumber
```

## Usage

```text
# change to git repository
$ cd /path/to/git/repository

# get buildnumber for a repository
$ git-buildnumber
v1.2.3.4

# get buildnumber using a year offset
$ git-buildnumber -year 2014
v3.2.3.4

# help
$ git-buildnumber --help
Usage of git-buildnumber:
  -prefix string
    	prefix (default "v")
  -rev string
    	git revision (default "HEAD")
  -sep string
    	field separator (default ".")
  -year string
    	start year
```

**Note**: if `-year` is not specified, then an attempt is made to retrieve the
year of the first commit in the repository. If the year cannot be determined,
then `0` will be used.
