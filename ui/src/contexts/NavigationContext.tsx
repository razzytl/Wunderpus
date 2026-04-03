import { createContext, useContext, useState, type ReactNode } from 'react';
import type { ViewType } from '../types/navigation';

interface NavigationValue {
  activeView: ViewType;
  setActiveView: (view: ViewType) => void;
}

const NavigationContext = createContext<NavigationValue | null>(null);

export function NavigationProvider({ children }: { children: ReactNode }) {
  const [activeView, setActiveView] = useState<ViewType>('chat');

  return (
    <NavigationContext.Provider value={{ activeView, setActiveView }}>
      {children}
    </NavigationContext.Provider>
  );
}

export function useNavigation(): NavigationValue {
  const ctx = useContext(NavigationContext);
  if (!ctx) throw new Error('useNavigation must be used within NavigationProvider');
  return ctx;
}
