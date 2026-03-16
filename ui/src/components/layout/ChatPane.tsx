import { useState, useRef, useEffect } from 'react';
import type { KeyboardEvent } from 'react';
import { Send, Bot, User, CheckCircle2, XCircle, Loader2 } from 'lucide-react';
import ReactMarkdown from 'react-markdown';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { useWebSocket } from '../../contexts/WebSocketContext';
import type { ChatMessage, ToolExecution } from '../../types/websocket';
import { Accordion, AccordionItem, AccordionTrigger, AccordionContent } from '../ui/accordion';

export function ChatPane() {
  const { messages, sendMessage, connected } = useWebSocket();
  const [input, setInput] = useState('');
  const bottomRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSubmit = (e?: React.FormEvent) => {
    e?.preventDefault();
    if (!input.trim() || !connected) return;
    sendMessage(input);
    setInput('');
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto'; // Reset height
    }
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  };

  const handleInput = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setInput(e.target.value);
    e.target.style.height = 'auto';
    e.target.style.height = `${Math.min(e.target.scrollHeight, 200)}px`;
  };

  return (
    <div className="flex flex-col h-screen bg-[#1E1E1E] border-r border-[#2C2C2C]">
      {/* Header */}
      <div className="h-14 border-b border-[#2C2C2C] flex items-center justify-between px-6 shrink-0 z-10 bg-[#1E1E1E]">
        <h1 className="text-gray-200 font-semibold">Agent Chat</h1>
        {!connected && (
          <div className="flex items-center gap-2 text-yellow-500 text-sm bg-yellow-500/10 px-3 py-1 rounded-full border border-yellow-500/20">
            <Loader2 size={14} className="animate-spin" />
            <span>Disconnected - Reconnecting...</span>
          </div>
        )}
      </div>

      {/* Message List */}
      <div className="flex-1 overflow-y-auto p-4 space-y-6">
        {messages.length === 0 ? (
          <div className="h-full flex items-center justify-center text-gray-500">
            Start a conversation with the agent...
          </div>
        ) : (
          messages.map((msg) => (
             <MessageBubble key={msg.id} message={msg} />
          ))
        )}
        <div ref={bottomRef} />
      </div>

      {/* Input Area */}
      <div className="p-4 border-t border-[#2C2C2C] bg-[#1E1E1E] shrink-0">
        <form onSubmit={handleSubmit} className="flex relative">
          <textarea
            ref={textareaRef}
            value={input}
            onChange={handleInput}
            onKeyDown={handleKeyDown}
            placeholder={connected ? "Send a message... (Shift+Enter for new line)" : "Connecting..."}
            disabled={!connected}
            rows={1}
            className="w-full bg-[#2C2C2C] border border-[#3C3C3C] rounded-lg px-4 py-3 pb-3 pr-12 text-gray-200 focus:outline-none focus:border-blue-500 disabled:opacity-50 resize-none overflow-y-auto"
            style={{ maxHeight: '200px' }}
          />
          <button
            type="submit"
            disabled={!input.trim() || !connected}
            className="absolute right-2 bottom-2 p-1.5 bg-blue-600 text-white rounded-md hover:bg-blue-500 disabled:opacity-50 disabled:hover:bg-blue-600 transition-colors"
          >
            <Send size={18} />
          </button>
        </form>
      </div>
    </div>
  );
}

