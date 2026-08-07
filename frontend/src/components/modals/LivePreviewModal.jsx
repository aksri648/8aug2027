import React, { useState, useEffect } from 'react';
import { X, RefreshCw, ExternalLink, Monitor, Tablet, Smartphone, Terminal, Eye, Tv } from 'lucide-react';

export default function LivePreviewModal({ isOpen, onClose, activeProjectID, activeProject }) {
  const [deviceMode, setDeviceMode] = useState('desktop'); // 'desktop' | 'tablet' | 'mobile'
  const [previewType, setPreviewType] = useState('novnc'); // 'novnc' | 'app'
  const [iframeKey, setIframeKey] = useState(0);
  const [loading, setLoading] = useState(true);
  const [previewInfo, setPreviewInfo] = useState(null);

  const authToken = localStorage.getItem('auth_token');

  // Same-origin safe endpoints for iframe stage to prevent X-Frame-Options blocking
  const localNoVNCUrl = `/api/v1/projects/${activeProjectID}/sandbox/novnc${authToken ? `?token=${authToken}` : ''}`;
  const localAppUrl = `/api/v1/projects/${activeProjectID}/sandbox/app${authToken ? `?token=${authToken}` : ''}`;

  const currentIframeSrc = previewType === 'novnc' ? localNoVNCUrl : localAppUrl;
  const externalUrl = previewInfo?.novnc_url || previewInfo?.preview_url;

  useEffect(() => {
    if (isOpen) {
      fetchPreviewInfo();
    }
  }, [isOpen, activeProjectID]);

  const fetchPreviewInfo = async () => {
    try {
      const headers = authToken ? { 'Authorization': `Bearer ${authToken}` } : {};
      const res = await fetch(`/api/v1/projects/${activeProjectID}/sandbox/preview`, { headers });
      if (res.ok) {
        const data = await res.json();
        setPreviewInfo(data);
      }
    } catch (e) {
      console.error('Error fetching preview info:', e);
    }
  };

  if (!isOpen) return null;

  const handleRefresh = () => {
    setLoading(true);
    setIframeKey((prev) => prev + 1);
  };

  const getContainerWidth = () => {
    switch (deviceMode) {
      case 'mobile':
        return 'w-[375px] h-[667px]';
      case 'tablet':
        return 'w-[768px] h-[85vh]';
      default:
        return 'w-full h-full';
    }
  };

  const formatDisplayAddress = () => {
    if (previewType === 'novnc' && externalUrl && (externalUrl.startsWith('http://') || externalUrl.startsWith('https://'))) {
      return externalUrl;
    }
    const path = previewType === 'novnc' ? localNoVNCUrl : localAppUrl;
    return `${window.location.origin}${path}`;
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-xs p-4">
      <div className="w-full max-w-6xl h-[90vh] bg-[#1e1e1e] border border-[#333333] rounded-xl shadow-2xl overflow-hidden flex flex-col">
        
        {/* Top Control Bar */}
        <div className="flex items-center justify-between px-6 py-3 border-b border-[#2c2c2c] bg-[#141414] shrink-0 text-xs">
          
          {/* Left Title & Status */}
          <div className="flex items-center space-x-3">
            <div className="flex items-center space-x-2 text-emerald-400 font-medium">
              <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
              <span className="font-semibold text-white">Daytona Live Preview</span>
            </div>
            <span className="text-gray-500">|</span>
            <span className="text-gray-400 font-mono text-[11px]">
              {activeProject?.name || 'Sandbox Workspace'}
            </span>
          </div>

          {/* Center Address Bar & Mode Controls */}
          <div className="flex items-center space-x-3 flex-1 max-w-2xl mx-4">
            
            {/* View Mode Toggle: noVNC vs App */}
            <div className="flex items-center bg-[#252525] border border-[#383838] rounded-lg p-0.5 space-x-1 shrink-0">
              <button
                onClick={() => { setPreviewType('novnc'); handleRefresh(); }}
                className={`px-2.5 py-1 rounded flex items-center space-x-1.5 transition-colors ${
                  previewType === 'novnc' ? 'bg-[#d97757] text-white font-medium' : 'text-gray-400 hover:text-white'
                }`}
                title="noVNC VNC Live Screen Stream"
              >
                <Tv className="w-3.5 h-3.5" />
                <span>noVNC Screen</span>
              </button>
              <button
                onClick={() => { setPreviewType('app'); handleRefresh(); }}
                className={`px-2.5 py-1 rounded flex items-center space-x-1.5 transition-colors ${
                  previewType === 'app' ? 'bg-[#0284c7] text-white font-medium' : 'text-gray-400 hover:text-white'
                }`}
                title="App Server Preview (:8080)"
              >
                <Eye className="w-3.5 h-3.5" />
                <span>App Server</span>
              </button>
            </div>

            {/* Address Bar */}
            <div className="flex-1 flex items-center bg-[#252525] border border-[#383838] rounded-lg px-3 py-1.5 space-x-2 text-gray-300 overflow-hidden">
              <span className="font-mono text-[11px] truncate flex-1 text-gray-200" title={formatDisplayAddress()}>
                {formatDisplayAddress()}
              </span>
              <button
                onClick={handleRefresh}
                className="text-gray-400 hover:text-white p-0.5 rounded transition-colors"
                title="Refresh Preview"
              >
                <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
              </button>
            </div>

            {/* Device Toggles */}
            <div className="flex items-center bg-[#252525] border border-[#383838] rounded-lg p-0.5 space-x-1 shrink-0">
              <button
                onClick={() => setDeviceMode('desktop')}
                className={`p-1 rounded ${deviceMode === 'desktop' ? 'bg-[#383838] text-white' : 'text-gray-400 hover:text-white'}`}
                title="Desktop View"
              >
                <Monitor className="w-3.5 h-3.5" />
              </button>
              <button
                onClick={() => setDeviceMode('tablet')}
                className={`p-1 rounded ${deviceMode === 'tablet' ? 'bg-[#383838] text-white' : 'text-gray-400 hover:text-white'}`}
                title="Tablet View"
              >
                <Tablet className="w-3.5 h-3.5" />
              </button>
              <button
                onClick={() => setDeviceMode('mobile')}
                className={`p-1 rounded ${deviceMode === 'mobile' ? 'bg-[#383838] text-white' : 'text-gray-400 hover:text-white'}`}
                title="Mobile View"
              >
                <Smartphone className="w-3.5 h-3.5" />
              </button>
            </div>
          </div>

          {/* Right Actions */}
          <div className="flex items-center space-x-2 shrink-0">
            {externalUrl && (
              <a
                href={externalUrl}
                target="_blank"
                rel="noreferrer"
                className="px-2.5 py-1 text-xs font-medium text-emerald-400 bg-emerald-950/60 border border-emerald-800/80 hover:bg-emerald-900/80 rounded-md transition-colors flex items-center space-x-1"
                title="Open directly in Daytona Cloud dashboard tab"
              >
                <span>Daytona Tab</span>
                <ExternalLink className="w-3.5 h-3.5 ml-1" />
              </a>
            )}
            <button onClick={onClose} className="p-1.5 text-gray-400 hover:text-white rounded-md hover:bg-[#2c2c2c]">
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Preview Iframe Stage Container */}
        <div className="flex-1 bg-[#121212] overflow-auto flex items-center justify-center relative p-4">
          <div className={`transition-all duration-300 shadow-2xl overflow-hidden rounded-xl bg-[#0f172a] border border-[#2d3748] ${getContainerWidth()}`}>
            <iframe
              key={iframeKey}
              src={currentIframeSrc}
              title="Daytona Sandbox Live Preview"
              onLoad={() => setLoading(false)}
              className="w-full h-full border-0"
            />
          </div>
        </div>

        {/* Footer Console Log Info */}
        <div className="px-6 py-2 border-t border-[#2c2c2c] bg-[#141414] text-[11px] font-mono text-gray-400 flex items-center justify-between shrink-0">
          <div className="flex items-center space-x-3">
            <Terminal className="w-3.5 h-3.5 text-[#d97757]" />
            <span>[Daytona Sandbox] Mode: {previewType === 'novnc' ? 'noVNC Stream (:6080)' : 'App Server (:8080)'} | Status: Connected</span>
          </div>
          <div className="flex items-center space-x-3">
            <span>Files: {previewInfo?.files_count || 4}</span>
            <span>Port: {previewType === 'novnc' ? '6080' : '8080'}</span>
          </div>
        </div>

      </div>
    </div>
  );
}
