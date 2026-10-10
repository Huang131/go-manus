import { createSSEStream, get, parseSSEStream, post, StreamEndError } from "./fetch";
import type {
  Run,
  RunHistoryPage,
  SSEEventData,
  SSEEventHandler,
} from "./types";

export type RunStreamErrorHandler = (error: Error) => void;

function idempotencyKey(): string {
  return crypto.randomUUID();
}

/** Run API 是执行事实的唯一前端入口；Session 只用于归属和展示。 */
export const runApi = {
  create: (
    sessionId: string,
    message: string,
    attachments: string[] = [],
    modelId?: string,
    key = idempotencyKey(),
  ): Promise<Run> => post<Run>(
    `/sessions/${sessionId}/runs`,
    { message, attachments, ...(modelId ? { model_id: modelId } : {}) },
    { headers: { "Idempotency-Key": key } },
  ),

  get: (runId: string): Promise<Run> => get<Run>(`/runs/${runId}`),

  listBySession: (sessionId: string, limit = 50, offset = 0): Promise<RunHistoryPage> =>
    get<RunHistoryPage>(`/sessions/${sessionId}/runs`, { limit, offset }),

  submitInput: (
    runId: string,
    message: string,
    replyToMessageId: string,
    attachments: string[] = [],
    modelId?: string,
    key = idempotencyKey(),
  ): Promise<Run> => post<Run>(
    `/runs/${runId}/input`,
    { idempotency_key: key, reply_to_message_id: replyToMessageId, message, attachments, ...(modelId ? { model_id: modelId } : {}) },
    { headers: { "Idempotency-Key": key } },
  ),

  cancel: (runId: string): Promise<Run> => post<Run>(`/runs/${runId}/cancel`, {}),

  streamEvents: (
    runId: string,
    lastEventId: string | undefined,
    onEvent: SSEEventHandler,
    onError?: RunStreamErrorHandler,
  ): (() => void) => {
    const controller = new AbortController();
    const start = async () => {
      let parseError = false;
      try {
        const stream = await createSSEStream(`/runs/${runId}/events`, undefined, {
          method: "GET",
          signal: controller.signal,
          timeout: 5 * 60 * 1000,
          headers: lastEventId ? { "Last-Event-ID": lastEventId } : {},
        });
        await parseSSEStream(
          stream,
          (event) => {
            if (controller.signal.aborted) return;
            onEvent({
              type: event.type as SSEEventData["type"],
              data: typeof event.data === "string" ? JSON.parse(event.data) : event.data,
              streamId: event.lastEventId || undefined,
            } as SSEEventData);
          },
          (error) => {
            if (!controller.signal.aborted && onError) {
              parseError = true;
              onError(error);
            }
          },
        );
        if (!controller.signal.aborted && onError && !parseError) onError(new StreamEndError());
      } catch (error) {
        if (!controller.signal.aborted && onError) {
          onError(error instanceof Error ? error : new Error("Run 事件流连接失败"));
        }
      }
    };
    void start();
    return () => controller.abort();
  },
};
