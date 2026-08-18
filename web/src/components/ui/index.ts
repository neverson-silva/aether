/* Design System — ponto de entrada único.
   Cada componente vive em seu próprio arquivo e é reexportado aqui. */

export { cn, SpinnerMini } from "./cn";
export { Button, BTN_BASE, BTN_VARIANTS, BTN_SIZES } from "./button";
export type { ButtonVariant, ButtonSize } from "./button";
export { Input, Select, Field } from "./form";
export { Card, Skeleton, SkeletonList } from "./card";
export { StatusPill } from "./status";
export { DeploymentStatus, isDeploymentActive } from "./runtime-status";
export { Popover, Spinner, EmptyState, CodeBlock } from "./feedback";
export { Table } from "./table";
export { Modal, ConfirmDialog } from "./modal";
export { useToast, ToastProvider } from "./toast";
export type { ToastLevel } from "./toast";
export { CardMenu } from "./card-menu";
export { MetricCard, fmtBytes, fmtDate } from "./metrics";

export { AppButton } from "./button";
export type { AppButtonVariant, AppButtonSize } from "./button";
export { AppLink } from "./app-link";
export { AppCard } from "./app-card";
export { AppPage, AppPageHeader, AppPageTitle, AppDescription, AppSection, AppSectionHeader, AppToolbar, AppToolbarActions } from "./app-page";
export { AppBadge, AppStatusBadge } from "./app-badge";
export { AppStatCard, AppInfoRow } from "./app-stat";
export { AppEmptyState, AppErrorState, AppLoading, AppSkeleton, AppSkeletonCard } from "./app-states";
