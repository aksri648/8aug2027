import React, { useState, useEffect } from 'react';
import { Sparkles, Folder, Terminal, FileText, GitBranch, MessageSquare, Plus, Bot, Settings, Trash2, Eye } from 'lucide-react';

export default function Sidebar({
  projects = [],
  activeProjectID,
  activeProject,
  uncommittedFiles = [],
  onSelectProject,
  onCreateNewChat,
  onDeleteProject,
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
      <div className="flex-1 flex flex-col overflow-y-auto min-h-0">
        
        {/* Brand / New Chat Header */}
        <div className="p-3 border-b border-[#2c2c2c] space-y-2">
          <div className="flex items-center space-x-2 px-2 py-1">
            <Bot className="w-5 h-5 text-[#d97757]" />
            <span className="font-bold text-sm tracking-wide text-white">Agent Platform</span>
          </div>

          <button
            onClick={() => onCreateNewChat()}
            className="w-full flex items-center justify-center space-x-2 px-3 py-2 text-xs font-semibold text-white bg-[#d97757] hover:bg-[#c66849] rounded-lg transition-colors shadow-sm"
          >
            <Plus className="w-4 h-4" />
            <span>New Chat</span>
          </button>
        </div>

        {/* Chat Conversations Listing */}
        <div className="p-3 border-b border-[#2c2c2c] flex-1 min-h-[140px] overflow-y-auto">
          <div className="flex items-center justify-between px-2 mb-2">
            <span className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider">
              Chats ({projects.length})
            </span>
            <button
              onClick={() => onOpenModal('projects')}
              className="text-[10px] text-[#d97757] hover:underline"
            >
              View All
            </button>
          </div>

          <div className="space-y-1">
            {projects.map((p) => {
              const isActive = p.id === activeProjectID;
              return (
                <div
                  key={p.id}
                  onClick={() => onSelectProject(p.id)}
                  className={`group w-full flex items-center justify-between px-2.5 py-2 text-xs rounded-lg transition-colors cursor-pointer select-none ${
                    isActive
                      ? 'bg-[#2a221f] text-white font-medium border border-[#d97757]/40'
                      : 'text-gray-300 hover:bg-[#2c2c2c] hover:text-white'
                  }`}
                >
                  <div className="flex items-center space-x-2.5 min-w-0 flex-1">
                    <MessageSquare className={`w-3.5 h-3.5 shrink-0 ${isActive ? 'text-[#d97757]' : 'text-gray-400'}`} />
                    <span className="truncate">{p.name || 'Untitled Chat'}</span>
                  </div>

                  <div className="flex items-center space-x-1 shrink-0 ml-1">
                    {isActive && (
                      <span className="w-1.5 h-1.5 rounded-full bg-[#d97757] shrink-0 mr-1" />
                    )}
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        onDeleteProject(p.id);
                      }}
                      className="p-1 text-gray-500 hover:text-rose-400 hover:bg-[#383838] rounded transition-colors opacity-0 group-hover:opacity-100"
                      title="Delete Chat"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </div>
                </div>
              );
            })}

            {projects.length === 0 && (
              <div className="px-2 py-3 text-xs text-gray-500 italic text-center">
                No active chats found
              </div>
            )}
          </div>
        </div>

        {/* Navigation / Tools Items */}
        <nav className="p-3 space-y-1 border-b border-[#2c2c2c]">
          <div className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider px-2 mb-1">
            Workspace Tools
          </div>

          <button
            onClick={() => onOpenModal('preview')}
            className="w-full flex items-center space-x-3 px-3 py-1.5 text-xs font-semibold text-emerald-400 bg-emerald-950/40 border border-emerald-800/40 hover:bg-emerald-900/40 rounded-lg transition-colors"
          >
            <Eye className="w-4 h-4 text-emerald-400" />
            <span>Live Sandbox Preview</span>
          </button>

          <button
            onClick={() => onOpenModal('skills')}
            className="w-full flex items-center space-x-3 px-3 py-1.5 text-xs font-normal text-gray-300 hover:text-white hover:bg-[#2c2c2c] rounded-lg transition-colors"
          >
            <Sparkles className="w-4 h-4 text-gray-400" />
            <span>Skills</span>
          </button>

          <button
            onClick={() => onOpenModal('projects')}
            className="w-full flex items-center space-x-3 px-3 py-1.5 text-xs font-normal text-gray-300 hover:text-white hover:bg-[#2c2c2c] rounded-lg transition-colors"
          >
            <Folder className="w-4 h-4 text-gray-400" />
            <span>Projects Directory</span>
          </button>

          <button
            onClick={() => onOpenModal('terminal')}
            className="w-full flex items-center space-x-3 px-3 py-1.5 text-xs font-normal text-gray-300 hover:text-white hover:bg-[#2c2c2c] rounded-lg transition-colors"
          >
            <Terminal className="w-4 h-4 text-gray-400" />
            <span>Terminal</span>
          </button>

          <button
            onClick={() => onOpenModal('files')}
            className="w-full flex items-center space-x-3 px-3 py-1.5 text-xs font-normal text-gray-300 hover:text-white hover:bg-[#2c2c2c] rounded-lg transition-colors"
          >
            <FileText className="w-4 h-4 text-gray-400" />
            <span>File Explorer</span>
          </button>

          <button
            onClick={() => onOpenModal('uncommitted')}
            className="w-full flex items-center space-x-3 px-3 py-1.5 text-xs font-normal text-gray-300 hover:text-white hover:bg-[#2c2c2c] rounded-lg transition-colors"
          >
            <GitBranch className="w-4 h-4 text-gray-400" />
            <span className="flex-1 text-left">Uncommitted Git Files</span>
            <span className="px-1.5 py-0.5 text-[10px] font-bold rounded-full bg-[#3a3a3a] text-amber-400 font-mono">
              {uncommittedFiles.length}
            </span>
          </button>
        </nav>

        {/* Live Uncommitted Files List (if changes exist) */}
        {uncommittedFiles.length > 0 && (
          <div className="px-4 py-2 border-b border-[#2c2c2c]">
            <div className="text-[11px] font-medium text-gray-400 uppercase tracking-wider mb-2">
              Modified Sandbox Files ({uncommittedFiles.length})
            </div>
            <div className="space-y-1 max-h-28 overflow-y-auto pr-1">
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
        <div className="p-3 space-y-2 mt-auto">
          <label className="block text-[11px] font-semibold text-gray-300">
            Git Remote URL
          </label>
          <input
            type="text"
            placeholder="https://github.com/user/repo.git"
            value={gitRemote}
            onChange={(e) => setGitRemote(e.target.value)}
            onBlur={handleRemoteBlur}
            className="w-full bg-[#121212] border border-[#3a3a3a] rounded-lg px-3 py-1.5 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-[#7c3aed]"
          />

          <button
            onClick={onPushGit}
            className="w-full py-1.5 px-4 rounded-lg bg-[#6366f1] hover:bg-[#4f46e5] text-white text-xs font-semibold shadow-md transition-colors flex items-center justify-center space-x-1.5"
          >
            <span>Push</span>
          </button>
        </div>
      </div>

      {/* Bottom Pinned User Profile */}
      <div className="p-3 border-t border-[#2c2c2c] bg-[#1a1a1a] flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <div className="w-8 h-8 rounded-full bg-[#3b82f6] text-white text-xs font-semibold flex items-center justify-center shadow-inner">
            A
          </div>
          <div className="flex flex-col">
            <span className="text-xs font-medium text-white leading-snug">Akshat</span>
            <span className="text-[10px] text-gray-400 leading-none">Pro Plan</span>
          </div>
        </div>

        <button
          onClick={() => onOpenModal('config')}
          className="p-1.5 text-gray-400 hover:text-white hover:bg-[#2c2c2c] rounded-lg transition-colors"
          title="Platform Configuration Setup"
        >
          <Settings className="w-4 h-4" />
        </button>
      </div>
    </aside>
  );
}
