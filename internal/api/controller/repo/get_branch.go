// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"

	"github.com/harness/gitness/gitrpc"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

// GetBranch gets a repo branch.
func (c *Controller) GetBranch(ctx context.Context, session *auth.Session,
	repoRef string, branchName string) (*Branch, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("faild to find repo: %w", err)
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoView, false); err != nil {
		return nil, fmt.Errorf("access check failed: %w", err)
	}

	rpcOut, err := c.gitRPCClient.GetBranch(ctx, &gitrpc.GetBranchParams{
		ReadParams: CreateRPCReadParams(repo),
		BranchName: branchName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get branch from gitrpc: %w", err)
	}

	branch, err := mapBranch(rpcOut.Branch)
	if err != nil {
		return nil, fmt.Errorf("failed to map branch: %w", err)
	}

	return &branch, nil
}