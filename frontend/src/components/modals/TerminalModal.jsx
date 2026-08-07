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

  useEffect(() => {
    if (!isOpen || !terminalRef.current) return;

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

    termInstanceRef.current = term;
    fitAddonRef.current = fitAddon;

    // Create session and connect WS
    const initTerminalSession = async () => {
      try {
        setStatus('connecting');
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
        const wsUrl = `${protocol}//${host}/api/v1/terminal/${data.session_token}`;

        const ws = new WebSocket(wsUrl);
        wsRef.current = ws;

        ws.onopen = () => {
          setStatus('connected');
        };

        ws.onmessage = (event) => {
          term.write(event.data);
        };

        ws.onclose = () => {
          setStatus('disconnected');
        };

        ws.onerror = () => {
          setStatus('disconnected');
        };

        term.onData((data) => {
          if (ws.readyState === WebSocket.OPEN) {
            ws.send(data);
          }
        });
      } catch (err) {
        console.error('Terminal session error:', err);
        setStatus('disconnected');
        term.write('\r\n\x1b[31mFailed to connect to Daytona Cloud Sandbox PTY.\x1b[0m\r\n');
      }
    };

    initTerminalSession();

    const handleResize = () => fitAddon.fit();
    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      if (wsRef.current) wsRef.current.close();
      if (termInstanceRef.current) termInstanceRef.current.dispose();
    };
  }, [isOpen, activeProjectID]);

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
          <button onClick={onClose} className="text-gray-400 hover:text-white p-1 rounded-md hover:bg-[#2c2c2c]">
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Terminal mount area */}
        <div className="flex-1 p-4 bg-[#121212] overflow-hidden">
          <div ref={terminalRef} className="h-full w-full" />
        </div>
      </div>
    </div>
  );
}
