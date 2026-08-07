import React, { useEffect, useRef, useState } from 'react';
import { X, Terminal as TerminalIcon, Circle, RefreshCw } from 'lucide-react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';

export default function TerminalModal({ isOpen, onClose, activeProjectID }) {
  const terminalRef = useRef(null);
  const termInstanceRef = useRef(null);
  const fitAddonRef = useRef(null);
  const wsRef = useRef(null);
  const [status, setStatus] = useState('connecting'); // connecting, connected, disconnected
  const [sessionKey, setSessionKey] = useState(0);

  const initTerminalSession = async () => {
    if (!terminalRef.current) return;

    try {
      setStatus('connecting');
      if (wsRef.current) {
        wsRef.current.close();
      }
      if (termInstanceRef.current) {
        termInstanceRef.current.dispose();
      }

      // Initialize xterm
      const term = new Terminal({
        cursorBlink: true,
        theme: {
          background: '#121212',
          foreground: '#e0e0e0',
          cursor: '#d97757',
          selectionBackground: '#3a3a3a',
        },
        fontSize: 14,
        fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      });

      const fitAddon = new FitAddon();
      term.loadAddon(fitAddon);
      term.open(terminalRef.current);
      fitAddon.fit();
      term.focus();

      termInstanceRef.current = term;
      fitAddonRef.current = fitAddon;

      const authToken = localStorage.getItem('auth_token');
      const headers = authToken ? { 'Authorization': `Bearer ${authToken}` } : {};
      const res = await fetch(`/api/v1/projects/${activeProjectID}/terminal/session`, {
        method: 'POST',
        headers,
      });
      if (!res.ok) throw new Error('Failed session');
      const data = await res.json();

      // Connect to WebSocket PTY
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const host = window.location.host;
      let wsUrl = `${protocol}//${host}/api/v1/terminal/${data.session_token}`;

      let ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      let fallbackAttempted = false;

      const setupWSHandlers = (targetWs) => {
        targetWs.onopen = () => {
          setStatus('connected');
          term.focus();
        };

        targetWs.onmessage = (event) => {
          term.write(event.data);
        };

        targetWs.onclose = () => {
          setStatus('disconnected');
        };

        targetWs.onerror = () => {
          if (!fallbackAttempted && host.includes(':3000')) {
            fallbackAttempted = true;
            console.warn('Vite proxy WS failed, trying direct backend port :8080...');
            const directUrl = `${protocol}//localhost:8080/api/v1/terminal/${data.session_token}`;
            const fallbackWs = new WebSocket(directUrl);
            wsRef.current = fallbackWs;
            setupWSHandlers(fallbackWs);
          } else {
            setStatus('disconnected');
          }
        };
      };

      setupWSHandlers(ws);

      term.onData((dataStr) => {
        if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
          wsRef.current.send(dataStr);
        }
      });
    } catch (err) {
      console.error('Terminal session error:', err);
      setStatus('disconnected');
      if (termInstanceRef.current) {
        termInstanceRef.current.write('\r\n\x1b[31mFailed to connect to Daytona Cloud Sandbox PTY.\x1b[0m\r\n');
      }
    }
  };

  useEffect(() => {
    if (!isOpen) return;

    initTerminalSession();

    const handleResize = () => {
      if (fitAddonRef.current) fitAddonRef.current.fit();
    };
    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      if (wsRef.current) wsRef.current.close();
      if (termInstanceRef.current) termInstanceRef.current.dispose();
    };
  }, [isOpen, activeProjectID, sessionKey]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-xs p-6">
      <div className="w-full max-w-5xl h-[85vh] bg-[#121212] border border-[#333333] rounded-xl shadow-2xl overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-3 border-b border-[#2a2a2a] bg-[#1a1a1a]">
          <div className="flex items-center space-x-3">
            <TerminalIcon className="w-5 h-5 text-[#d97757]" />
            <h2 className="text-sm font-medium text-white">Daytona Sandbox Live Terminal (Project: {activeProjectID})</h2>
            <div className="flex items-center space-x-1.5 px-2.5 py-0.5 rounded-full bg-[#262626] border border-[#383838]">
              <Circle
                className={`w-2 h-2 fill-current ${
                  status === 'connected'
                    ? 'text-emerald-400'
                    : status === 'connecting'
                    ? 'text-amber-400 animate-pulse'
                    : 'text-red-400'
                }`}
              />
              <span className="text-[11px] text-gray-300 font-mono capitalize">{status}</span>
            </div>
          </div>

          <div className="flex items-center space-x-2">
            <button
              onClick={() => setSessionKey((prev) => prev + 1)}
              className="px-2.5 py-1 text-xs text-gray-300 hover:text-white bg-[#262626] hover:bg-[#333] border border-[#383838] rounded-md transition-colors flex items-center space-x-1"
              title="Reconnect Terminal Session"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${status === 'connecting' ? 'animate-spin' : ''}`} />
              <span>Reconnect</span>
            </button>
            <button onClick={onClose} className="text-gray-400 hover:text-white p-1 rounded-md hover:bg-[#2c2c2c]">
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Terminal mount area */}
        <div className="flex-1 p-4 bg-[#121212] overflow-hidden">
          <div ref={terminalRef} className="h-full w-full" />
        </div>
      </div>
    </div>
  );
}
