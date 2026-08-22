import type { ReactNode } from 'react'
import { FormActions } from '../form-actions/form-actions'
import { Progress } from '../progress/progress'
import { Tabs } from '../tabs/tabs'
export interface DeploymentComposerProps {
  source?: ReactNode
  environment?: ReactNode
  variables?: ReactNode
  review?: ReactNode
  progress?: number
  status?: 'draft' | 'deploying' | 'success' | 'error' | 'rollback'
  onDeploy?: () => void
  onRollback?: () => void
  dirty?: boolean
  loading?: boolean
}
export function DeploymentComposer({
  dirty,
  environment,
  loading,
  onDeploy,
  onRollback,
  progress = 0,
  review,
  source,
  status = 'draft',
  variables,
}: DeploymentComposerProps) {
  return (
    <section className="rounded-xl border border-border bg-surface-card p-5">
      <Tabs
        items={[
          {
            value: 'source',
            label: 'Source',
            content: source ?? (
              <p className="text-body-sm text-muted-foreground">
                Select a source branch or commit.
              </p>
            ),
          },
          {
            value: 'environment',
            label: 'Environment',
            content: environment ?? (
              <p className="text-body-sm text-muted-foreground">
                Choose a target environment.
              </p>
            ),
          },
          {
            value: 'variables',
            label: 'Variables',
            content: variables ?? (
              <p className="text-body-sm text-muted-foreground">
                Configure runtime variables.
              </p>
            ),
          },
          {
            value: 'review',
            label: 'Review',
            content: review ?? (
              <p className="text-body-sm text-muted-foreground">
                Review deployment changes.
              </p>
            ),
          },
        ]}
      />
      {status === 'deploying' ? (
        <div className="mt-6">
          <Progress value={progress} label="Deployment progress" />
        </div>
      ) : null}
      {status === 'error' ? (
        <p role="alert" className="mt-4 text-body-sm text-status-danger">
          Deployment failed. Review logs or rollback.
        </p>
      ) : null}
      <div className="mt-6">
        <FormActions
          dirty={dirty}
          loading={loading}
          onSave={onDeploy}
          saveLabel={status === 'deploying' ? 'Deploying...' : 'Deploy'}
          success={status === 'success' ? 'Deployment complete' : undefined}
          error={status === 'error' ? 'Deployment failed' : undefined}
        >
          {status === 'rollback' ? (
            <button
              type="button"
              onClick={onRollback}
              className="text-body-sm text-status-danger"
            >
              Rollback
            </button>
          ) : null}
        </FormActions>
      </div>
    </section>
  )
}
