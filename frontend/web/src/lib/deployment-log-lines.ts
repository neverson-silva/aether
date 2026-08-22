import type { LogLine } from "@aether/design-system";

const timestampPattern = /^(\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})?)/;

export function toDeploymentLogLines(content: string, error?: string): LogLine[] {
  const messages = [
    ...(error ? [error] : []),
    ...content.split("\n").filter((line) => line.trim().length > 0),
  ];

  return messages.map((rawMessage, index) => {
    const message = rawMessage.replace(/\u001b\[[0-9;]*[A-Za-z]/g, "");
    const timestamp = message.match(timestampPattern)?.[1];
    const lowerMessage = message.toLowerCase();
    const severity = /\b(error|failed|fatal|panic|exception|traceback)\b/.test(lowerMessage)
      ? "error"
      : /\b(warn|warning)\b/.test(lowerMessage)
        ? "warning"
        : /\b(healthy|passed|ready|started|running|success|successful)\b/.test(lowerMessage)
          ? "success"
          : "info";

    return {
      id: `${index}-${message}`,
      timestamp,
      severity,
      message: timestamp ? message.slice(timestamp.length).trimStart() : message,
    };
  });
}
