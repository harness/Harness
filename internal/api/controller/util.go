// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package controller

import (
	"context"
	"fmt"

	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/githook"
	"github.com/harness/gitness/internal/url"
	"github.com/harness/gitness/types"
)

// TODO: this file should be in gitrpc package and should accept
// params as interface (contract)

// CreateRPCWriteParams creates base write parameters for gitrpc write operations.
// IMPORTANT: session & repo are assumed to be not nil!
// TODO: this is duplicate function from repo controller, we need to see where this
// function will be best fit.
func CreateRPCWriteParams(ctx context.Context, urlProvider *url.Provider,
	session *auth.Session, repo *types.Repository) (gitrpc.WriteParams, error) {
	// generate envars (add everything githook CLI needs for execution)
	envVars, err := githook.GenerateEnvironmentVariables(
		ctx,
		urlProvider.GetAPIBaseURLInternal(),
		repo.ID,
		session.Principal.ID,
		false,
	)
	if err != nil {
		return gitrpc.WriteParams{}, fmt.Errorf("failed to generate git hook environment variables: %w", err)
	}

	return gitrpc.WriteParams{
		Actor: gitrpc.Identity{
			Name:  session.Principal.DisplayName,
			Email: session.Principal.Email,
		},
		RepoUID: repo.GitUID,
		EnvVars: envVars,
	}, nil
}

func MapCommit(c *gitrpc.Commit) (*types.Commit, error) {
	if c == nil {
		return nil, fmt.Errorf("commit is nil")
	}

	author, err := MapSignature(&c.Author)
	if err != nil {
		return nil, fmt.Errorf("failed to map author: %w", err)
	}

	committer, err := MapSignature(&c.Committer)
	if err != nil {
		return nil, fmt.Errorf("failed to map committer: %w", err)
	}

	return &types.Commit{
		SHA:       c.SHA,
		Title:     c.Title,
		Message:   c.Message,
		Author:    *author,
		Committer: *committer,
	}, nil
}

func MapRenameDetails(c *gitrpc.RenameDetails) *types.RenameDetails {
	if c == nil {
		return nil
	}
	return &types.RenameDetails{
		OldPath:         c.OldPath,
		NewPath:         c.NewPath,
		CommitShaBefore: c.CommitShaBefore,
		CommitShaAfter:  c.CommitShaAfter,
	}
}

func MapSignature(s *gitrpc.Signature) (*types.Signature, error) {
	if s == nil {
		return nil, fmt.Errorf("signature is nil")
	}

	return &types.Signature{
		Identity: types.Identity{
			Name:  s.Identity.Name,
			Email: s.Identity.Email,
		},
		When: s.When,
	}, nil
}