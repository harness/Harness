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

package check

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type ReportInput struct {
	CheckUID string             `json:"check_uid"`
	Status   enum.CheckStatus   `json:"status"`
	Summary  string             `json:"summary"`
	Link     string             `json:"link"`
	Payload  types.CheckPayload `json:"payload"`
}

var regexpCheckUID = "^[0-9a-zA-Z-_.$]{1,127}$"
var matcherCheckUID = regexp.MustCompile(regexpCheckUID)

// Validate validates and sanitizes the ReportInput data.
func (in *ReportInput) Validate() error {
	if in.CheckUID == "" {
		return usererror.BadRequest("Status check UID is missing")
	}

	if !matcherCheckUID.MatchString(in.CheckUID) {
		return usererror.BadRequestf("Status check UID must match the regular expression: %s", regexpCheckUID)
	}

	_, ok := in.Status.Sanitize()
	if !ok {
		return usererror.BadRequest("Invalid value provided for status check status")
	}

	payloadKind, ok := in.Payload.Kind.Sanitize()
	if !ok {
		return usererror.BadRequest("Invalid value provided for the payload type")
	}
	in.Payload.Kind = payloadKind

	switch in.Payload.Kind {
	case enum.CheckPayloadKindEmpty:
		// the default payload kind (empty) does not support the payload data: clear it here
		in.Payload.Version = ""
		in.Payload.Data = []byte("{}")

		if in.Link == "" { // the link is mandatory as there is nothing in the payload
			return usererror.BadRequest("Link is missing")
		}

	case enum.CheckPayloadKindRaw, enum.CheckPayloadKindMarkdown:
		// the text payload kinds (raw and markdown) do not support the version
		if in.Payload.Version != "" {
			return usererror.BadRequestf("Payload version must be empty for the payload kind '%s'",
				in.Payload.Kind)
		}

		payloadDataJSON, err := sanitizeJSONPayload(in.Payload.Data, &types.CheckPayloadText{})
		if err != nil {
			return err
		}

		in.Payload.Data = payloadDataJSON

	case enum.CheckPayloadKindPipeline:
		return usererror.BadRequest("Kind cannot be pipeline for external checks")
	}

	return nil
}

func sanitizeJSONPayload(source json.RawMessage, data any) (json.RawMessage, error) {
	if len(source) == 0 {
		return json.Marshal(data) // marshal the empty object
	}

	decoder := json.NewDecoder(bytes.NewReader(source))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&data); err != nil {
		return nil, usererror.BadRequestf("Payload data doesn't match the required format: %s", err.Error())
	}

	buffer := bytes.NewBuffer(nil)
	buffer.Grow(512)

	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(data); err != nil {
		return nil, fmt.Errorf("failed to sanitize json payload: %w", err)
	}

	result := buffer.Bytes()

	if result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}

	return result, nil
}

// Report modifies an existing or creates a new (if none yet exists) status check report for a specific commit.
func (c *Controller) Report(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	commitSHA string,
	in *ReportInput,
	metadata map[string]string,
) (*types.Check, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoReportCommitCheck)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access access to repo: %w", err)
	}

	if errValidate := in.Validate(); errValidate != nil {
		return nil, errValidate
	}

	if !gitrpc.ValidateCommitSHA(commitSHA) {
		return nil, usererror.BadRequest("invalid commit SHA provided")
	}

	_, err = c.gitRPCClient.GetCommit(ctx, &gitrpc.GetCommitParams{
		ReadParams: gitrpc.ReadParams{RepoUID: repo.GitUID},
		SHA:        commitSHA,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to commit sha=%s: %w", commitSHA, err)
	}

	now := time.Now().UnixMilli()

	metadataJSON, _ := json.Marshal(metadata)

	statusCheckReport := &types.Check{
		CreatedBy:  session.Principal.ID,
		Created:    now,
		Updated:    now,
		RepoID:     repo.ID,
		CommitSHA:  commitSHA,
		UID:        in.CheckUID,
		Status:     in.Status,
		Summary:    in.Summary,
		Link:       in.Link,
		Payload:    in.Payload,
		Metadata:   metadataJSON,
		ReportedBy: *session.Principal.ToPrincipalInfo(),
	}

	err = c.checkStore.Upsert(ctx, statusCheckReport)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert status check result for repo=%s: %w", repo.UID, err)
	}

	return statusCheckReport, nil
}
