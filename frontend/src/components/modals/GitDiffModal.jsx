import React, { useEffect, useState } from 'react';
import { X, GitCommit, FileCode } from 'lucide-react';

export default function GitDiffModal({ isOpen, onClose, activeProjectID, filePath }) {
  const [diff, setDiff] = useState('');

  useEffect(() => {
    if (isOpen && filePath) {
      fetchDiff();
    }
  }, [isOpen, filePath, activeProjectID]);

  const fetchDiff = async () => {
    try {
      const res = await fetch(`/api/v1/projects/${activeProjectID}/git/diff?path=${encodeURIComponent(filePath)}`);
      if (res.ok) {
        const data = await res.json();
        setDiff(data.diff);
      }
    } catch (e) {
      console.error('Error fetching git diff:', e);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-xs p-6">
      <div className="w-full max-w-4xl h-[75vh] bg-[#222222] border border-[#333333] rounded-xl shadow-2xl overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[#333333] bg-[#1a1a1a]">
          <div className="flex items-center space-x-2 text-[#d97757]">
            <GitCommit className="w-5 h-5" />
            <h2 className="text-sm font-semibold text-white font-mono">Git Diff: {filePath}</h2>
          </div>
          <button onClick={onClose} className="text-gray-400 hover:text-white p-1 rounded-md hover:bg-[#333333]">
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Diff view */}
        <div className="flex-1 p-6 bg-[#181818] overflow-y-auto font-mono text-xs">
          {diff.split('\n').map((line, i) => {
            let color = 'text-gray-300';
            let bg = 'transparent';
            if (line.startsWith('+')) {
              color = 'text-emerald-400';
              bg = 'bg-emerald-950/40';
            } else if (line.startsWith('-')) {
              color = 'text-rose-400';
              bg = 'bg-rose-950/40';
            } else if (line.startsWith('@')) {
              color = 'text-amber-400';
            }
            return (
              <div key={i} className={`px-2 py-0.5 rounded-xs ${color} ${bg}`}>
                {line}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
