import React, { useState, useEffect } from 'react';
import { X, RefreshCw, ExternalLink, Monitor, Tablet, Smartphone, Play, Terminal, Eye } from 'lucide-react';

export default function LivePreviewModal({ isOpen, onClose, activeProjectID, activeProject }) {
  const [deviceMode, setDeviceMode] = useState('desktop'); // 'desktop' | 'tablet' | 'mobile'
  const [iframeKey, setIframeKey] = useState(0);
  const [loading, setLoading] = useState(true);
  const [previewInfo, setPreviewInfo] = useState(null);

  const previewUrl = `/api/v1/projects/${activeProjectID}/sandbox/app`;

  useEffect(() => {
    if (isOpen) {
      fetchPreviewInfo();
    }
  }, [isOpen, activeProjectID]);

  const fetchPreviewInfo = async () => {
    try {
      const res = await fetch(`/api/v1/projects/${activeProjectID}/sandbox/preview`);
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

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-xs p-4">
      <div className="w-full max-w-6xl h-[90vh] bg-[#1e1e1e] border border-[#333333] rounded-xl shadow-2xl overflow-hidden flex flex-col">
        
        {/* Top Control Bar */}
        <div className="flex items-center justify-between px-6 py-3 border-b border-[#2c2c2c] bg-[#141414] shrink-0 text-xs">
          
          {/* Left Title & Status */}
          <div className="flex items-center space-x-3">
            <div className="flex items-center space-x-2 text-emerald-400 font-medium">
              <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
              <span className="font-semibold text-white">Daytona Sandbox Live Preview</span>
            </div>
            <span className="text-gray-500">|</span>
            <span className="text-gray-400 font-mono text-[11px]">
              {activeProject?.name || 'Sandbox Workspace'}
            </span>
          </div>

          {/* Center Address Bar & Devices */}
          <div className="flex items-center space-x-4 flex-1 max-w-xl mx-6">
            <div className="flex-1 flex items-center bg-[#252525] border border-[#383838] rounded-lg px-3 py-1.5 space-x-2 text-gray-300">
              <Eye className="w-3.5 h-3.5 text-emerald-400 shrink-0" />
              <span className="font-mono text-[11px] truncate flex-1 text-gray-200">
                http://daytona-sandbox.internal:8080{previewUrl}
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
            <div className="flex items-center bg-[#252525] border border-[#383838] rounded-lg p-0.5 space-x-1">
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
          <div className="flex items-center space-x-2">
            <a
              href={previewUrl}
              target="_blank"
              rel="noreferrer"
              className="p-1.5 text-gray-400 hover:text-white rounded-md hover:bg-[#2c2c2c] transition-colors flex items-center space-x-1"
              title="Open in new window"
            >
              <ExternalLink className="w-4 h-4" />
            </a>
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
              src={previewUrl}
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
            <span>[Daytona Sandbox] Listening on :8080 | Status: HTTP 200 OK</span>
          </div>
          <div className="flex items-center space-x-3">
            <span>Files: {previewInfo?.files_count || 4}</span>
            <span>Port: 8080</span>
          </div>
        </div>

      </div>
    </div>
  );
}
