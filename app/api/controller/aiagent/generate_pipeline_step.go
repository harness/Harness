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

package aiagent

import (
	"context"
	"fmt"

	"github.com/harness/gitness/types"
)

type GeneratePipelineStepInput struct {
	Prompt  string `json:"prompt"`
	RepoRef string `json:"repo_ref"`
}

type PipelineStepData struct {
	StepYaml string `json:"yaml_step"`
}

type GeneratePipelineStepOutput struct {
	Status string           `json:"status"`
	Data   PipelineStepData `json:"data"`
}

func (c *Controller) GeneratePipelineStep(
	ctx context.Context,
	in *GeneratePipelineInput,
) (*GeneratePipelineStepOutput, error) {
	generateRequest := &types.PipelineStepGenerateRequest{
		Prompt:  in.Prompt,
		RepoRef: in.RepoRef,
	}

	output, err := c.intelligenceService.GenerateStep(ctx, generateRequest)
	if err != nil {
		return nil, fmt.Errorf("generate pipeline: %w", err)
	}
	return &GeneratePipelineStepOutput{
		Status: "SUCCESS",
		Data: PipelineStepData{
			StepYaml: output.YAML,
		},
	}, nil
}
