import {
  AlertDialog,
  Badge,
  Button,
  Card,
  DropdownMenu,
  EmptyState,
  Field,
  Input,
  Marker,
  Modal,
  Select,
  Skeleton,
  Typography,
} from '@aether/elisyum-ds'
import {
  CaretDown,
  Fingerprint,
  TrashSimple,
  UserPlus,
  UsersThree,
} from '@phosphor-icons/react'
import { useState } from 'react'
import { useOrg } from '../components/OrgProvider'
import { useAddMember } from '../hooks/use-add-member'
import { useDeleteMember } from '../hooks/use-delete-member'
import { useMembers } from '../hooks/use-members'
import { useUpdateMember } from '../hooks/use-update-member'

const roles = ['owner', 'admin', 'developer', 'viewer']
const roleDescriptions: Record<string, string> = {
  owner: 'Full organization control',
  admin: 'Apps, members and platform operations',
  developer: 'Create, edit and deploy services',
  viewer: 'Read-only operational access',
}

export function EmptyWorkspace() {
  const { currentOrg } = useOrg()
  const members = useMembers()
  if (members.isLoading)
    return (
      <div className="grid gap-4">
        <Skeleton className="h-48 rounded-2xl" />
        <Skeleton className="h-64 rounded-2xl" />
      </div>
    )
  if (members.isError || !currentOrg)
    return (
      <EmptyState
        title="Workspace context unavailable"
        description="The control plane did not return the current organization context."
        action={
          <Button
            tone="neutral"
            onClick={() => {
              void members.refetch()
            }}
          >
            Retry request
          </Button>
        }
      />
    )
  return (
    <div className="grid gap-7">
      <header className="flex flex-wrap items-end justify-between gap-5">
        <div className="grid gap-3">
          <span className="text-label tracking-[0.16em] text-text-subtle">
            ACCESS / ORGANIZATION
          </span>
          <Typography
            as="h1"
            role="page-title"
          >
            People and access
          </Typography>
          <Typography
            className="max-w-2xl"
            role="supporting"
          >
            Identity ownership, role boundaries and the operators currently in scope.
          </Typography>
        </div>
        <MemberEditor />
      </header>
      <MembersSurface
        members={members.data ?? []}
        orgId={currentOrg.id}
      />
    </div>
  )
}

function MemberEditor() {
  const addMember = useAddMember()
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState('developer')
  const [error, setError] = useState('')
  const submit = async () => {
    if (!(name.trim() && email.trim() && password)) return
    setError('')
    try {
      await addMember.mutateAsync({
        name: name.trim(),
        email: email.trim(),
        password,
        role,
      })
      setOpen(false)
      setName('')
      setEmail('')
      setPassword('')
      setRole('developer')
    } catch (value) {
      setError(
        value instanceof Error ? value.message : 'The member could not be added.',
      )
    }
  }
  return (
    <Modal
      open={open}
      onOpenChange={setOpen}
      title="Add organization member"
      description="Create an operator identity with an explicit role in this workspace."
      trigger={
        <>
          <UserPlus size={17} />
          Add member
        </>
      }
      footer={
        <Button
          disabled={addMember.isPending || !name.trim() || !email.trim() || !password}
          onClick={() => void submit()}
        >
          {addMember.isPending ? 'Adding…' : 'Add member'}
        </Button>
      }
    >
      <div className="grid gap-4">
        <Field
          id="member-name"
          label="Name"
          required
        >
          <Input
            onChange={(event) => setName(event.target.value)}
            placeholder="Alex Morgan"
            value={name}
          />
        </Field>
        <Field
          id="member-email"
          label="Work email"
          required
        >
          <Input
            onChange={(event) => setEmail(event.target.value)}
            placeholder="alex@company.com"
            type="email"
            value={email}
          />
        </Field>
        <Field
          id="member-password"
          label="Temporary password"
          description="The member can change this after signing in."
          required
        >
          <Input
            onChange={(event) => setPassword(event.target.value)}
            type="password"
            value={password}
          />
        </Field>
        <Field
          id="member-role"
          label="Role"
          required
        >
          <Select
            onChange={(event) => setRole(event.target.value)}
            value={role}
          >
            {roles.map((item) => (
              <option
                key={item}
                value={item}
              >
                {item[0].toUpperCase() + item.slice(1)}
              </option>
            ))}
          </Select>
        </Field>
        {error ? (
          <p
            className="text-supporting text-danger"
            role="alert"
          >
            {error}
          </p>
        ) : null}
      </div>
    </Modal>
  )
}

