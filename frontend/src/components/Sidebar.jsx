import React, { useState, useEffect } from 'react';
import { Sparkles, Folder, Terminal, FileText, GitBranch, ArrowUpRight } from 'lucide-react';

export default function Sidebar({
  activeProject,
  uncommittedFiles,
  onOpenModal,
  onViewDiff,
  onUpdateProjectRemote,
  onPushGit,
}) {
  const [gitRemote, setGitRemote] = useState('');

  useEffect(() => {
    if (activeProject && activeProject.git_remote_url) {
      setGitRemote(activeProject.git_remote_url);
    } else {
      setGitRemote('');
    }
  }, [activeProject]);

  const handleRemoteBlur = async () => {
    if (!activeProject) return;
    if (gitRemote === activeProject.git_remote_url) return;
    await onUpdateProjectRemote(gitRemote);
  };

  const getStatusBadge = (status) => {
    switch (status) {
      case 'A':
        return <span className="w-4 h-4 rounded-xs bg-emerald-950 text-emerald-400 font-mono text-[10px] flex items-center justify-center font-bold">A</span>;
      case 'M':
        return <span className="w-4 h-4 rounded-xs bg-amber-950 text-amber-400 font-mono text-[10px] flex items-center justify-center font-bold">M</span>;
      case 'D':
        return <span className="w-4 h-4 rounded-xs bg-rose-950 text-rose-400 font-mono text-[10px] flex items-center justify-center font-bold">D</span>;
      default:
        return <span className="w-4 h-4 rounded-xs bg-gray-800 text-gray-400 font-mono text-[10px] flex items-center justify-center font-bold">?</span>;
    }
  };

  return (
    <aside className="w-[260px] h-full bg-[#1e1e1e] border-r border-[#2d2d2d] flex flex-col justify-between shrink-0 select-none text-gray-200">
      {/* Top Menu Section */}
      <div className="flex-1 flex flex-col overflow-y-auto">
        {/* Navigation Items */}
        <nav className="p-3 space-y-1">
          <button
            onClick={() => onOpenModal('skills')}
            className="w-full flex items-center space-x-3 px-3 py-2 text-sm font-normal text-gray-300 hover:text-white hover:bg-[#2c2c2c] rounded-lg transition-colors"
          >
            <Sparkles className="w-4 h-4 text-gray-400" />
            <span>Skills</span>
          </button>

          <button
            onClick={() => onOpenModal('projects')}
            className="w-full flex items-center space-x-3 px-3 py-2 text-sm font-normal text-gray-300 hover:text-white hover:bg-[#2c2c2c] rounded-lg transition-colors"
          >
            <Folder className="w-4 h-4 text-gray-400" />
            <span>Projects</span>
          </button>

          <button
            onClick={() => onOpenModal('terminal')}
            className="w-full flex items-center space-x-3 px-3 py-2 text-sm font-normal text-gray-300 hover:text-white hover:bg-[#2c2c2c] rounded-lg transition-colors"
          >
            <Terminal className="w-4 h-4 text-gray-400" />
            <span>Terminal</span>
          </button>

          <button
            onClick={() => onOpenModal('files')}
            className="w-full flex items-center space-x-3 px-3 py-2 text-sm font-normal text-gray-300 hover:text-white hover:bg-[#2c2c2c] rounded-lg transition-colors"
          >
            <FileText className="w-4 h-4 text-gray-400" />
            <span>File Explorer</span>
          </button>

          <button
            onClick={() => onOpenModal('uncommitted')}
            className="w-full flex items-center space-x-3 px-3 py-2 text-sm font-normal text-gray-300 hover:text-white hover:bg-[#2c2c2c] rounded-lg transition-colors"
          >
            <GitBranch className="w-4 h-4 text-gray-400" />
            <span>Uncommitted Git Files</span>
          </button>
        </nav>

        {/* Live Uncommitted Files List (if changes exist) */}
        {uncommittedFiles.length > 0 && (
          <div className="px-4 py-2 border-t border-[#2c2c2c]">
            <div className="text-[11px] font-medium text-gray-400 uppercase tracking-wider mb-2">
              Modified Sandbox Files ({uncommittedFiles.length})
            </div>
            <div className="space-y-1 max-h-36 overflow-y-auto pr-1">
              {uncommittedFiles.map((file) => (
                <div
                  key={file.path}
                  onClick={() => onViewDiff(file.path)}
                  className="flex items-center justify-between px-2 py-1 rounded hover:bg-[#2c2c2c] cursor-pointer text-xs group"
                >
                  <span className="truncate text-gray-300 group-hover:text-white font-mono text-[11px]">
                    {file.path}
                  </span>
                  {getStatusBadge(file.status)}
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Git Remote URL Section */}
        <div className="p-4 border-t border-[#2c2c2c] space-y-3 mt-auto">
          <label className="block text-xs font-semibold text-gray-300">
            Git Remote URL
          </label>
          <input
            type="text"
            placeholder="https://github.com/user/repo.git"
            value={gitRemote}
            onChange={(e) => setGitRemote(e.target.value)}
            onBlur={handleRemoteBlur}
            className="w-full bg-[#121212] border border-[#3a3a3a] rounded-lg px-3 py-2 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-[#7c3aed]"
          />

          <button
            onClick={onPushGit}
            className="w-full py-2 px-4 rounded-lg bg-[#6366f1] hover:bg-[#4f46e5] text-white text-xs font-semibold shadow-md transition-colors flex items-center justify-center space-x-1.5"
          >
            <span>Push</span>
          </button>
        </div>
      </div>

      {/* Bottom Pinned User Profile */}
      <div className="p-4 border-t border-[#2c2c2c] bg-[#1a1a1a] flex items-center space-x-3">
        <div className="w-9 h-9 rounded-full bg-[#3b82f6] text-white text-sm font-semibold flex items-center justify-center shadow-inner">
          A
        </div>
        <div className="flex flex-col">
          <span className="text-sm font-medium text-white leading-snug">Akshat</span>
          <span className="text-xs text-gray-400 leading-none">Pro Plan</span>
        </div>
      </div>
    </aside>
  );
}