function MessageBubble({ message }: { message: ChatMessage }) {
  const isUser = message.role === 'user';
  
  return (
    <div className={`flex gap-4 ${isUser ? 'flex-row-reverse' : 'flex-row'}`}>
      <div className={`w-8 h-8 rounded-full flex items-center justify-center shrink-0 mt-1 ${isUser ? 'bg-blue-600' : message.role === 'system' ? 'bg-red-500' : 'bg-[#2C2C2C]'}`}>
         {isUser ? <User size={16} className="text-white" /> : <Bot size={16} className="text-gray-300" />}
      </div>
      <div className={`max-w-[80%] rounded-lg p-4 ${isUser ? 'bg-[#2C2C2C] text-gray-200' : 'bg-transparent text-gray-200'}`}>
        {!isUser && message.toolExecutions && message.toolExecutions.length > 0 && (
          <div className="mb-4 space-y-2">
            {message.toolExecutions.map(exec => (
              <ToolExecutionBlock key={exec.requestId} execution={exec} />
            ))}
          </div>
        )}

        {isUser ? (
          <div className="whitespace-pre-wrap">{message.content}</div>
        ) : (
          <div className="prose prose-invert max-w-none prose-pre:bg-transparent prose-pre:p-0">
            {message.content && (
              <ReactMarkdown
                components={{
                  code({ node, inline, className, children, ...props }: any) {
                    const match = /language-(\w+)/.exec(className || '');
                    return !inline && match ? (
                      <SyntaxHighlighter
                        {...props}
                        children={String(children).replace(/\n$/, '')}
                        style={vscDarkPlus as any}
                        language={match[1]}
                        PreTag="div"
                        className="rounded-md my-2"
                      />
                    ) : (
                      <code {...props} className="bg-[#3C3C3C] px-1.5 py-0.5 rounded text-sm text-blue-300 gap-1">
                        {children}
                      </code>
                    );
                  }
                }}
              >
                {message.content}
              </ReactMarkdown>
            )}
            {message.isStreaming && (
              <span className="inline-flex items-center gap-1 ml-2 text-blue-400">
                <span className="animate-pulse">●</span>
              </span>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function ToolExecutionBlock({ execution }: { execution: ToolExecution }) {
  const isRunning = execution.status === 'running';
  const isSuccess = execution.status === 'success';
  const isError = execution.status === 'error';

  return (
    <Accordion type="single" collapsible className="w-full">
      <AccordionItem value="item-1" className="border border-[#3C3C3C] rounded-md bg-[#222] px-2 overflow-hidden">
        <AccordionTrigger className="hover:no-underline py-2">
          <div className="flex items-center gap-2">
            {isRunning && <Loader2 size={16} className="text-blue-400 animate-spin" />}
            {isSuccess && <CheckCircle2 size={16} className="text-green-500" />}
            {isError && <XCircle size={16} className="text-red-500" />}
            
            <span className={`font-mono text-xs ${isRunning ? 'text-gray-300' : isSuccess ? 'text-green-400' : 'text-red-400'}`}>
              {isRunning && `Agent is using [${execution.toolName}]...`}
              {isSuccess && `Successfully used [${execution.toolName}]`}
              {isError && `Error using [${execution.toolName}]`}
            </span>
          </div>
        </AccordionTrigger>
        <AccordionContent className="pt-2 pb-2 border-t border-[#3C3C3C]">
          <div className="space-y-4">
            {/* Input Arguments */}
            <div>
              <div className="text-[10px] text-gray-500 uppercase tracking-wider mb-1 font-bold">Request JSON:</div>
              <div className="bg-[#111] p-2 rounded text-xs font-mono text-gray-400 overflow-x-auto whitespace-pre">
                {execution.arguments ? JSON.stringify(execution.arguments, null, 2) : '{}'}
              </div>
            </div>

            {/* Output Result */}
            {execution.output && (
              <div>
                <div className="text-[10px] text-gray-500 uppercase tracking-wider mb-1 font-bold">Response:</div>
                <div className="bg-[#111] p-2 rounded text-xs font-mono text-green-400/80 overflow-x-auto whitespace-pre max-h-64">
                  {execution.output}
                </div>
              </div>
            )}

            {/* Error Message */}
            {execution.error && (
              <div>
                <div className="text-[10px] text-red-500 uppercase tracking-wider mb-1 font-bold">Error Output:</div>
                <div className="bg-red-950/30 border border-red-900/50 p-2 rounded text-xs font-mono text-red-300 overflow-x-auto whitespace-pre">
                  {execution.error}
                </div>
              </div>
            )}
          </div>
        </AccordionContent>
      </AccordionItem>
    </Accordion>
  );
}
