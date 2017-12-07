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
W - The number of years since start offset by a start year (ie, 2014 would be '0') of a git commit
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
