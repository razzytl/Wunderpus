import { Terminal, Settings, MessageSquare, Database, Activity } from 'lucide-react';
import { useWebSocket } from '../../contexts/WebSocketContext';

export function Sidebar() {
  const { connected } = useWebSocket();

  return (
    <div className="w-16 h-screen border-r border-[#2C2C2C] bg-[#1E1E1E] flex flex-col items-center py-4 text-gray-400">
      <div className="mb-8">
        <div className="w-10 h-10 bg-blue-600 rounded-lg flex items-center justify-center text-white font-bold cursor-pointer hover:bg-blue-500 transition-colors">
          W
        </div>
      </div>
      
      <div className="flex flex-col gap-6 flex-1">
        <SidebarIcon icon={<MessageSquare size={24} />} active />
        <SidebarIcon icon={<Terminal size={24} />} />
        <SidebarIcon icon={<Database size={24} />} />
        <SidebarIcon icon={<Activity size={24} />} />
      </div>

      <div className="flex flex-col gap-4 items-center">
        <div 
          className={`w-3 h-3 rounded-full ${connected ? 'bg-green-500' : 'bg-red-500'}`} 
          title={connected ? "Connected to Backend" : "Disconnected from Backend"}
        />
        <SidebarIcon icon={<Settings size={24} />} />
      </div>
    </div>
  );
}

function SidebarIcon({ icon, active }: { icon: React.ReactNode, active?: boolean }) {
  return (
    <div className={`p-2 rounded-lg cursor-pointer transition-colors ${active ? 'bg-[#2C2C2C] text-white' : 'hover:bg-[#2C2C2C] hover:text-white'}`}>
      {icon}
    </div>
  );
}
