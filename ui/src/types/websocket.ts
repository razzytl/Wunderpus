// WebSocket message types — mirrors web/types.go on the Go backend

// Message type constants
export const MSG_TYPE = {
  // Client → Server
  USER_MESSAGE: 'user_message',

  // Server → Client
  CHAT_TOKEN: 'chat_token',
  CHAT_COMPLETE: 'chat_complete',
  TOOL_EXECUTION_START: 'tool_execution_start',
  TOOL_EXECUTION_RESULT: 'tool_execution_result',
  SYSTEM_LOG: 'system_log',
  ERROR: 'error',
} as const;

export type MessageType = (typeof MSG_TYPE)[keyof typeof MSG_TYPE];

// Envelope for all WebSocket messages
export interface WSMessage<T = unknown> {
  type: MessageType;
  timestamp: string;
  session_id?: string;
  payload: T;
}

// --- Client → Server ---

export interface UserMessagePayload {
  content: string;
  session_id?: string;
}

// --- Server → Client ---

export interface ChatTokenPayload {
  token: string;
  done: boolean;
}

export interface ChatCompletePayload {
  content: string;
  token_count?: number;
}

export interface ToolExecutionStartPayload {
  tool_name: string;
  arguments?: Record<string, unknown>;
  request_id: string;
}

export interface ToolExecutionResultPayload {
  tool_name: string;
  request_id: string;
  success: boolean;
  output?: string;
  error?: string;
  duration?: string;
}

export interface SystemLogPayload {
  level: 'info' | 'warn' | 'error';
  message: string;
}

export interface ErrorPayload {
  code?: string;
  message: string;
}

// --- UI State Types ---

export type ChatMessageRole = 'user' | 'assistant' | 'system';

export interface ChatMessage {
  id: string;
  role: ChatMessageRole;
  content: string;
  timestamp: Date;
  isStreaming?: boolean;
  toolExecutions?: ToolExecution[];
}

export interface ToolExecution {
  requestId: string;
  toolName: string;
  arguments?: Record<string, unknown>;
  status: 'running' | 'success' | 'error';
  output?: string;
  error?: string;
  duration?: string;
}
