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
	"encoding/base64"
	"fmt"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// CommitFileAction holds file operation data.
type CommitFileAction struct {
	Action   gitrpc.FileAction        `json:"action"`
	Path     string                   `json:"path"`
	Payload  string                   `json:"payload"`
	Encoding enum.ContentEncodingType `json:"encoding"`
	SHA      string                   `json:"sha"`
}

// CommitFilesOptions holds the data for file operations.
type CommitFilesOptions struct {
	Title     string             `json:"title"`
	Message   string             `json:"message"`
	Branch    string             `json:"branch"`
	NewBranch string             `json:"new_branch"`
	Actions   []CommitFileAction `json:"actions"`
}

func (c *Controller) CommitFiles(ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *CommitFilesOptions,
) (types.CommitFilesResponse, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush, false)
	if err != nil {
		return types.CommitFilesResponse{}, err
	}

	if in.NewBranch == "" {
		isSpaceOwner, err := apiauth.IsSpaceAdmin(ctx, c.authorizer, session, repo)
		if err != nil {
			return types.CommitFilesResponse{}, err
		}

		protectionRules, err := c.protectionManager.ForRepository(ctx, repo.ID)
		if err != nil {
			return types.CommitFilesResponse{},
				fmt.Errorf("failed to fetch protection rules for the repository: %w", err)
		}

		_, violations, err := protectionRules.CanPush(ctx, protection.CanPushInput{
			Actor:        &session.Principal,
			IsSpaceOwner: isSpaceOwner,
			Repo:         repo,
			BranchNames:  []string{in.Branch},
		})
		if err != nil {
			return types.CommitFilesResponse{}, fmt.Errorf("failed to verify protection rules for git push: %w", err)
		}

		if protection.IsCritical(violations) {
			return types.CommitFilesResponse{RuleViolations: violations}, nil
		}
	}

	actions := make([]gitrpc.CommitFileAction, len(in.Actions))
	for i, action := range in.Actions {
		var rawPayload []byte
		switch action.Encoding {
		case enum.ContentEncodingTypeBase64:
			rawPayload, err = base64.StdEncoding.DecodeString(action.Payload)
			if err != nil {
				return types.CommitFilesResponse{}, fmt.Errorf("failed to decode base64 payload: %w", err)
			}
		case enum.ContentEncodingTypeUTF8:
			fallthrough
		default:
			// by default we treat content as is
			rawPayload = []byte(action.Payload)
		}

		actions[i] = gitrpc.CommitFileAction{
			Action:  action.Action,
			Path:    action.Path,
			Payload: rawPayload,
			SHA:     action.SHA,
		}
	}

	// Create internal write params. Note: This will skip the pre-commit protection rules check.
	writeParams, err := controller.CreateRPCInternalWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return types.CommitFilesResponse{}, fmt.Errorf("failed to create RPC write params: %w", err)
	}

	now := time.Now()
	commit, err := c.gitRPCClient.CommitFiles(ctx, &gitrpc.CommitFilesParams{
		WriteParams:   writeParams,
		Title:         in.Title,
		Message:       in.Message,
		Branch:        in.Branch,
		NewBranch:     in.NewBranch,
		Actions:       actions,
		Committer:     rpcIdentityFromPrincipal(bootstrap.NewSystemServiceSession().Principal),
		CommitterDate: &now,
		Author:        rpcIdentityFromPrincipal(session.Principal),
		AuthorDate:    &now,
	})
	if err != nil {
		return types.CommitFilesResponse{}, err
	}
	return types.CommitFilesResponse{
		CommitID: commit.CommitID,
	}, nil
}
