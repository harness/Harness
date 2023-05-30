// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import "github.com/google/wire"

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideClient,
)

func ProvideClient(config Config) (Interface, error) {
	return New(config)
}