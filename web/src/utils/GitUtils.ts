/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Code copied from https://github.com/vweevers/is-git-ref-name-valid and
// https://github.com/vweevers/is-git-branch-name-valid (MIT, © Vincent Weevers)
// Last updated for git 2.29.0.

import type { IconName } from '@harnessio/icons'
import type {
  EnumWebhookTrigger,
  OpenapiContentInfo,
  OpenapiDirContent,
  OpenapiGetContentOutput,
  TypesCommit,
  TypesPullReq,
  TypesRepository
} from 'services/code'

export interface GitInfoProps {
  repoMetadata: TypesRepository
  gitRef: string
  resourcePath: string
  resourceContent: OpenapiGetContentOutput
  commitRef: string
  commits: TypesCommit[]
  pullRequestMetadata: TypesPullReq
}
export interface RepoFormData {
  name: string
  description: string
  license: string
  defaultBranch: string
  gitignore: string
  addReadme: boolean
  isPublic: RepoVisibility
}
export interface ImportFormData {
  repoUrl: string
  username: string
  password: string
  name: string
  description: string
  isPublic: RepoVisibility
}

export interface ExportFormData {
  accountId: string
  token: string
  organization: string
  name: string
}

export interface ExportFormDataExtended extends ExportFormData {
  repoCount: number
}

export interface ImportSpaceFormData {
  gitProvider: string
  username: string
  password: string
  name: string
  description: string
  organization: string
  host: string
}

export enum RepoVisibility {
  PUBLIC = 'public',
  PRIVATE = 'private'
}

export enum RepoCreationType {
  IMPORT = 'import',
  CREATE = 'create'
}

export enum SpaceCreationType {
  IMPORT = 'import',
  CREATE = 'create'
}

export enum GitContentType {
  FILE = 'file',
  DIR = 'dir',
  SYMLINK = 'symlink',
  SUBMODULE = 'submodule'
}

export enum GitBranchType {
  ACTIVE = 'active',
  INACTIVE = 'inactive',
  YOURS = 'yours',
  ALL = 'all'
}

export enum GitRefType {
  BRANCH = 'branch',
  TAG = 'tag'
}

export enum PrincipalUserType {
  USER = 'user',
  SERVICE = 'service'
}

export enum GitCommitAction {
  DELETE = 'DELETE',
  CREATE = 'CREATE',
  UPDATE = 'UPDATE',
  MOVE = 'MOVE'
}

export enum PullRequestState {
  OPEN = 'open',
  MERGED = 'merged',
  CLOSED = 'closed'
}

export const PullRequestFilterOption = {
  ...PullRequestState,
  // REJECTED: 'rejected',
  DRAFT: 'draft',
  YOURS: 'yours',
  ALL: 'all'
}

export const CodeIcon = {
  Logo: 'code' as IconName,
  PullRequest: 'git-pull' as IconName,
  Merged: 'code-merged' as IconName,
  Draft: 'code-draft' as IconName,
  PullRequestRejected: 'main-close' as IconName,
  Add: 'plus' as IconName,
  BranchSmall: 'code-branch-small' as IconName,
  Branch: 'code-branch' as IconName,
  Tag: 'main-tags' as IconName,
  Clone: 'code-clone' as IconName,
  Close: 'code-close' as IconName,
  CommitLight: 'code-commit-light' as IconName,
  CommitSmall: 'code-commit-small' as IconName,
  Commit: 'code-commit' as IconName,
  Copy: 'code-copy' as IconName,
  Delete: 'code-delete' as IconName,
  Edit: 'code-edit' as IconName,
  FileLight: 'code-file-light' as IconName,
  File: 'code-file' as IconName,
  Folder: 'code-folder' as IconName,
  History: 'code-history' as IconName,
  Info: 'code-info' as IconName,
  More: 'code-more' as IconName,
  Repo: 'code-repo' as IconName,
  Settings: 'code-settings' as IconName,
  Webhook: 'code-webhook' as IconName,
  InputSpinner: 'steps-spinne' as IconName,
  InputSearch: 'search' as IconName,
  Chat: 'code-chat' as IconName,
  Checks: 'main-tick' as IconName,
  ChecksSuccess: 'success-tick' as IconName
}

export enum Organization {
  GITHUB = 'Github',
  GITLAB = 'Gitlab'
}

