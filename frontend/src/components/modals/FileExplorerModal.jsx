import React, { useState, useEffect } from 'react';
import { X, FileText, Folder, RefreshCw, ChevronRight, ChevronDown, Code } from 'lucide-react';

export default function FileExplorerModal({ isOpen, onClose, activeProjectID }) {
  const [files, setFiles] = useState([]);
  const [selectedFile, setSelectedFile] = useState(null);
  const [fileContent, setFileContent] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (isOpen) {
      fetchFiles('/');
    }
  }, [isOpen, activeProjectID]);

  const fetchFiles = async (dirPath) => {
    try {
      setLoading(true);
      const res = await fetch(`/api/v1/projects/${activeProjectID}/files?path=${encodeURIComponent(dirPath)}`);
      if (res.ok) {
        const data = await res.json();
        setFiles(data);
      }
    } catch (e) {
      console.error('Error listing files:', e);
    } finally {
      setLoading(false);
    }
  };

  const handleSelectFile = async (item) => {
    if (item.is_dir) return;
    setSelectedFile(item.path);
    try {
      const res = await fetch(`/api/v1/projects/${activeProjectID}/files/content?path=${encodeURIComponent(item.path)}`);
      if (res.ok) {
        const data = await res.json();
        setFileContent(data.content);
      }
    } catch (e) {
      console.error('Error fetching file content:', e);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-xs p-6">
      <div className="w-full max-w-5xl h-[85vh] bg-[#222222] border border-[#333333] rounded-xl shadow-2xl overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[#333333] bg-[#1a1a1a]">
          <div className="flex items-center space-x-2 text-[#d97757]">
            <Folder className="w-5 h-5" />
            <h2 className="text-lg font-semibold text-white">Sandbox File Explorer</h2>
          </div>
          <div className="flex items-center space-x-3">
            <button
              onClick={() => fetchFiles('/')}
              className="p-1.5 text-gray-400 hover:text-white rounded-md hover:bg-[#333333] transition-colors"
              title="Refresh File Tree"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            </button>
            <button onClick={onClose} className="text-gray-400 hover:text-white p-1 rounded-md hover:bg-[#333333]">
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* 2-Pane Body */}
        <div className="flex-1 flex overflow-hidden">
          {/* Left Tree Pane */}
          <div className="w-1/3 border-r border-[#333333] bg-[#1a1a1a] p-4 overflow-y-auto">
            <div className="text-xs font-semibold text-gray-400 uppercase tracking-wider mb-3">Workspace Files</div>
            <div className="space-y-1">
              {files.map((item) => (
                <div
                  key={item.path}
                  onClick={() => handleSelectFile(item)}
                  className={`flex items-center space-x-2 px-3 py-2 rounded-lg cursor-pointer text-sm transition-colors ${
                    selectedFile === item.path
                      ? 'bg-[#2a221f] text-[#d97757] font-medium border border-[#d97757]/30'
                      : 'text-gray-300 hover:bg-[#252525]'
                  }`}
                >
                  {item.is_dir ? (
                    <Folder className="w-4 h-4 text-amber-400 shrink-0" />
                  ) : (
                    <FileText className="w-4 h-4 text-gray-400 shrink-0" />
                  )}
                  <span className="truncate">{item.name}</span>
                </div>
              ))}
            </div>
          </div>

          {/* Right Content Preview Pane */}
          <div className="flex-1 bg-[#151515] p-6 overflow-y-auto flex flex-col">
            {selectedFile ? (
              <div className="space-y-4 flex-1 flex flex-col">
                <div className="flex items-center justify-between pb-3 border-b border-[#2c2c2c]">
                  <div className="flex items-center space-x-2">
                    <Code className="w-4 h-4 text-[#d97757]" />
                    <span className="font-mono text-sm text-gray-200">{selectedFile}</span>
                  </div>
                  <span className="text-xs text-gray-500 font-mono">UTF-8</span>
                </div>
                <pre className="flex-1 p-4 bg-[#1e1e1e] rounded-lg border border-[#2c2c2c] font-mono text-sm text-gray-200 overflow-x-auto whitespace-pre-wrap">
                  {fileContent}
                </pre>
              </div>
            ) : (
              <div className="flex-1 flex flex-col items-center justify-center text-gray-500">
                <FileText className="w-12 h-12 mb-2 stroke-[1.5]" />
                <p className="text-sm">Select a file from the tree to preview contents.</p>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
