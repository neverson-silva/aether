import { Check, Clock, X } from "@phosphor-icons/react";
import type { ReactNode } from "react";
export interface ApprovalPerson {
  id: string;
  name: string;
  role?: string;
  status: "pending" | "approved" | "rejected" | "expired";
}
export interface ApprovalFlowProps {
  requester: ReactNode;
  approvers: ApprovalPerson[];
  policy?: ReactNode;
  comment?: string;
  onCommentChange?: (comment: string) => void;
  onApprove?: () => void;
  onReject?: () => void;
  status?: "pending" | "approved" | "rejected" | "expired";
}
export function ApprovalFlow({
  approvers,
  comment,
  onApprove,
  onCommentChange,
  onReject,
  policy,
  requester,
  status = "pending",
}: ApprovalFlowProps) {
  const icons = {
    pending: <Clock size={16} />,
    approved: <Check size={16} />,
    rejected: <X size={16} />,
    expired: <Clock size={16} />,
  };
  return (
    <section className="space-y-5 rounded-2xl border border-border bg-surface-card p-5 shadow-sm sm:p-6">
      <div>
        <div className="text-label-caps text-muted-foreground">
          Requested by
        </div>
        <div className="mt-1 text-body-sm text-foreground">{requester}</div>
      </div>
      {policy ? (
        <div className="rounded-xl border border-border/70 bg-surface-container-low p-4 text-body-sm text-muted-foreground">
          {policy}
        </div>
      ) : null}
      <div>
        <div className="mb-2 text-label-caps text-muted-foreground">
          Approvers
        </div>
        <div className="space-y-2">
          {approvers.map((approver) => (
            <div
              key={approver.id}
              className="flex items-center gap-3 rounded-xl border border-border bg-surface-container-low/40 p-3 transition-colors hover:bg-surface-container"
            >
              <span
                className={`inline-flex size-7 items-center justify-center rounded-full ${approver.status === "approved" ? "bg-status-success text-status-success-foreground" : approver.status === "rejected" || approver.status === "expired" ? "bg-status-danger text-status-danger-foreground" : "bg-surface-container text-muted-foreground"}`}
              >
                {icons[approver.status]}
              </span>
              <span className="min-w-0 flex-1">
                <span className="block text-body-sm font-semibold">
                  {approver.name}
                </span>
                {approver.role ? (
                  <span className="block text-body-sm text-muted-foreground">
                    {approver.role}
                  </span>
                ) : null}
              </span>
              <span className="text-label-caps text-muted-foreground">
                {approver.status}
              </span>
            </div>
          ))}
        </div>
      </div>
      {status === "pending" ? (
        <>
          <textarea
            value={comment ?? ""}
            onChange={(event) => onCommentChange?.(event.target.value)}
            placeholder="Add a comment (optional)"
            className="min-h-24 w-full rounded-xl border border-border bg-surface-control p-3 text-body-sm outline-none transition-[background-color,border-color,box-shadow] duration-200 hover:bg-surface-container-highest/40 focus:border-primary focus:bg-surface-card focus:ring-2 focus:ring-primary/20"
          />
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={onReject}
              className="rounded-xl border border-status-danger/40 px-3 py-2 text-body-sm text-status-danger outline-none transition-[background-color,transform] duration-150 hover:bg-status-danger/10 focus-visible:ring-2 focus-visible:ring-status-danger/40 active:scale-[0.985]"
            >
              Reject
            </button>
            <button
              type="button"
              onClick={onApprove}
              className="rounded-xl bg-button-success px-3 py-2 text-body-sm text-button-success-foreground outline-none transition-[filter,transform] duration-150 hover:brightness-110 focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985]"
            >
              Approve
            </button>
          </div>
        </>
      ) : (
        <div className="text-body-sm font-semibold text-muted-foreground">
          Request {status}.
        </div>
      )}
    </section>
  );
}
