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

package space

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

var (
	errParentIDNegative = usererror.BadRequest(
		"Parent ID has to be either zero for a root space or greater than zero for a child space.")
)

type CreateInput struct {
	ParentRef   string `json:"parent_ref"`
	UID         string `json:"uid"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
}

// Create creates a new space.
//
//nolint:gocognit // refactor if required
func (c *Controller) Create(
	ctx context.Context,
	session *auth.Session,
	in *CreateInput,
) (*types.Space, error) {
	parentID, err := c.getSpaceCheckAuthSpaceCreation(ctx, session, in.ParentRef)
	if err != nil {
		return nil, err
	}

	if err := c.sanitizeCreateInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}
	var space *types.Space
	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		space, err = c.createSpaceInnerInTX(ctx, session, parentID, in)
		return err
	})
	if err != nil {
		return nil, err
	}

	return space, nil
}

func (c *Controller) createSpaceInnerInTX(
	ctx context.Context,
	session *auth.Session,
	parentID int64,
	in *CreateInput,
) (*types.Space, error) {
	spacePath := in.UID
	if parentID > 0 {
		// (re-)read parent path in transaction to ensure correctness
		parentPath, err := c.spacePathStore.FindPrimaryBySpaceID(ctx, parentID)
		if err != nil {
			return nil, fmt.Errorf("failed to find primary path for parent '%d': %w", parentID, err)
		}
		spacePath = paths.Concatinate(parentPath.Value, in.UID)

		// ensure path is within accepted depth!
		err = check.PathDepth(spacePath, true)
		if err != nil {
			return nil, fmt.Errorf("path is invalid: %w", err)
		}
	}

	now := time.Now().UnixMilli()
	space := &types.Space{
		Version:     0,
		ParentID:    parentID,
		UID:         in.UID,
		Description: in.Description,
		IsPublic:    in.IsPublic,
		Path:        spacePath,
		CreatedBy:   session.Principal.ID,
		Created:     now,
		Updated:     now,
	}
	err := c.spaceStore.Create(ctx, space)
	if err != nil {
		return nil, fmt.Errorf("space creation failed: %w", err)
	}

	pathSegment := &types.SpacePathSegment{
		UID:       space.UID,
		IsPrimary: true,
		SpaceID:   space.ID,
		ParentID:  parentID,
		CreatedBy: space.CreatedBy,
		Created:   now,
		Updated:   now,
	}
	err = c.spacePathStore.InsertSegment(ctx, pathSegment)
	if err != nil {
		return nil, fmt.Errorf("failed to insert primary path segment: %w", err)
	}

	// add space membership to top level space only (as the user doesn't have inherited permissions already)
	if parentID == 0 {
		membership := &types.Membership{
			MembershipKey: types.MembershipKey{
				SpaceID:     space.ID,
				PrincipalID: session.Principal.ID,
			},
			Role: enum.MembershipRoleSpaceOwner,

			// membership has been created by the system
			CreatedBy: bootstrap.NewSystemServiceSession().Principal.ID,
			Created:   now,
			Updated:   now,
		}
		err = c.membershipStore.Create(ctx, membership)
		if err != nil {
			return nil, fmt.Errorf("failed to make user owner of the space: %w", err)
		}
	}

	return space, nil
}

func (c *Controller) getSpaceCheckAuthSpaceCreation(
	ctx context.Context,
	session *auth.Session,
	parentRef string,
) (int64, error) {
	parentRefAsID, err := strconv.ParseInt(parentRef, 10, 64)
	if (parentRefAsID <= 0 && err == nil) || (len(strings.TrimSpace(parentRef)) == 0) {
		// TODO: Restrict top level space creation - should be move to authorizer?
		if session == nil {
			return 0, fmt.Errorf("anonymous user not allowed to create top level spaces: %w", usererror.ErrUnauthorized)
		}

		return 0, nil
	}

	parentSpace, err := c.spaceStore.FindByRef(ctx, parentRef)
	if err != nil {
		return 0, fmt.Errorf("failed to get parent space: %w", err)
	}

	// create is a special case - check permission without specific resource
	scope := &types.Scope{SpacePath: parentSpace.Path}
	resource := &types.Resource{
		Type: enum.ResourceTypeSpace,
		Name: "",
	}
	if err = apiauth.Check(ctx, c.authorizer, session, scope, resource, enum.PermissionSpaceCreate); err != nil {
		return 0, fmt.Errorf("authorization failed: %w", err)
	}

	return parentSpace.ID, nil
}

func (c *Controller) sanitizeCreateInput(in *CreateInput) error {
	if len(in.ParentRef) > 0 && !c.nestedSpacesEnabled {
		// TODO (Nested Spaces): Remove once support is added
		return errNestedSpacesNotSupported
	}

	parentRefAsID, err := strconv.ParseInt(in.ParentRef, 10, 64)
	if err == nil && parentRefAsID < 0 {
		return errParentIDNegative
	}

	isRoot := false
	if (err == nil && parentRefAsID == 0) || (len(strings.TrimSpace(in.ParentRef)) == 0) {
		isRoot = true
	}

	if err := c.uidCheck(in.UID, isRoot); err != nil {
		return err
	}

	in.Description = strings.TrimSpace(in.Description)
	if err := check.Description(in.Description); err != nil { //nolint:revive
		return err
	}

	return nil
}
