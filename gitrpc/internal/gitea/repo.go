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
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/harness/gitness/gitrpc/internal/types"

	gitea "code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/util"
)

var lsRemoteHeadRegexp = regexp.MustCompile(`ref: refs/heads/([^\s]+)\s+HEAD`)

// InitRepository initializes a new Git repository.
func (g Adapter) InitRepository(ctx context.Context, repoPath string, bare bool) error {
	return gitea.InitRepository(ctx, repoPath, bare)
}

// SetDefaultBranch sets the default branch of a repo.
func (g Adapter) SetDefaultBranch(ctx context.Context, repoPath string,
	defaultBranch string, allowEmpty bool) error {
	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return processGiteaErrorf(err, "failed to open repository")
	}
	defer giteaRepo.Close()

	// if requested, error out if branch doesn't exist. Otherwise, blindly set it.
	if !allowEmpty && !giteaRepo.IsBranchExist(defaultBranch) {
		// TODO: ensure this returns not found error to caller
		return fmt.Errorf("branch '%s' does not exist", defaultBranch)
	}

	// change default branch
	err = giteaRepo.SetDefaultBranch(defaultBranch)
	if err != nil {
		return processGiteaErrorf(err, "failed to set new default branch")
	}

	return nil
}

// GetDefaultBranch gets the default branch of a repo.
func (g Adapter) GetDefaultBranch(ctx context.Context, repoPath string) (string, error) {
	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return "", processGiteaErrorf(err, "failed to open gitea repo")
	}
	defer giteaRepo.Close()

	// get default branch
	branch, err := giteaRepo.GetDefaultBranch()
	if err != nil {
		return "", processGiteaErrorf(err, "failed to get default branch")
	}

	return branch, nil
}

// GetRemoteDefaultBranch retrieves the default branch of a remote repository.
// If the repo doesn't have a default branch, types.ErrNoDefaultBranch is returned.
func (g Adapter) GetRemoteDefaultBranch(ctx context.Context, remoteURL string) (string, error) {
	args := []string{
		"-c", "credential.helper=",
		"ls-remote",
		"--symref",
		"-q",
		remoteURL,
		"HEAD",
	}

	cmd := gitea.NewCommand(ctx, args...)
	stdOut, _, err := cmd.RunStdString(nil)
	if err != nil {
		return "", processGiteaErrorf(err, "failed to ls remote repo")
	}

	// git output looks as follows, and we are looking for the ref that HEAD points to
	// 		ref: refs/heads/main    HEAD
	// 		46963bc7f0b5e8c5f039d50ac9e6e51933c78cdf        HEAD
	match := lsRemoteHeadRegexp.FindStringSubmatch(stdOut)
	if match == nil {
		return "", types.ErrNoDefaultBranch
	}

	return match[1], nil
}

func (g Adapter) Clone(ctx context.Context, from, to string, opts types.CloneRepoOptions) error {
	err := gitea.Clone(ctx, from, to, gitea.CloneRepoOptions{
		Timeout:       opts.Timeout,
		Mirror:        opts.Mirror,
		Bare:          opts.Bare,
		Quiet:         opts.Quiet,
		Branch:        opts.Branch,
		Shared:        opts.Shared,
		NoCheckout:    opts.NoCheckout,
		Depth:         opts.Depth,
		Filter:        opts.Filter,
		SkipTLSVerify: opts.SkipTLSVerify,
	})
	if err != nil {
		return processGiteaErrorf(err, "failed to clone repo")
	}

	return nil
}

// Sync synchronizes the repository to match the provided source.
// NOTE: This is a read operation and doesn't trigger any server side hooks.
func (g Adapter) Sync(ctx context.Context, repoPath string, source string, refSpecs []string) error {
	if len(refSpecs) == 0 {
		refSpecs = []string{"+refs/*:refs/*"}
	}
	args := []string{
		"-c", "advice.fetchShowForcedUpdates=false",
		"-c", "credential.helper=",
		"fetch",
		"--quiet",
		"--prune",
		"--atomic",
		"--force",
		"--no-write-fetch-head",
		"--no-show-forced-updates",
		source,
	}
	args = append(args, refSpecs...)

	cmd := gitea.NewCommand(ctx, args...)
	_, _, err := cmd.RunStdString(&gitea.RunOpts{
		Dir:               repoPath,
		UseContextTimeout: true,
	})
	if err != nil {
		return processGiteaErrorf(err, "failed to sync repo")
	}

	return nil
}

