import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import './index.css';
import App from './App.tsx';
import { WebSocketProvider } from './contexts/WebSocketContext.tsx';
import { NavigationProvider } from './contexts/NavigationContext.tsx';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <NavigationProvider>
      <WebSocketProvider>
        <App />
      </WebSocketProvider>
    </NavigationProvider>
  </StrictMode>,
);
