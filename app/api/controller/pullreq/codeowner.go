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

package pullreq

import (
	"context"
	"errors"
	"fmt"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) CodeOwners(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
) (types.CodeOwnerEvaluation, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return types.CodeOwnerEvaluation{}, fmt.Errorf("failed to acquire access to repo: %w", err)
	}
	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, pullreqNum)
	if err != nil {
		return types.CodeOwnerEvaluation{}, fmt.Errorf("failed to get pull request by number: %w", err)
	}

	reviewers, err := c.reviewerStore.List(ctx, pr.ID)
	if err != nil {
		return types.CodeOwnerEvaluation{}, fmt.Errorf("failed to get reviewers by pr: %w", err)
	}

	ownerEvaluation, err := c.codeOwners.Evaluate(ctx, repo, pr, reviewers)
	if errors.Is(codeowners.ErrNotFound, err) {
		return types.CodeOwnerEvaluation{}, usererror.ErrNotFound
	}
	if codeowners.IsTooLargeError(err) {
		return types.CodeOwnerEvaluation{}, usererror.UnprocessableEntityf(err.Error())
	}
	if err != nil {
		return types.CodeOwnerEvaluation{}, err
	}

	return types.CodeOwnerEvaluation{
		EvaluationEntries: mapCodeOwnerEvaluation(ownerEvaluation),
		FileSha:           ownerEvaluation.FileSha,
	}, nil
}

func mapCodeOwnerEvaluation(ownerEvaluation *codeowners.Evaluation) []types.CodeOwnerEvaluationEntry {
	codeOwnerEvaluationEntries := make([]types.CodeOwnerEvaluationEntry, len(ownerEvaluation.EvaluationEntries))
	for i, entry := range ownerEvaluation.EvaluationEntries {
		ownerEvaluations := make([]types.OwnerEvaluation, len(entry.OwnerEvaluations))
		for j, owner := range entry.OwnerEvaluations {
			ownerEvaluations[j] = types.OwnerEvaluation{
				Owner:          owner.Owner,
				ReviewDecision: owner.ReviewDecision,
				ReviewSHA:      owner.ReviewSHA,
			}
		}
		codeOwnerEvaluationEntries[i] = types.CodeOwnerEvaluationEntry{
			Pattern:          entry.Pattern,
			OwnerEvaluations: ownerEvaluations,
		}
	}
	return codeOwnerEvaluationEntries
}
