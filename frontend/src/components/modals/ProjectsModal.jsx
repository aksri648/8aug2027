import React, { useState, useEffect } from 'react';
import { X, Folder, Plus, Check, GitBranch, Calendar } from 'lucide-react';

export default function ProjectsModal({ isOpen, onClose, activeProjectID, onSelectProject }) {
  const [projects, setProjects] = useState([]);
  const [newProjectName, setNewProjectName] = useState('');
  const [isCreating, setIsCreating] = useState(false);

  useEffect(() => {
    if (isOpen) {
      fetchProjects();
    }
  }, [isOpen]);

  const fetchProjects = async () => {
    try {
      const res = await fetch('/api/v1/projects');
      if (res.ok) {
        const data = await res.json();
        setProjects(data);
      }
    } catch (e) {
      console.error('Error fetching projects:', e);
    }
  };

  const handleCreateProject = async (e) => {
    e.preventDefault();
    try {
      const res = await fetch('/api/v1/projects', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newProjectName }),
      });
      if (res.ok) {
        const created = await res.json();
        setNewProjectName('');
        setIsCreating(false);
        fetchProjects();
        onSelectProject(created.id);
        onClose();
      }
    } catch (e) {
      console.error('Error creating project:', e);
    }
  };

  const getStatusBadge = (status) => {
    switch (status) {
      case 'deployed':
        return <span className="px-2 py-0.5 text-xs rounded-full bg-emerald-950 text-emerald-400 border border-emerald-800">Deployed</span>;
      case 'building':
        return <span className="px-2 py-0.5 text-xs rounded-full bg-amber-950 text-amber-400 border border-amber-800">Building</span>;
      case 'error':
        return <span className="px-2 py-0.5 text-xs rounded-full bg-red-950 text-red-400 border border-red-800">Error</span>;
      default:
        return <span className="px-2 py-0.5 text-xs rounded-full bg-gray-800 text-gray-400 border border-gray-700">Draft</span>;
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-xs p-4">
      <div className="w-full max-w-2xl bg-[#222222] border border-[#333333] rounded-xl shadow-2xl overflow-hidden flex flex-col max-h-[80vh]">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[#333333] bg-[#1a1a1a]">
          <div className="flex items-center space-x-2 text-[#d97757]">
            <Folder className="w-5 h-5" />
            <h2 className="text-lg font-semibold text-white">Projects Directory</h2>
          </div>
          <div className="flex items-center space-x-3">
            <button
              onClick={() => setIsCreating(!isCreating)}
              className="px-3 py-1.5 bg-[#d97757] hover:bg-[#c66849] text-white text-xs font-medium rounded-md transition-colors flex items-center space-x-1"
            >
              <Plus className="w-4 h-4" />
              <span>New Project</span>
            </button>
            <button onClick={onClose} className="text-gray-400 hover:text-white p-1 rounded-md hover:bg-[#333333]">
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Inline Create Form */}
        {isCreating && (
          <form onSubmit={handleCreateProject} className="p-4 bg-[#1a1a1a] border-b border-[#333333] flex items-center space-x-3">
            <input
              type="text"
              placeholder="Project Name (e.g. Payments Microservice)"
              value={newProjectName}
              onChange={(e) => setNewProjectName(e.target.value)}
              className="flex-1 bg-[#2a2a2a] border border-[#3a3a3a] rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-[#d97757]"
              autoFocus
            />
            <button
              type="submit"
              className="px-4 py-2 bg-[#d97757] hover:bg-[#c66849] text-white text-sm font-medium rounded-lg"
            >
              Create
            </button>
          </form>
        )}

        {/* Project List */}
        <div className="p-6 overflow-y-auto space-y-3 flex-1">
          {projects.map((p) => {
            const isActive = p.id === activeProjectID;
            return (
              <div
                key={p.id}
                onClick={() => {
                  onSelectProject(p.id);
                  onClose();
                }}
                className={`p-4 rounded-xl border transition-all cursor-pointer flex items-center justify-between ${
                  isActive
                    ? 'bg-[#2a221f] border-[#d97757] shadow-md'
                    : 'bg-[#1a1a1a] border-[#333333] hover:border-[#444444] hover:bg-[#202020]'
                }`}
              >
                <div className="space-y-1">
                  <div className="flex items-center space-x-3">
                    <span className="font-semibold text-white text-sm">{p.name}</span>
                    {getStatusBadge(p.status)}
                    {isActive && (
                      <span className="flex items-center space-x-1 text-xs text-[#d97757] font-medium">
                        <Check className="w-3.5 h-3.5" />
                        <span>Active Context</span>
                      </span>
                    )}
                  </div>
                  <div className="flex items-center space-x-4 text-xs text-gray-400">
                    <span className="flex items-center space-x-1">
                      <GitBranch className="w-3.5 h-3.5 text-gray-500" />
                      <span>{p.git_remote_url || 'No remote linked'}</span>
                    </span>
                    <span className="flex items-center space-x-1">
                      <Calendar className="w-3.5 h-3.5 text-gray-500" />
                      <span>{new Date(p.created_at).toLocaleDateString()}</span>
                    </span>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
