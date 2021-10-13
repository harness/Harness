// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

import (
	"context"

	dronetypes "github.com/drone/drone-go/drone"
)

type (

	// StepStore persists build step information to storage.
	StepStore interface {
		// List returns a build stage list from the datastore.
		List(context.Context, int64) ([]*dronetypes.Step, error)

		// Find returns a build stage from the datastore by ID.
		Find(context.Context, int64) (*dronetypes.Step, error)

		// FindNumber returns a stage from the datastore by number.
		FindNumber(context.Context, int64, int) (*dronetypes.Step, error)

		// Create persists a new stage to the datastore.
		Create(context.Context, *dronetypes.Step) error

		// Update persists an updated stage to the datastore.
		Update(context.Context, *dronetypes.Step) error
	}
)

// StepIsDone returns true if the step has a completed state.
func StepIsDone(s *dronetypes.Step) bool {
	switch s.Status {
	case StatusWaiting,
		StatusPending,
		StatusRunning,
		StatusBlocked:
		return false
	default:
		return true
	}
}
