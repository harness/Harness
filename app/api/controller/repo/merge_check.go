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

package repo

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/types/enum"
)

type MergeCheck struct {
	Mergeable     bool     `json:"mergeable"`
	ConflictFiles []string `json:"conflict_files,omitempty"`
}

func (c *Controller) MergeCheck(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	diffPath string,
) (MergeCheck, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView, false)
	if err != nil {
		return MergeCheck{}, err
	}

	info, err := parseDiffPath(diffPath)
	if err != nil {
		return MergeCheck{}, err
	}

	writeParams, err := controller.CreateRPCInternalWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return MergeCheck{}, fmt.Errorf("failed to create rpc write params: %w", err)
	}

	_, err = c.gitRPCClient.Merge(ctx, &gitrpc.MergeParams{
		WriteParams: writeParams,
		BaseBranch:  info.BaseRef,
		HeadRepoUID: writeParams.RepoUID, // forks are not supported for now
		HeadBranch:  info.HeadRef,
	})
	if err != nil {
		if gitrpc.ErrorStatus(err) == gitrpc.StatusNotMergeable {
			return MergeCheck{
				Mergeable:     false,
				ConflictFiles: gitrpc.AsConflictFilesError(err),
			}, nil
		}
		return MergeCheck{}, fmt.Errorf("merge check execution failed: %w", err)
	}

	return MergeCheck{
		Mergeable: true,
	}, nil
}
