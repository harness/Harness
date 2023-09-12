import React, { useMemo, useState } from 'react'
import { useHistory } from 'react-router'
import { useMutate } from 'restful-react'
import * as yup from 'yup'
import { capitalize } from 'lodash'
import { FontVariation } from '@harnessio/design-system'
import {
  Button,
  ButtonVariation,
  Container,
  Dialog,
  Formik,
  FormikForm,
  Layout,
  Text,
  useToaster
} from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import { useModalHook } from 'hooks/useModalHook'
import type { CreateExecutionQueryParams, TypesExecution, TypesRepository } from 'services/code'
import { getErrorMessage } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { BranchTagSelect } from 'components/BranchTagSelect/BranchTagSelect'

import css from './RunPipelineModal.module.scss'

interface FormData {
  branch: string
}

const useRunPipelineModal = () => {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const { showSuccess, showError, clear: clearToaster } = useToaster()
  const history = useHistory()
  const [repo, setRepo] = useState<TypesRepository>()
  const [pipeline, setPipeline] = useState<string>('')
  const repoPath = useMemo(() => repo?.path || '', [repo])

  const { mutate: startExecution } = useMutate<TypesExecution>({
    verb: 'POST',
    path: `/api/v1/repos/${repoPath}/+/pipelines/${pipeline}/executions`
  })

  const runPipeline = (formData: FormData): void => {
    const { branch } = formData
    try {
      startExecution(
        {},
        {
          pathParams: { path: `/api/v1/repos/${repoPath}/+/pipelines/${pipeline}/executions` },
          queryParams: { branch } as CreateExecutionQueryParams
        }
      )
        .then(response => {
          clearToaster()
          showSuccess(getString('pipelines.executionStarted'))
          if (response?.number && !isNaN(response.number)) {
            history.push(routes.toCODEExecution({ repoPath, pipeline, execution: response.number.toString() }))
          }
          hideModal()
        })
        .catch(error => {
          showError(getErrorMessage(error), 0, 'pipelines.executionCouldNotStart')
        })
    } catch (exception) {
      showError(getErrorMessage(exception), 0, 'pipelines.executionCouldNotStart')
    }
  }

  const [openModal, hideModal] = useModalHook(() => {
    const onClose = () => {
      hideModal()
    }
    return (
      <Dialog isOpen enforceFocus={false} onClose={onClose} title={getString('pipelines.run')}>
        <Formik
          formName="run-pipeline-form"
          initialValues={{ branch: repo?.default_branch || '' }}
          validationSchema={yup.object().shape({
            branch: yup
              .string()
              .trim()
              .required(`${getString('branch')} ${getString('isRequired')}`)
          })}
          onSubmit={runPipeline}
          enableReinitialize>
          {formik => {
            return (
              <FormikForm>
                <Layout.Vertical spacing="medium">
                  <Layout.Vertical spacing="xsmall" padding={{ bottom: 'medium' }}>
                    <Text font={{ variation: FontVariation.BODY }}>{capitalize(getString('branch'))}</Text>
                    <Container className={css.branchSelect}>
                      <BranchTagSelect
                        gitRef={formik?.values?.branch || repo?.default_branch || ''}
                        onSelect={(ref: string) => {
                          formik?.setFieldValue('branch', ref)
                        }}
                        repoMetadata={repo || {}}
                        disableBranchCreation
                        disableViewAllBranches
                        forBranchesOnly
                      />
                    </Container>
                  </Layout.Vertical>
                  <Layout.Horizontal spacing="medium">
                    <Button variation={ButtonVariation.PRIMARY} type="submit" text={getString('pipelines.run')} />
                    <Button variation={ButtonVariation.SECONDARY} text={getString('cancel')} onClick={onClose} />
                  </Layout.Horizontal>
                </Layout.Vertical>
              </FormikForm>
            )
          }}
        </Formik>
      </Dialog>
    )
  }, [repo?.default_branch, pipeline])

  return {
    openModal: ({ repoMetadata, pipeline }: { repoMetadata: TypesRepository; pipeline: string }) => {
      setRepo(repoMetadata)
      setPipeline(pipeline)
      openModal()
    },
    hideModal
  }
}

export default useRunPipelineModal