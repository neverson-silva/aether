import { Button, Field, InlineError, Input, Modal, Textarea } from '@aether/elisyum-ds'
import { Plus } from '@phosphor-icons/react'
import { useState } from 'react'
import { useCreateProject } from '../hooks/use-create-project'

export function CreateProjectDialog() {
  const createProject = useCreateProject()
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const reset = () => {
    setName('')
    setDescription('')
    createProject.reset()
  }
  const submit = async () => {
    if (!name.trim()) return
    await createProject.mutateAsync({
      name: name.trim(),
      description: description.trim() || undefined,
    })
    setOpen(false)
    reset()
  }
  return (
    <Modal
      open={open}
      onOpenChange={(nextOpen) => {
        setOpen(nextOpen)
        if (!nextOpen) reset()
      }}
      trigger={
        <span className="flex items-center gap-2">
          <Plus size={17} />
          Create project
        </span>
      }
      popupClassName="max-w-xl"
      title="Create project"
      description="Establish a new operational boundary for environments, services and delivery ownership."
      footer={
        <Button
          disabled={!name.trim() || createProject.isPending}
          onClick={() => void submit()}
        >
          {createProject.isPending ? 'Creating...' : 'Create project'}
        </Button>
      }
    >
      <div className="grid gap-5">
        <Field
          id="project-name"
          label="Project name"
          description="Use the name operators will see across the workspace."
          required
        >
          <Input
            autoFocus
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder="Payments platform"
          />
        </Field>
        <Field
          id="project-description"
          label="Description"
          description="Optional context for the team that owns this boundary."
        >
          <Textarea
            value={description}
            onChange={(event) => setDescription(event.target.value)}
            placeholder="Customer-facing payment workloads and their recovery boundary."
          />
        </Field>
        {createProject.isError ? (
          <InlineError>
            The project could not be created. Verify the name and try again.
          </InlineError>
        ) : null}
      </div>
    </Modal>
  )
}
