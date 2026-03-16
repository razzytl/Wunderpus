import { Sidebar } from './components/layout/Sidebar';
import { ChatPane } from './components/layout/ChatPane';
import { TelemetryPane } from './components/layout/TelemetryPane';

function App() {
  return (
    <div className="flex h-screen w-full bg-[#1E1E1E] overflow-hidden text-sm">
      {/* Left Sidebar (Narrow) */}
      <Sidebar />
      
      {/* Center Chat Pane (Flexible) */}
      <div className="flex-1 min-w-0">
        <ChatPane />
      </div>

      {/* Right Telemetry Pane (Fixed width) */}
      <div className="w-[450px] shrink-0 border-l border-[#2C2C2C]">
        <TelemetryPane />
      </div>
    </div>
  );
}

export default App;
