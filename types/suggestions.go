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

package types

type PipelineSuggestionsRequest struct {
	RepoRef  string
	Pipeline string
}

type PipelineGenerateRequest struct {
	Prompt  string
	RepoRef string
}

type PipelineUpdateRequest struct {
	Prompt   string
	RepoRef  string
	Pipeline string
}

type PipelineStepGenerateRequest struct {
	Prompt  string
	RepoRef string
}

type PipelineGenerateResponse struct {
	YAML string
}

type PipelineUpdateResponse struct {
	YAML string
}

type Suggestion struct {
	ID             string
	Prompt         string
	UserSuggestion string
	Suggestion     string
}

type PipelineSuggestionsResponse struct {
	Suggestions []Suggestion
}

type PipelineStepGenerateResponse struct {
	YAML string
}