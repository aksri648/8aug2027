import React, { useState, useEffect } from 'react';
import { X, Sparkles, Upload, Plus, Trash2, FileText, CheckCircle2 } from 'lucide-react';

export default function SkillsModal({ isOpen, onClose }) {
  const [activeTab, setActiveTab] = useState('manual');
  const [skills, setSkills] = useState([]);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [content, setContent] = useState('');
  const [uploadStatus, setUploadStatus] = useState('');

  useEffect(() => {
    if (isOpen) {
      fetchSkills();
    }
  }, [isOpen]);

  const fetchSkills = async () => {
    try {
      const res = await fetch('/api/v1/skills');
      if (res.ok) {
        const data = await res.json();
        setSkills(data);
      }
    } catch (e) {
      console.error('Failed to fetch skills:', e);
    }
  };

  const handleCreateManual = async (e) => {
    e.preventDefault();
    if (!name || !content) return;
    try {
      const res = await fetch('/api/v1/skills', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, description, content }),
      });
      if (res.ok) {
        setName('');
        setDescription('');
        setContent('');
        fetchSkills();
      }
    } catch (e) {
      console.error('Error creating skill:', e);
    }
  };

  const handleFileUpload = async (e) => {
    const files = e.target.files;
    if (!files || files.length === 0) return;

    const formData = new FormData();
    for (let i = 0; i < files.length; i++) {
      formData.append('files', files[i]);
    }

    try {
      setUploadStatus('Uploading skills...');
      const res = await fetch('/api/v1/skills/upload', {
        method: 'POST',
        body: formData,
      });
      if (res.ok) {
        setUploadStatus('Successfully uploaded skill file(s)!');
        fetchSkills();
      } else {
        setUploadStatus('Upload failed');
      }
    } catch (err) {
      setUploadStatus('Error uploading file');
    }
  };

  const handleDelete = async (id) => {
    try {
      const res = await fetch(`/api/v1/skills/${id}`, { method: 'DELETE' });
      if (res.ok) {
        fetchSkills();
      }
    } catch (e) {
      console.error('Error deleting skill:', e);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-xs p-4">
      <div className="w-full max-w-3xl bg-[#222222] border border-[#333333] rounded-xl shadow-2xl overflow-hidden flex flex-col max-h-[85vh]">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[#333333] bg-[#1a1a1a]">
          <div className="flex items-center space-x-2 text-[#d97757]">
            <Sparkles className="w-5 h-5" />
            <h2 className="text-lg font-semibold text-white">Skills Library</h2>
          </div>
          <button onClick={onClose} className="text-gray-400 hover:text-white p-1 rounded-md hover:bg-[#333333]">
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-[#333333] bg-[#1c1c1c] px-6">
          <button
            onClick={() => setActiveTab('manual')}
            className={`py-3 px-4 text-sm font-medium border-b-2 transition-colors flex items-center space-x-2 ${
              activeTab === 'manual'
                ? 'border-[#d97757] text-[#d97757]'
                : 'border-transparent text-gray-400 hover:text-gray-200'
            }`}
          >
            <Plus className="w-4 h-4" />
            <span>Create Manually</span>
          </button>
          <button
            onClick={() => setActiveTab('upload')}
            className={`py-3 px-4 text-sm font-medium border-b-2 transition-colors flex items-center space-x-2 ${
              activeTab === 'upload'
                ? 'border-[#d97757] text-[#d97757]'
                : 'border-transparent text-gray-400 hover:text-gray-200'
            }`}
          >
            <Upload className="w-4 h-4" />
            <span>Upload Markdown</span>
          </button>
        </div>

        {/* Body */}
        <div className="p-6 overflow-y-auto space-y-6 flex-1">
          {activeTab === 'manual' && (
            <form onSubmit={handleCreateManual} className="space-y-4 bg-[#1a1a1a] p-4 rounded-lg border border-[#333333]">
              <div>
                <label className="block text-xs font-medium text-gray-300 mb-1">Skill Name</label>
                <input
                  type="text"
                  placeholder="e.g. Go Microservice Guidelines"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full bg-[#2a2a2a] border border-[#3a3a3a] rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-[#d97757]"
                  required
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-300 mb-1">Description (Guidance for Agent)</label>
                <input
                  type="text"
                  placeholder="e.g. Enforces Chi router, structured logs, and health checks"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  className="w-full bg-[#2a2a2a] border border-[#3a3a3a] rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-[#d97757]"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-300 mb-1">Instructions / Rules (.md)</label>
                <textarea
                  rows={4}
                  placeholder="# Playbook Instructions..."
                  value={content}
                  onChange={(e) => setContent(e.target.value)}
                  className="w-full bg-[#2a2a2a] border border-[#3a3a3a] rounded-lg p-3 text-sm font-mono text-white placeholder-gray-500 focus:outline-none focus:border-[#d97757]"
                  required
                />
              </div>
              <button
                type="submit"
                className="px-4 py-2 bg-[#d97757] hover:bg-[#c66849] text-white text-sm font-medium rounded-lg transition-colors flex items-center space-x-2"
              >
                <Plus className="w-4 h-4" />
                <span>Add Skill</span>
              </button>
            </form>
          )}

          {activeTab === 'upload' && (
            <div className="bg-[#1a1a1a] p-8 rounded-lg border-2 border-dashed border-[#3a3a3a] flex flex-col items-center justify-center text-center">
              <Upload className="w-10 h-10 text-[#d97757] mb-3" />
              <p className="text-sm font-medium text-white mb-1">Drag & drop skill markdown files</p>
              <p className="text-xs text-gray-400 mb-4">Accepts one or more <code className="bg-[#2a2a2a] px-1 py-0.5 rounded text-amber-300">&lt;skill-name&gt;.md</code> files</p>
              <label className="px-4 py-2 bg-[#2a2a2a] hover:bg-[#383838] border border-[#444] text-white text-sm font-medium rounded-lg cursor-pointer transition-colors">
                Browse Files
                <input type="file" multiple accept=".md" onChange={handleFileUpload} className="hidden" />
              </label>
              {uploadStatus && (
                <div className="mt-4 text-xs text-emerald-400 flex items-center space-x-1">
                  <CheckCircle2 className="w-4 h-4" />
                  <span>{uploadStatus}</span>
                </div>
              )}
            </div>
          )}

          {/* Registered Skills List */}
          <div>
            <h3 className="text-sm font-semibold text-gray-300 mb-3 uppercase tracking-wider text-xs">Registered Skills ({skills.length})</h3>
            {skills.length === 0 ? (
              <p className="text-xs text-gray-500 italic">No skills registered yet.</p>
            ) : (
              <div className="space-y-3">
                {skills.map((sk) => (
                  <div key={sk.id} className="bg-[#1a1a1a] p-4 rounded-lg border border-[#333333] flex items-start justify-between">
                    <div className="space-y-1">
                      <div className="flex items-center space-x-2">
                        <FileText className="w-4 h-4 text-[#d97757]" />
                        <span className="text-sm font-medium text-white">{sk.name}</span>
                        <span className="text-[10px] px-2 py-0.5 rounded-full bg-[#2a2a2a] text-gray-300 border border-[#3a3a3a] uppercase">
                          {sk.source}
                        </span>
                      </div>
                      <p className="text-xs text-gray-400">{sk.description || 'No description provided.'}</p>
                    </div>
                    <button
                      onClick={() => handleDelete(sk.id)}
                      className="text-gray-500 hover:text-red-400 p-1 rounded-md hover:bg-[#2c2c2c] transition-colors"
                      title="Delete Skill"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
