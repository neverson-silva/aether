import { Button, Field, Input, Modal, Textarea } from '@aether/elisyum-ds'
import { useState } from 'react'
import { apiPut } from '../api/client'
import { useCreateCompose } from '../hooks/use-create-compose'
import {
  type EnvironmentVariableDraft,
  EnvironmentVariableEditor,
} from './environment-variable-editor'

export function ComposeStackDialog({
  environmentId,
  onCreated,
  onClose,
  open,
  projectId,
}: {
  environmentId?: string
  onCreated: (serviceId: string) => void
  onClose: () => void
  open: boolean
  projectId: string
}) {
  const [name, setName] = useState('')
  const [compose, setCompose] = useState('')
  const [variables, setVariables] = useState<EnvironmentVariableDraft[]>([])
  const [createdServiceId, setCreatedServiceId] = useState('')
  const [error, setError] = useState('')
  const create = useCreateCompose()
  const variablesValid = variables.every(
    (variable) =>
      /^[A-Za-z_][A-Za-z0-9_]*$/.test(variable.name.trim()) &&
      variables.filter((candidate) => candidate.name.trim() === variable.name.trim())
        .length === 1,
  )
  const close = () => {
    setName('')
    setCompose('')
    setVariables([])
    setCreatedServiceId('')
    setError('')
    onClose()
  }
  const submit = async () => {
    if (!(name.trim() && compose.trim() && variablesValid)) return
    setError('')
    try {
      let serviceId = createdServiceId
      if (!serviceId) {
        const result = await create.mutateAsync({
          project_id: projectId,
          environment_id: environmentId,
          name: name.trim(),
          compose: compose.trim(),
        })
        serviceId = result.service_id || result.id
        setCreatedServiceId(serviceId)
      }
      for (const variable of variables) {
        await apiPut(`/api/v1/services/${serviceId}/environment`, {
          name: variable.name.trim(),
          value: variable.value,
          secret: variable.secret,
        })
      }
      close()
      onCreated(serviceId)
    } catch (reason) {
      setError(
        reason instanceof Error
          ? reason.message
          : 'The compose stack could not be created. Verify the definition and try again.',
      )
    }
  }
  return (
    <Modal
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) close()
      }}
      trigger={<span>Open compose creation</span>}
      triggerClassName="sr-only"
      title="Create compose stack"
      description="Deploy a multi-container Docker Compose stack inside the selected project boundary."
      popupClassName="max-w-3xl"
      footer={
        <Button
          disabled={
            !(name.trim() && compose.trim() && variablesValid) || create.isPending
          }
          onClick={() => void submit()}
        >
          {create.isPending
            ? 'Creating...'
            : createdServiceId
              ? 'Retry variable save'
              : 'Create compose stack'}
        </Button>
      }
    >
      <div className="grid gap-5">
        <Field
          id="compose-name"
          label="Stack name"
          required
          description="Use the name operators will see in the runtime inventory."
        >
          <Input
            autoFocus
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder="payments-stack"
          />
        </Field>
        <Field
          id="compose-definition"
          label="Compose definition"
          required
          description="Paste the Docker Compose YAML that defines the stack."
        >
          <Textarea
            className="min-h-64 font-technical text-code"
            value={compose}
            onChange={(event) => setCompose(event.target.value)}
            placeholder={'services:\n  web:\n    image: nginx:stable'}
          />
        </Field>
        <EnvironmentVariableEditor
          variables={variables}
          onChange={setVariables}
        />
        {!variablesValid ? (
          <p
            className="text-supporting text-danger"
            role="alert"
          >
            Fix invalid or duplicate environment variable names to continue.
          </p>
        ) : null}
        {error || create.isError ? (
          <p
            className="text-supporting text-danger"
            role="alert"
          >
            {error ||
              'The compose stack could not be created. Verify the definition and try again.'}
          </p>
        ) : null}
      </div>
    </Modal>
  )
}