export const normalizeGitRef = (gitRef: string | undefined) => {
  if (isRefATag(gitRef)) {
    return gitRef
  } else if (isRefABranch(gitRef)) {
    return gitRef
  } else if (gitRef === '') {
    return ''
  } else if (gitRef && isGitRev(gitRef)) {
    return gitRef
  } else {
    return `refs/heads/${gitRef}`
  }
}

export const REFS_TAGS_PREFIX = 'refs/tags/'
export const REFS_BRANCH_PREFIX = 'refs/heads/'

export function formatTriggers(triggers: EnumWebhookTrigger[]) {
  return triggers.map(trigger => {
    return trigger
      .split('_')
      .map(word => word.charAt(0).toUpperCase() + word.slice(1))
      .join(' ')
  })
}

// eslint-disable-next-line no-control-regex
const BAD_GIT_REF_REGREX = /(^|[/.])([/.]|$)|^@$|@{|[\x00-\x20\x7f~^:?*[\\]|\.lock(\/|$)/
const BAD_GIT_BRANCH_REGREX = /^(-|HEAD$)/

function isGitRefValid(name: string, onelevel: boolean): boolean {
  return !BAD_GIT_REF_REGREX.test(name) && (!!onelevel || name.includes('/'))
}

export function isGitBranchNameValid(name: string): boolean {
  return isGitRefValid(name, true) && !BAD_GIT_BRANCH_REGREX.test(name)
}

export const isDir = (content: Nullable<OpenapiGetContentOutput>): boolean => content?.type === GitContentType.DIR
export const isFile = (content: Nullable<OpenapiGetContentOutput>): boolean => content?.type === GitContentType.FILE
export const isSymlink = (content: Nullable<OpenapiGetContentOutput>): boolean =>
  content?.type === GitContentType.SYMLINK
export const isSubmodule = (content: Nullable<OpenapiGetContentOutput>): boolean =>
  content?.type === GitContentType.SUBMODULE

export const findReadmeInfo = (content: Nullable<OpenapiGetContentOutput>): OpenapiContentInfo | undefined =>
  (content?.content as OpenapiDirContent)?.entries?.find(
    entry => entry.type === GitContentType.FILE && /^readme(.md)?$/.test(entry?.name?.toLowerCase() || '')
  )

export const findMarkdownInfo = (content: Nullable<OpenapiGetContentOutput>): OpenapiContentInfo | undefined =>
  content?.type === GitContentType.FILE && /.md$/.test(content?.name?.toLowerCase() || '') ? content : undefined

export const isRefATag = (gitRef: string | undefined) => gitRef?.includes(REFS_TAGS_PREFIX) || false
export const isRefABranch = (gitRef: string | undefined) => gitRef?.includes(REFS_BRANCH_PREFIX) || false

/**
 * Make a diff refs string to use in URL.
 * @param targetGitRef target git ref (base ref).
 * @param sourceGitRef source git ref (compare ref).
 * @returns A concatenation string of `targetGitRef...sourceGitRef`.
 */
export const makeDiffRefs = (targetGitRef: string, sourceGitRef: string) => `${targetGitRef}...${sourceGitRef}`

/**
 * Split a diff refs string into targetRef and sourceRef.
 * @param diffRefs diff refs string.
 * @returns An object of { targetGitRef, sourceGitRef }
 */
export const diffRefsToRefs = (diffRefs: string) => {
  const parts = diffRefs.split('...')

  return {
    targetGitRef: parts[0] || '',
    sourceGitRef: parts[1] || ''
  }
}

export const decodeGitContent = (content = '') => {
  try {
    // Decode base64 content for text file
    return decodeURIComponent(escape(window.atob(content)))
  } catch (_exception) {
    try {
      // Return original base64 content for binary file
      return content
    } catch (exception) {
      console.error(exception) // eslint-disable-line no-console
    }
  }
  return ''
}

export const parseUrl = (url: string) => {
  const pattern = /^(https?:\/\/(?:www\.)?(github|gitlab)\.com\/([^/]+\/[^/]+))/
  const match = url.match(pattern)

  if (match) {
    const provider = match[2]
    const fullRepo = match[3]
    const repoName = match[3].split('/')[1].replace('.git', '')
    return { provider, fullRepo, repoName }
  } else {
    return null
  }
}

// Check if gitRef is a git commit hash (https://github.com/diegohaz/is-git-rev, MIT © Diego Haz)
export const isGitRev = (gitRef = ''): boolean => /^[0-9a-f]{7,40}$/i.test(gitRef)