function MembersSurface({
  members,
  orgId,
}: {
  members: Array<{ user_id: string; name: string; email: string; role: string }>
  orgId: string
}) {
  return (
    <section className="grid gap-4">
      <div className="flex items-center gap-3 text-label tracking-[0.12em] text-text-subtle">
        <Marker tone="accent" />
        <UsersThree size={18} />
        {members.length} IDENTITIES IN SCOPE
      </div>
      {members.length ? (
        <Card className="grid gap-2 rounded-2xl border-border-subtle bg-surface-1 p-3">
          {members.map((member) => (
            <MemberRow
              key={member.user_id}
              member={member}
              orgId={orgId}
            />
          ))}
        </Card>
      ) : (
        <EmptyState
          title="No operators in scope"
          description="Add a member to establish the first access boundary for this organization."
          icon={<UsersThree size={28} />}
          action={<MemberEditor />}
        />
      )}
    </section>
  )
}

function MemberRow({
  member,
  orgId,
}: {
  member: { user_id: string; name: string; email: string; role: string }
  orgId: string
}) {
  const update = useUpdateMember()
  const remove = useDeleteMember()
  const [error, setError] = useState('')
  const updateRole = (role: string) => {
    setError('')
    update.mutate(
      { userID: member.user_id, role },
      {
        onError: (value) =>
          setError(
            value instanceof Error ? value.message : 'The role could not be updated.',
          ),
      },
    )
  }
  const deleteMember = () =>
    remove.mutate(
      { orgId, userId: member.user_id },
      {
        onError: (value) =>
          setError(
            value instanceof Error ? value.message : 'The member could not be removed.',
          ),
      },
    )
  return (
    <div className="grid gap-4 rounded-xl border border-transparent bg-surface-2 p-4 transition-colors hover:border-border-default hover:bg-surface-3 sm:grid-cols-[auto_minmax(0,1fr)_minmax(11rem,0.8fr)_auto] sm:items-center">
      <span className="grid size-10 place-items-center rounded-lg bg-surface-1 text-text-tertiary">
        <Fingerprint size={19} />
      </span>
      <span className="grid min-w-0 gap-1">
        <span className="truncate text-supporting text-text-primary">
          {member.name || member.email}
        </span>
        <span className="truncate font-technical text-log text-text-subtle">
          {member.email}
        </span>
      </span>
      <div className="grid gap-2">
        <div className="flex items-center gap-2">
          <Badge
            tone={
              member.role === 'owner' || member.role === 'admin' ? 'accent' : 'neutral'
            }
          >
            {member.role}
          </Badge>
          <DropdownMenu
            className="min-w-64 !rounded-xl !border-border-subtle !bg-surface-1/90 p-2 backdrop-blur-xl"
            items={roles.map((role) => ({
              label: (
                <span className="flex items-center gap-2">
                  <span
                    className={`size-1.5 rounded-full ${role === member.role ? 'bg-action' : 'bg-transparent'}`}
                  />
                  {role[0].toUpperCase() + role.slice(1)}
                </span>
              ),
              onSelect: () => updateRole(role),
              disabled: update.isPending,
            }))}
            title="Organization role"
            trigger={
              <span
                aria-label={`Change role for ${member.name || member.email}`}
                className="flex min-w-0 items-center justify-between gap-3"
              >
                <span className="truncate text-supporting font-medium">
                  {member.role}
                </span>
                <CaretDown
                  aria-hidden="true"
                  className="shrink-0 text-text-tertiary"
                  size={16}
                  weight="bold"
                />
              </span>
            }
            triggerClassName="inline-flex min-h-10 min-w-44 cursor-pointer items-center justify-between rounded-xl px-2.5 text-text-primary transition-[background-color,color,transform] duration-[var(--ely-duration-fast)] ease-ely-out hover:bg-surface-2 active:bg-surface-3 motion-safe:active:scale-[var(--ely-motion-press-scale)] focus-visible:outline-2 focus-visible:outline-focus disabled:pointer-events-none disabled:opacity-60"
          />
        </div>
        <span className="text-supporting text-text-tertiary">
          {roleDescriptions[member.role] || 'Organization access'}
        </span>
      </div>
      <AlertDialog
        triggerClassName="grid size-9 cursor-pointer place-items-center rounded-lg text-text-tertiary transition-colors hover:bg-danger-soft hover:text-danger focus-visible:outline-2 focus-visible:outline-focus"
        trigger={
          <span>
            <TrashSimple
              aria-hidden="true"
              size={17}
            />
            <span className="sr-only">Remove {member.name || member.email}</span>
          </span>
        }
        title="Remove organization member"
        description={`This revokes ${member.email}'s access to the current organization.`}
        confirmLabel={remove.isPending ? 'Removing…' : 'Remove member'}
        onConfirm={deleteMember}
      />
      {error ? (
        <p
          className="text-supporting text-danger sm:col-start-2 sm:col-span-3"
          role="alert"
        >
          {error}
        </p>
      ) : null}
    </div>
  )
}
