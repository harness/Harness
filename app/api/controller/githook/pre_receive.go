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

package githook

import (
	"context"
	"fmt"
	"strings"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/githook"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"golang.org/x/exp/slices"
)

// PreReceive executes the pre-receive hook for a git repository.
//
//nolint:revive // not yet fully implemented
func (c *Controller) PreReceive(
	ctx context.Context,
	session *auth.Session,
	repoID int64,
	principalID int64,
	internal bool,
	in githook.PreReceiveInput,
) (*githook.Output, error) {
	output := &githook.Output{}

	repo, err := c.getRepoCheckAccess(ctx, session, repoID, enum.PermissionRepoEdit)
	if err != nil {
		return nil, err
	}

	refUpdates := groupRefsByAction(in.RefUpdates)

	if slices.Contains(refUpdates.branches.deleted, repo.DefaultBranch) {
		// Default branch mustn't be deleted.
		output.Error = ptr.String(usererror.ErrDefaultBranchCantBeDeleted.Error())
		return output, nil
	}

	if internal {
		// It's an internal call, so no need to verify protection rules.
		return output, nil
	}

	// TODO: Remove the dummy session and use the real session, once that has been done and the session has a value.
	dummySession := &auth.Session{
		Principal: types.Principal{ID: principalID},
		Metadata:  nil,
	}

	err = c.checkProtectionRules(ctx, dummySession, repo, refUpdates, output)
	if err != nil {
		return nil, fmt.Errorf("failed to check protection rules: %w", err)
	}

	return output, nil
}

func (c *Controller) checkProtectionRules(
	ctx context.Context,
	session *auth.Session,
	repo *types.Repository,
	refUpdates changedRefs,
	output *githook.Output,
) error {
	isSpaceOwner, err := apiauth.IsSpaceAdmin(ctx, c.authorizer, session, repo)
	if err != nil {
		return err
	}

	protectionRules, err := c.protectionManager.ForRepository(ctx, repo.ID)
	if err != nil {
		return fmt.Errorf("failed to fetch protection rules for the repository: %w", err)
	}

	var ruleViolations []types.RuleViolations

	// TODO: protectionRules.CanCreateBranch
	// if len(refUpdates.branches.created) > 0 {}

	// TODO: protectionRules.CanDeleteBranch
	// if len(refUpdates.branches.deleted) > 0 {}

	if len(refUpdates.branches.updated) > 0 {
		_, violations, err := protectionRules.CanPush(ctx, protection.CanPushInput{
			Actor:        &session.Principal,
			IsSpaceOwner: isSpaceOwner,
			Repo:         repo,
			BranchNames:  refUpdates.branches.updated,
		})
		if err != nil {
			return fmt.Errorf("failed to verify protection rules for git push: %w", err)
		}

		ruleViolations = append(ruleViolations, violations...)
	}

	var criticalViolation bool

	for _, ruleViolation := range ruleViolations {
		criticalViolation = criticalViolation || ruleViolation.IsCritical()
		for _, violation := range ruleViolation.Violations {
			message := fmt.Sprintf("Rule %q violation: %s", ruleViolation.Rule.UID, violation.Message)
			output.Messages = append(output.Messages, message)
		}
	}

	if criticalViolation {
		output.Error = ptr.String("Blocked by protection rules.")
	}

	return nil
}

type changes struct {
	created []string
	deleted []string
	updated []string
}

func (c *changes) groupByAction(refUpdate githook.ReferenceUpdate, name string) {
	switch {
	case refUpdate.Old == types.NilSHA:
		c.created = append(c.created, name)
	case refUpdate.New == types.NilSHA:
		c.deleted = append(c.deleted, name)
	default:
		c.updated = append(c.updated, name)
	}
}

type changedRefs struct {
	branches changes
	tags     changes
	other    changes
}

func groupRefsByAction(refUpdates []githook.ReferenceUpdate) (c changedRefs) {
	for _, refUpdate := range refUpdates {
		switch {
		case strings.HasPrefix(refUpdate.Ref, gitReferenceNamePrefixBranch):
			branchName := refUpdate.Ref[len(gitReferenceNamePrefixBranch):]
			c.branches.groupByAction(refUpdate, branchName)
		case strings.HasPrefix(refUpdate.Ref, gitReferenceNamePrefixTag):
			tagName := refUpdate.Ref[len(gitReferenceNamePrefixTag):]
			c.tags.groupByAction(refUpdate, tagName)
		default:
			c.other.groupByAction(refUpdate, refUpdate.Ref)
		}
	}
	return
}
