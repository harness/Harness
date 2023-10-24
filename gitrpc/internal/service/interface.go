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

package service

import (
	"context"
	"io"

	"github.com/harness/gitness/gitrpc/enum"
	"github.com/harness/gitness/gitrpc/internal/types"
)

// GitAdapter for accessing git commands from gitea.
type GitAdapter interface {
	InitRepository(ctx context.Context, path string, bare bool) error
	Config(ctx context.Context, repoPath, key, value string) error
	SetDefaultBranch(ctx context.Context, repoPath string, defaultBranch string, allowEmpty bool) error
	GetDefaultBranch(ctx context.Context, repoPath string) (string, error)
	GetRemoteDefaultBranch(ctx context.Context, remoteURL string) (string, error)
	Clone(ctx context.Context, from, to string, opts types.CloneRepoOptions) error
	AddFiles(repoPath string, all bool, files ...string) error
	Commit(ctx context.Context, repoPath string, opts types.CommitChangesOptions) error
	Push(ctx context.Context, repoPath string, opts types.PushOptions) error
	ReadTree(ctx context.Context, repoPath, ref string, w io.Writer, args ...string) error
	GetTreeNode(ctx context.Context, repoPath string, ref string, treePath string) (*types.TreeNode, error)
	ListTreeNodes(ctx context.Context, repoPath string, ref string, treePath string) ([]types.TreeNode, error)
	PathsDetails(ctx context.Context, repoPath string, ref string, paths []string) ([]types.PathDetails, error)
	GetSubmodule(ctx context.Context, repoPath string, ref string, treePath string) (*types.Submodule, error)
	GetBlob(ctx context.Context, repoPath string, sha string, sizeLimit int64) (*types.BlobReader, error)
	WalkReferences(ctx context.Context, repoPath string, handler types.WalkReferencesHandler,
		opts *types.WalkReferencesOptions) error
	GetCommit(ctx context.Context, repoPath string, ref string) (*types.Commit, error)
	GetCommits(ctx context.Context, repoPath string, refs []string) ([]types.Commit, error)
	ListCommits(ctx context.Context, repoPath string,
		ref string, page int, limit int, filter types.CommitFilter) ([]types.Commit, []types.PathRenameDetails, error)
	ListCommitSHAs(ctx context.Context, repoPath string,
		ref string, page int, limit int, filter types.CommitFilter) ([]string, error)
	GetLatestCommit(ctx context.Context, repoPath string, ref string, treePath string) (*types.Commit, error)
	GetFullCommitID(ctx context.Context, repoPath, shortID string) (string, error)
	GetAnnotatedTag(ctx context.Context, repoPath string, sha string) (*types.Tag, error)
	GetAnnotatedTags(ctx context.Context, repoPath string, shas []string) ([]types.Tag, error)
	CreateTag(ctx context.Context, repoPath string, name string, targetSHA string, opts *types.CreateTagOptions) error
	GetBranch(ctx context.Context, repoPath string, branchName string) (*types.Branch, error)
	GetCommitDivergences(ctx context.Context, repoPath string,
		requests []types.CommitDivergenceRequest, max int32) ([]types.CommitDivergence, error)
	GetRef(ctx context.Context, repoPath string, reference string) (string, error)
	UpdateRef(ctx context.Context, repoPath, reference, newValue, oldValue string) error
	CreateTemporaryRepoForPR(ctx context.Context, reposTempPath string, pr *types.PullRequest,
		baseBranch, trackingBranch string) (types.TempRepository, error)
	Merge(ctx context.Context, pr *types.PullRequest, mergeMethod enum.MergeMethod, baseBranch, trackingBranch string,
		tmpBasePath string, mergeMsg string, env []string, identity *types.Identity) error
	GetMergeBase(ctx context.Context, repoPath, remote, base, head string) (string, string, error)
	Blame(ctx context.Context, repoPath, rev, file string, lineFrom, lineTo int) types.BlameReader
	Sync(ctx context.Context, repoPath string, source string) error

	//
	// Diff operations
	//

	GetDiffTree(ctx context.Context,
		repoPath,
		baseBranch,
		headBranch string) (string, error)

	RawDiff(ctx context.Context,
		repoPath,
		base,
		head string,
		mergeBase bool,
		w io.Writer) error

	CommitDiff(ctx context.Context,
		repoPath,
		sha string,
		w io.Writer) error

	DiffShortStat(ctx context.Context,
		repoPath string,
		baseRef string,
		headRef string,
		useMergeBase bool) (types.DiffShortStat, error)

	GetDiffHunkHeaders(ctx context.Context,
		repoPath string,
		targetRef string,
		sourceRef string) ([]*types.DiffFileHunkHeaders, error)

	DiffCut(ctx context.Context,
		repoPath string,
		targetRef string,
		sourceRef string,
		path string,
		params types.DiffCutParams) (types.HunkHeader, types.Hunk, error)

	MatchFiles(ctx context.Context,
		repoPath string,
		ref string,
		dirPath string,
		regExpDef string,
		maxSize int) ([]types.FileContent, error)

	DiffFileStat(
		ctx context.Context,
		repoPath string,
		baseRef string,
		headRef string) ([]string, error)
}
