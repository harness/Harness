// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.
package trigger

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) List(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	pipelineUID string,
	filter types.ListQueryFilter,
) ([]*types.Trigger, int64, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find parent space: %w", err)
	}
	pipeline, err := c.pipelineStore.FindByUID(ctx, space.ID, pipelineUID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find pipeline: %w", err)
	}

	err = apiauth.CheckPipeline(ctx, c.authorizer, session, space.Path, pipeline.UID, enum.PermissionPipelineView)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to authorize: %w", err)
	}

	count, err := c.triggerStore.Count(ctx, pipeline.ID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count triggers in space: %w", err)
	}

	triggers, err := c.triggerStore.List(ctx, pipeline.ID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list triggers: %w", err)
	}

	return triggers, count, nil
}
