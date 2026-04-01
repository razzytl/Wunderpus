import {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
  useCallback,
  type ReactNode,
} from 'react';
import type {
  WSMessage,
  ChatMessage,
  ChatTokenPayload,
  ChatCompletePayload,
  ToolExecutionStartPayload,
  ToolExecutionResultPayload,
  SystemLogPayload,
  ErrorPayload,
  UserMessagePayload,
  ToolExecution,
} from '../types/websocket';
import { MSG_TYPE } from '../types/websocket';

// --- Context Types ---

interface WebSocketContextValue {
  connected: boolean;
  messages: ChatMessage[];
  systemLogs: SystemLogPayload[];
  sendMessage: (content: string) => void;
  clearMessages: () => void;
}

const WebSocketContext = createContext<WebSocketContextValue | null>(null);

// --- Provider ---

interface WebSocketProviderProps {
  children: ReactNode;
  url?: string;
}

let messageCounter = 0;
function nextId(): string {
  return `msg_${Date.now()}_${++messageCounter}`;
}

export function WebSocketProvider({ children, url }: WebSocketProviderProps) {
  const [connected, setConnected] = useState(false);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [systemLogs, setSystemLogs] = useState<SystemLogPayload[]>([]);

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const reconnectAttemptRef = useRef(0);

  // Resolve WebSocket URL
  const wsUrl = url ?? `ws://${window.location.host}/ws`;

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) return;

    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      setConnected(true);
      reconnectAttemptRef.current = 0;
      console.log('[WS] Connected to', wsUrl);
    };

    ws.onclose = () => {
      setConnected(false);
      console.log('[WS] Disconnected');

      // Auto-reconnect with exponential backoff
      const delay = Math.min(1000 * 2 ** reconnectAttemptRef.current, 30000);
      reconnectAttemptRef.current++;
      console.log(`[WS] Reconnecting in ${delay}ms...`);
      reconnectTimeoutRef.current = setTimeout(connect, delay);
    };

    ws.onerror = (event) => {
      console.error('[WS] Error:', event);
    };

    ws.onmessage = (event) => {
      try {
        const msg: WSMessage = JSON.parse(event.data);
        handleMessage(msg);
      } catch (err) {
        console.error('[WS] Failed to parse message:', err);
      }
    };
  }, [wsUrl]);

  // eslint-disable-next-line react-hooks/exhaustive-deps
  const handleMessage = useCallback((msg: WSMessage) => {
    switch (msg.type) {
      case MSG_TYPE.CHAT_TOKEN: {
        const payload = msg.payload as ChatTokenPayload;
        setMessages((prev) => {
          // Find the last assistant message that's still streaming
          const lastIdx = prev.length - 1;
          const last = prev[lastIdx];

          if (last && last.role === 'assistant' && last.isStreaming) {
            // Append token to existing streaming message
            const updated = [...prev];
            updated[lastIdx] = {
              ...last,
              content: last.content + payload.token,
              isStreaming: !payload.done,
            };
            return updated;
          }

          // Start a new assistant message
          return [
            ...prev,
            {
              id: nextId(),
              role: 'assistant',
              content: payload.token,
              timestamp: new Date(msg.timestamp),
              isStreaming: !payload.done,
            },
          ];
        });
        break;
      }

      case MSG_TYPE.CHAT_COMPLETE: {
        const payload = msg.payload as ChatCompletePayload;
        setMessages((prev) => {
          const lastIdx = prev.length - 1;
          const last = prev[lastIdx];

          if (last && last.role === 'assistant') {
            const updated = [...prev];
            updated[lastIdx] = {
              ...last,
              content: payload.content,
              isStreaming: false,
            };
            return updated;
          }

          return [
            ...prev,
            {
              id: nextId(),
              role: 'assistant',
              content: payload.content,
              timestamp: new Date(msg.timestamp),
              isStreaming: false,
            },
          ];
        });
        break;
      }

      case MSG_TYPE.TOOL_EXECUTION_START: {
        const payload = msg.payload as ToolExecutionStartPayload;
        const toolExec: ToolExecution = {
          requestId: payload.request_id,
          toolName: payload.tool_name,
          arguments: payload.arguments,
          status: 'running',
        };

        setMessages((prev) => {
          const lastIdx = prev.length - 1;
          const last = prev[lastIdx];

          if (last && last.role === 'assistant') {
            const updated = [...prev];
            updated[lastIdx] = {
              ...last,
              toolExecutions: [...(last.toolExecutions ?? []), toolExec],
            };
            return updated;
          }

          return [
            ...prev,
            {
              id: nextId(),
              role: 'assistant',
              content: '',
              timestamp: new Date(msg.timestamp),
              isStreaming: true,
              toolExecutions: [toolExec],
            },
          ];
        });
        break;
      }

      case MSG_TYPE.TOOL_EXECUTION_RESULT: {
        const payload = msg.payload as ToolExecutionResultPayload;

        setMessages((prev) => {
          return prev.map((m) => {
            if (!m.toolExecutions) return m;

            const updatedExecs = m.toolExecutions.map((te) => {
              if (te.requestId === payload.request_id) {
                return {
                  ...te,
                  status: (payload.success ? 'success' : 'error') as ToolExecution['status'],
                  output: payload.output,
                  error: payload.error,
                  duration: payload.duration,
                };
              }
              return te;
            });

            return { ...m, toolExecutions: updatedExecs };
          });
        });
        break;
      }

      case MSG_TYPE.SYSTEM_LOG: {
        const payload = msg.payload as SystemLogPayload;
        setSystemLogs((prev) => [...prev.slice(-99), payload]); // Keep last 100
        break;
      }

      case MSG_TYPE.ERROR: {
        const payload = msg.payload as ErrorPayload;
        setMessages((prev) => [
          ...prev,
          {
            id: nextId(),
            role: 'system',
            content: `⚠️ Error: ${payload.message}`,
            timestamp: new Date(msg.timestamp),
          },
        ]);
        break;
      }
    }
  }, []);

  // Connect on mount
  useEffect(() => {
    connect();
    return () => {
      clearTimeout(reconnectTimeoutRef.current);
      wsRef.current?.close();
    };
  }, [connect]);

  // Send a user message
  const sendMessage = useCallback(
    (content: string) => {
      if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
        console.error('[WS] Not connected');
        return;
      }

      // Add user message to local state immediately
      setMessages((prev) => [
        ...prev,
        {
          id: nextId(),
          role: 'user',
          content,
          timestamp: new Date(),
        },
      ]);

      // Send over WebSocket
      const envelope: WSMessage<UserMessagePayload> = {
        type: MSG_TYPE.USER_MESSAGE,
        timestamp: new Date().toISOString(),
        payload: { content },
      };

      wsRef.current.send(JSON.stringify(envelope));
    },
    []
  );

  const clearMessages = useCallback(() => {
    setMessages([]);
  }, []);

  return (
    <WebSocketContext.Provider
      value={{ connected, messages, systemLogs, sendMessage, clearMessages }}
    >
      {children}
    </WebSocketContext.Provider>
  );
}

// --- Hook ---

export function useWebSocket(): WebSocketContextValue {
  const ctx = useContext(WebSocketContext);
  if (!ctx) {
    throw new Error('useWebSocket must be used within a WebSocketProvider');
  }
  return ctx;
}
