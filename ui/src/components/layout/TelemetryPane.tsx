import { Activity, Wrench, Terminal, CheckCircle2, XCircle, Clock } from 'lucide-react';
import { useWebSocket } from '../../contexts/WebSocketContext';
import { ToolExecution } from '../../types/websocket';

export function TelemetryPane() {
  const { systemLogs, messages } = useWebSocket();

  // Extract all tool executions from messages
  const toolExecs = messages
    .flatMap(m => m.toolExecutions || [])
    .reverse(); // Newest first

  return (
    <div className="flex flex-col h-screen bg-[#1A1A1A]">
      <div className="h-14 border-b border-[#2C2C2C] flex items-center px-6 shrink-0">
        <h1 className="text-gray-200 font-semibold flex items-center gap-2">
          <Activity size={18} className="text-green-500" />
          Telemetry & Tools
        </h1>
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-6">
        {/* Active/Recent Tools Section */}
        <div>
           <h2 className="text-gray-400 text-xs font-bold uppercase tracking-wider mb-3 flex items-center gap-2">
            <Wrench size={14} /> Tool Executions
          </h2>
          <div className="space-y-3">
            {toolExecs.length === 0 ? (
              <div className="text-sm text-gray-500 text-center py-4 bg-[#222] rounded border border-[#2C2C2C] border-dashed">
                No tools executed yet
              </div>
            ) : (
              toolExecs.map((te, idx) => (
                <ToolCard key={te.requestId + idx} execution={te} />
              ))
            )}
          </div>
        </div>

        {/* System Logs Section */}
        <div>
          <h2 className="text-gray-400 text-xs font-bold uppercase tracking-wider mb-3 flex items-center gap-2 mt-8">
            <Terminal size={14} /> System Logs
          </h2>
          <div className="bg-[#111] rounded-lg border border-[#2C2C2C] p-3 font-mono text-xs max-h-[400px] overflow-y-auto">
            {systemLogs.length === 0 ? (
              <div className="text-gray-500 italic">No logs available.</div>
            ) : (
              systemLogs.map((log, i) => (
                <div key={i} className={`mb-1 ${log.level === 'error' ? 'text-red-400' : log.level === 'warn' ? 'text-yellow-400' : 'text-gray-400'}`}>
                  <span className="text-gray-600">[{new Date().toLocaleTimeString()}]</span> [{log.level.toUpperCase()}] {log.message}
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function ToolCard({ execution }: { execution: ToolExecution }) {
  const StatusIcon = 
    execution.status === 'success' ? <CheckCircle2 size={14} className="text-green-500" /> :
    execution.status === 'error' ? <XCircle size={14} className="text-red-500" /> :
    <Clock size={14} className="text-yellow-500 animate-pulse" />;

  return (
    <div className="bg-[#222] border border-[#2C2C2C] rounded-lg p-3 text-sm">
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2 font-mono text-gray-300">
          {StatusIcon}
          {execution.toolName}
        </div>
        {execution.duration && <div className="text-xs text-gray-500">{execution.duration}</div>}
      </div>
      
      {execution.arguments && (
        <div className="mt-2 bg-[#111] rounded p-2 text-xs font-mono text-gray-400 overflow-x-auto whitespace-pre">
          {JSON.stringify(execution.arguments, null, 2)}
        </div>
      )}

      {execution.status !== 'running' && execution.output && (
        <div className="mt-2">
           <div className="text-xs text-gray-500 mb-1">Result:</div>
           <div className="bg-[#111] rounded p-2 text-xs font-mono text-gray-400 overflow-x-auto whitespace-pre max-h-32">
             {execution.output}
           </div>
        </div>
      )}

      {execution.status === 'error' && execution.error && (
        <div className="mt-2">
           <div className="text-xs text-red-400 mb-1">Error:</div>
           <div className="bg-red-950/30 text-red-300 rounded p-2 text-xs font-mono overflow-x-auto whitespace-pre">
             {execution.error}
           </div>
        </div>
      )}
    </div>
  );
}
