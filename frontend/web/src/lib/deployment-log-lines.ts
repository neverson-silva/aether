import type { LogLine } from "@aether/design-system";

const timestampPattern = /^(\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})?)/;
const ansiPattern = /\u001b\[[0-?]*[ -/]*[@-~]/g;

export function toDeploymentLogLines(content: string, error?: string): LogLine[] {
  const messages = [
    ...(error ? [error] : []),
    ...content.split("\n").filter((line) => line.trim().length > 0),
  ];

  return messages.map((rawMessage, index) => {
    const plainMessage = rawMessage.replace(ansiPattern, "");
    const timestamp = plainMessage.match(timestampPattern)?.[1];
    const lowerMessage = plainMessage.toLowerCase();
    const severity = /\b(error|failed|fatal|panic|exception|traceback)\b/.test(lowerMessage)
      ? "error"
      : /\b(warn|warning)\b/.test(lowerMessage)
        ? "warning"
        : /\b(healthy|passed|ready|started|running|success|successful)\b/.test(lowerMessage)
          ? "success"
          : "info";

    return {
      id: `${index}-${plainMessage}`,
      timestamp,
      severity,
      message: timestamp && rawMessage.startsWith(timestamp) ? rawMessage.slice(timestamp.length).trimStart() : rawMessage,
    };
  });
}