func (g Adapter) AddFiles(repoPath string, all bool, files ...string) error {
	err := gitea.AddChanges(repoPath, all, files...)
	if err != nil {
		return processGiteaErrorf(err, "failed to add changes")
	}

	return nil
}

// Commit commits the changes of the repository.
// NOTE: Modification of gitea implementation that supports commiter_date + author_date.
func (g Adapter) Commit(ctx context.Context, repoPath string, opts types.CommitChangesOptions) error {
	// setup environment variables used by git-commit
	// See https://git-scm.com/book/en/v2/Git-Internals-Environment-Variables
	env := []string{
		"GIT_AUTHOR_NAME=" + opts.Author.Identity.Name,
		"GIT_AUTHOR_EMAIL=" + opts.Author.Identity.Email,
		"GIT_AUTHOR_DATE=" + opts.Author.When.Format(time.RFC3339),
		"GIT_COMMITTER_NAME=" + opts.Committer.Identity.Name,
		"GIT_COMMITTER_EMAIL=" + opts.Committer.Identity.Email,
		"GIT_COMMITTER_DATE=" + opts.Committer.When.Format(time.RFC3339),
	}

	args := []string{
		"commit",
		"-m",
		opts.Message,
	}

	_, _, err := gitea.NewCommand(ctx, args...).RunStdString(&gitea.RunOpts{Dir: repoPath, Env: env})
	// No stderr but exit status 1 means nothing to commit (see gitea CommitChanges)
	if err != nil && err.Error() != "exit status 1" {
		return processGiteaErrorf(err, "failed to commit changes")
	}
	return nil
}

func (g Adapter) Push(ctx context.Context, repoPath string, opts types.PushOptions) error {
	err := Push(ctx, repoPath, opts)
	if err != nil {
		return processGiteaErrorf(err, "failed to push changes")
	}

	return nil
}

// Push pushs local commits to given remote branch.
// NOTE: Modification of gitea implementation that supports --force-with-lease.
// TODOD: return our own error types and move to above adapter.Push method
func Push(ctx context.Context, repoPath string, opts types.PushOptions) error {
	cmd := gitea.NewCommand(ctx,
		"-c", "credential.helper=",
		"push",
	)
	if opts.Force {
		cmd.AddArguments("-f")
	}
	if opts.ForceWithLease != "" {
		cmd.AddArguments(fmt.Sprintf("--force-with-lease=%s", opts.ForceWithLease))
	}
	if opts.Mirror {
		cmd.AddArguments("--mirror")
	}
	cmd.AddArguments("--", opts.Remote)

	if len(opts.Branch) > 0 {
		cmd.AddArguments(opts.Branch)
	}

	// remove credentials if there are any
	logRemote := opts.Remote
	if strings.Contains(logRemote, "://") && strings.Contains(logRemote, "@") {
		logRemote = util.SanitizeCredentialURLs(logRemote)
	}
	cmd.SetDescription(
		fmt.Sprintf(
			"pushing %s to %s (Force: %t, ForceWithLease: %s)",
			opts.Branch,
			logRemote,
			opts.Force,
			opts.ForceWithLease,
		),
	)
	var outbuf, errbuf strings.Builder

	if opts.Timeout == 0 {
		opts.Timeout = -1
	}

	err := cmd.Run(&gitea.RunOpts{
		Env:     opts.Env,
		Timeout: opts.Timeout,
		Dir:     repoPath,
		Stdout:  &outbuf,
		Stderr:  &errbuf,
	})
	if err != nil {
		switch {
		case strings.Contains(errbuf.String(), "non-fast-forward"):
			return &gitea.ErrPushOutOfDate{
				StdOut: outbuf.String(),
				StdErr: errbuf.String(),
				Err:    err,
			}
		case strings.Contains(errbuf.String(), "! [remote rejected]"):
			err := &gitea.ErrPushRejected{
				StdOut: outbuf.String(),
				StdErr: errbuf.String(),
				Err:    err,
			}
			err.GenerateMessage()
			return err
		case strings.Contains(errbuf.String(), "matches more than one"):
			err := &gitea.ErrMoreThanOne{
				StdOut: outbuf.String(),
				StdErr: errbuf.String(),
				Err:    err,
			}
			return err
		default:
			// fall through to normal error handling
		}
	}

	if errbuf.Len() > 0 && err != nil {
		return fmt.Errorf("%w - %s", err, errbuf.String())
	}

	return err
}
