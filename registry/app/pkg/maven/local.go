//  Copyright 2023 Harness, Inc.
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

package maven

import (
	"context"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/store/database/dbtx"
)

const (
	ArtifactTypeLocalRegistry = "Local Registry"
)

func NewLocalRegistry(dBStore *DBStore, tx dbtx.Transactor,
) Registry {
	return &LocalRegistry{
		DBStore: dBStore,
		tx:      tx,
	}
}

type LocalRegistry struct {
	DBStore *DBStore
	tx      dbtx.Transactor
}

func (r *LocalRegistry) GetMavenArtifactType() string {
	return ArtifactTypeLocalRegistry
}

func (r *LocalRegistry) HeadArtifact(_ context.Context, _ pkg.MavenArtifactInfo) (
	responseHeaders *commons.ResponseHeaders, errs []error) {
	return nil, nil
}

func (r *LocalRegistry) GetArtifact(_ context.Context, _ pkg.MavenArtifactInfo) (
	responseHeaders *commons.ResponseHeaders, body *storage.FileReader, errs []error) {
	return nil, nil, nil
}

func (r *LocalRegistry) PutArtifact(_ context.Context, _ pkg.MavenArtifactInfo) (
	responseHeaders *commons.ResponseHeaders, errs []error) {
	return nil, nil
}