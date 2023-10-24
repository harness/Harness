// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gitea

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/gitrpc/internal/types"

	gitea "code.gitea.io/gitea/modules/git"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (g Adapter) Blame(ctx context.Context,
	repoPath, rev, file string,
	lineFrom, lineTo int,
) types.BlameReader {
	// prepare the git command line arguments
	args := make([]string, 0, 8)
	args = append(args, "blame", "--porcelain", "--encoding=UTF-8")
	if lineFrom > 0 || lineTo > 0 {
		var lines string
		if lineFrom > 0 {
			lines = strconv.Itoa(lineFrom)
		}
		if lineTo > 0 {
			lines += "," + strconv.Itoa(lineTo)
		}

		args = append(args, "-L", lines)
	}
	args = append(args, rev, "--", file)

	pipeRead, pipeWrite := io.Pipe()
	stderr := &bytes.Buffer{}
	go func() {
		var err error

		defer func() {
			// If running of the command below fails, make the pipe reader also fail with the same error.
			_ = pipeWrite.CloseWithError(err)
		}()

		cmd := gitea.NewCommand(ctx, args...)
		err = cmd.Run(&gitea.RunOpts{
			Dir:    repoPath,
			Stdout: pipeWrite,
			Stderr: stderr, // We capture stderr output in a buffer.
		})
	}()

	return &BlameReader{
		scanner:     bufio.NewScanner(pipeRead),
		commitCache: make(map[string]*types.Commit),
		errReader:   stderr, // Any stderr output will cause the BlameReader to fail.
	}
}

// blamePorcelainHeadRE is used to detect line header start in git blame porcelain output.
// It is explained here: https://www.git-scm.com/docs/git-blame#_the_porcelain_format
var blamePorcelainHeadRE = regexp.MustCompile(`^([0-9a-f]{40}|[0-9a-f]{64}) (\d+) (\d+)( (\d+))?$`)

var blamePorcelainOutOfRangeErrorRE = regexp.MustCompile(`has only \d+ lines$`)

type BlameReader struct {
	scanner     *bufio.Scanner
	lastLine    string
	commitCache map[string]*types.Commit
	errReader   io.Reader
}

func (r *BlameReader) nextLine() (string, error) {
	if line := r.lastLine; line != "" {
		r.lastLine = ""
		return line, nil
	}

	for r.scanner.Scan() {
		line := r.scanner.Text()
		if line != "" {
			return line, nil
		}
	}

	if err := r.scanner.Err(); err != nil {
		return "", err
	}

	return "", io.EOF
}

func (r *BlameReader) unreadLine(line string) {
	r.lastLine = line
}

//nolint:gocognit,nestif // it's ok
func (r *BlameReader) NextPart() (*types.BlamePart, error) {
	var commit *types.Commit
	var lines []string
	var err error

	for {
		var line string
		line, err = r.nextLine()
		if err != nil {
			break // This is the only place where we break the loop. Normally it will be the io.EOF.
		}

		if matches := blamePorcelainHeadRE.FindStringSubmatch(line); matches != nil {
			sha := matches[1]

			if commit == nil {
				commit = r.commitCache[sha]
				if commit == nil {
					commit = &types.Commit{SHA: sha}
				}

				if matches[5] != "" {
					// At index 5 there's number of lines in this section. However, the resulting
					// BlamePart might contain more than this because we join consecutive sections
					// if the commit SHA is the same.
					lineCount, _ := strconv.Atoi(matches[5])
					lines = make([]string, 0, lineCount)
				}

				continue
			}

			if sha != commit.SHA {
				r.unreadLine(line)
				r.commitCache[commit.SHA] = commit

				return &types.BlamePart{
					Commit: *commit,
					Lines:  lines,
				}, nil
			}

			continue
		}

		if commit == nil {
			// Continue reading the lines until a line header is reached.
			// This should not happen. Normal output always starts with a line header (with a commit SHA).
			continue
		}

		if line[0] == '\t' {
			// all output that contains actual file data is prefixed with tab, otherwise it's a header line
			lines = append(lines, line[1:])
			continue
		}

		parseBlameHeaders(line, commit)
	}

	// Check if there's something in the error buffer... If yes, that's the error!
	// It should contain error string from the git.
	errRaw, _ := io.ReadAll(r.errReader)
	if len(errRaw) > 0 {
		line := string(errRaw)

		if idx := bytes.IndexByte(errRaw, '\n'); idx > 0 {
			line = line[:idx] // get only the first line of the output
		}

		line = strings.TrimPrefix(line, "fatal: ") // git errors start with the "fatal: " prefix

		switch {
		case strings.Contains(line, "no such path"):
			return nil, status.Error(codes.NotFound, line)
		case strings.Contains(line, "bad revision"):
			return nil, status.Error(codes.NotFound, line)
		case blamePorcelainOutOfRangeErrorRE.MatchString(line):
			return nil, status.Error(codes.InvalidArgument, line)
		default:
			return nil, status.Error(codes.Unknown, line)
		}
	}

	// This error can happen if the command git failed to start. Triggered by pipe writer's CloseWithError call.
	if !errors.Is(err, io.EOF) {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var part *types.BlamePart

	if commit != nil && len(lines) > 0 {
		part = &types.BlamePart{
			Commit: *commit,
			Lines:  lines,
		}
	}

	return part, err
}

func parseBlameHeaders(line string, commit *types.Commit) {
	// This is the list of git blame headers that we process. Other headers we ignore.
	const (
		headerSummary       = "summary "
		headerAuthorName    = "author "
		headerAuthorMail    = "author-mail "
		headerAuthorTime    = "author-time "
		headerCommitterName = "committer "
		headerCommitterMail = "committer-mail "
		headerCommitterTime = "committer-time "
	)

	switch {
	case strings.HasPrefix(line, headerSummary):
		commit.Title = extractName(line[len(headerSummary):])
	case strings.HasPrefix(line, headerAuthorName):
		commit.Author.Identity.Name = extractName(line[len(headerAuthorName):])
	case strings.HasPrefix(line, headerAuthorMail):
		commit.Author.Identity.Email = extractEmail(line[len(headerAuthorMail):])
	case strings.HasPrefix(line, headerAuthorTime):
		commit.Author.When = extractTime(line[len(headerAuthorTime):])
	case strings.HasPrefix(line, headerCommitterName):
		commit.Committer.Identity.Name = extractName(line[len(headerCommitterName):])
	case strings.HasPrefix(line, headerCommitterMail):
		commit.Committer.Identity.Email = extractEmail(line[len(headerCommitterMail):])
	case strings.HasPrefix(line, headerCommitterTime):
		commit.Committer.When = extractTime(line[len(headerCommitterTime):])
	}
}

func extractName(s string) string {
	return s
}

// extractEmail extracts email from git blame output.
// The email address is wrapped between "<" and ">" characters.
// If "<" or ">" are not in place it returns the string as it.
func extractEmail(s string) string {
	if len(s) >= 2 && s[0] == '<' && s[len(s)-1] == '>' {
		s = s[1 : len(s)-1]
	}
	return s
}

// extractTime extracts timestamp from git blame output.
// The timestamp is UNIX time (in seconds).
// In case of an error it simply returns zero UNIX time.
func extractTime(s string) time.Time {
	milli, _ := strconv.ParseInt(s, 10, 64)
	return time.Unix(milli, 0)
}
