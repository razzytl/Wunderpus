import { Terminal, Settings, MessageSquare, Database, Activity } from 'lucide-react';
import { useWebSocket } from '../../contexts/WebSocketContext';
import { useNavigation } from '../../contexts/NavigationContext';
import type { NavItem } from '../../types/navigation';

const navItems: NavItem[] = [
  { id: 'chat', label: 'Chat', icon: <MessageSquare size={24} />, description: 'Chat with AI agent' },
  { id: 'terminal', label: 'Terminal', icon: <Terminal size={24} />, description: 'Coming Soon' },
  { id: 'database', label: 'Database', icon: <Database size={24} />, description: 'Coming Soon' },
  { id: 'activity', label: 'Activity', icon: <Activity size={24} />, description: 'Coming Soon' },
];

export function Sidebar() {
  const { connected } = useWebSocket();
  const { activeView, setActiveView } = useNavigation();

  const handleNavClick = (item: NavItem) => {
    if (item.id === 'chat') {
      setActiveView(item.id);
    } else {
      alert(`${item.label} - Coming Soon`);
    }
  };

  return (
    <div className="w-16 h-screen border-r border-[#2C2C2C] bg-[#1E1E1E] flex flex-col items-center py-4 text-gray-400">
      <div className="mb-8">
        <div className="w-10 h-10 bg-blue-600 rounded-lg flex items-center justify-center text-white font-bold cursor-pointer hover:bg-blue-500 transition-colors">
          W
        </div>
      </div>
      
      <div className="flex flex-col gap-6 flex-1">
        {navItems.map((item) => (
          <SidebarIcon
            key={item.id}
            icon={item.icon}
            active={activeView === item.id}
            title={item.description}
            onClick={() => handleNavClick(item)}
          />
        ))}
      </div>

      <div className="flex flex-col gap-4 items-center">
        <div 
          className={`w-3 h-3 rounded-full ${connected ? 'bg-green-500' : 'bg-red-500'}`} 
          title={connected ? "Connected to Backend" : "Disconnected from Backend"}
        />
        <SidebarIcon icon={<Settings size={24} />} title="Settings" />
      </div>
    </div>
  );
}

function SidebarIcon({ icon, active, title, onClick }: { icon: React.ReactNode, active?: boolean, title?: string, onClick?: () => void }) {
  return (
    <div
      className={`p-2 rounded-lg cursor-pointer transition-colors ${active ? 'bg-[#2C2C2C] text-white' : 'hover:bg-[#2C2C2C] hover:text-white'}`}
      title={title}
      onClick={onClick}
    >
      {icon}
    </div>
  );
}
