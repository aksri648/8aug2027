import React, { useState, useEffect, useRef } from 'react';
import Sidebar from './components/Sidebar';
import ChatPanel from './components/ChatPanel';
import SkillsModal from './components/modals/SkillsModal';
import ProjectsModal from './components/modals/ProjectsModal';
import TerminalModal from './components/modals/TerminalModal';
import FileExplorerModal from './components/modals/FileExplorerModal';
import GitDiffModal from './components/modals/GitDiffModal';
import SecretPromptModal from './components/modals/SecretPromptModal';
import ConfigModal from './components/modals/ConfigModal';
import TaskWizardModal from './components/modals/TaskWizardModal';
import LivePreviewModal from './components/modals/LivePreviewModal';

export default function App() {
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [activeProjectID, setActiveProjectID] = useState('proj-default');
  const [activeProject, setActiveProject] = useState(null);
  const [projects, setProjects] = useState([]);
  const [messages, setMessages] = useState([]);
  const [uncommittedFiles, setUncommittedFiles] = useState([]);
  const [openModal, setOpenModal] = useState(null); // 'skills' | 'projects' | 'terminal' | 'files' | 'config' | 'wizard' | 'preview'
  const [diffFilePath, setDiffFilePath] = useState(null);
  const [secretModalOpen, setSecretModalOpen] = useState(false);
  const [secretType, setSecretType] = useState('github_pat');
  const [systemStatusEvents, setSystemStatusEvents] = useState([]);
  const [streamingMessage, setStreamingMessage] = useState('');
  const [authToken, setAuthToken] = useState(() => localStorage.getItem('auth_token') || '');
  const wsRef = useRef(null);

  // Auto-login seeded user and validate auth token
  useEffect(() => {
    const initAuth = async () => {
      let token = localStorage.getItem('auth_token');
      let valid = false;

      if (token) {
        try {
          const testRes = await fetch('/api/v1/projects', {
            headers: { 'Authorization': `Bearer ${token}` }
          });
          if (testRes.ok) {
            valid = true;
            setAuthToken(token);
          } else {
            localStorage.removeItem('auth_token');
            setAuthToken('');
            token = null;
          }
        } catch (e) {
          valid = false;
        }
      }

      if (!valid) {
        try {
          const res = await fetch('/api/v1/auth/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email: 'developer@example.com', password: 'defaultpassword123' }),
          });
          if (res.ok) {
            const data = await res.json();
            token = data.token;
            localStorage.setItem('auth_token', token);
            setAuthToken(token);
          }
        } catch (e) {
          console.error('Failed auto-login:', e);
        }
      }
    };
    initAuth().then(() => {
      fetchProjects();
    });
  }, []);

  useEffect(() => {
    if (activeProjectID) {
      fetchProjectDetail(activeProjectID);
      fetchMessages(activeProjectID);
      fetchGitStatus(activeProjectID);
      connectWebSocket(activeProjectID);
    }

    return () => {
      if (wsRef.current) wsRef.current.close();
    };
  }, [activeProjectID, authToken]);

  const getAuthHeaders = () => {
    const token = authToken || localStorage.getItem('auth_token');
    return token ? { 'Authorization': `Bearer ${token}` } : {};
  };

  const fetchProjects = async () => {
    try {
      const res = await fetch('/api/v1/projects', { headers: getAuthHeaders() });
      if (res.ok) {
        const data = await res.json();
        setProjects(data);
        if (data.length > 0 && !data.some(p => p.id === activeProjectID)) {
          setActiveProjectID(data[0].id);
        }
      }
    } catch (e) {
      console.error('Failed to fetch projects:', e);
    }
  };

  const handleCreateNewChat = async (customName) => {
    const nameStr = (typeof customName === 'string' && customName.trim()) 
      ? customName 
      : `Chat ${projects.length + 1}`;

    try {
      const res = await fetch('/api/v1/projects', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
        body: JSON.stringify({ name: nameStr }),
      });
      if (res.ok) {
        const created = await res.json();
        setMessages([]);
        setActiveProjectID(created.id);
        await fetchProjects();
      }
    } catch (e) {
      console.error('Error creating chat:', e);
    }
  };

  const handleDeleteProject = async (pID) => {
    try {
      const res = await fetch(`/api/v1/projects/${pID}`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
      });
      if (res.ok) {
        const remaining = projects.filter((p) => p.id !== pID);
        setProjects(remaining);
        if (pID === activeProjectID) {
          if (remaining.length > 0) {
            setActiveProjectID(remaining[0].id);
          } else {
            handleCreateNewChat();
          }
        }
      }
    } catch (e) {
      console.error('Error deleting project:', e);
    }
  };

  const fetchProjectDetail = async (pID) => {
    try {
      const res = await fetch(`/api/v1/projects/${pID}`, { headers: getAuthHeaders() });
      if (res.ok) {
        const data = await res.json();
        setActiveProject(data);
      } else if (res.status === 404) {
        // Stale or deleted project ID: fallback to default project
        console.warn(`Project ${pID} not found (404). Falling back to proj-default.`);
        setActiveProjectID('proj-default');
        fetchProjects();
      }
    } catch (e) {
      console.error('Failed to fetch project:', e);
    }
  };

  const fetchMessages = async (pID) => {
    try {
      const res = await fetch(`/api/v1/projects/${pID}/messages`, { headers: getAuthHeaders() });
      if (res.ok) {
        const data = await res.json();
        setMessages(data);
      }
    } catch (e) {
      console.error('Failed to fetch messages:', e);
    }
  };

  const fetchGitStatus = async (pID) => {
    try {
      const res = await fetch(`/api/v1/projects/${pID}/git/status`, { headers: getAuthHeaders() });
      if (res.ok) {
        const data = await res.json();
        setUncommittedFiles(data.uncommitted || []);
      }
    } catch (e) {
      console.error('Failed to fetch git status:', e);
    }
  };

  const connectWebSocket = (pID) => {
    if (wsRef.current) {
      wsRef.current.close();
    }

    const token = authToken || localStorage.getItem('auth_token');
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const wsUrl = `${protocol}//${host}/api/v1/projects/${pID}/stream?token=${encodeURIComponent(token)}`;

    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onmessage = (evt) => {
      try {
        const event = JSON.parse(evt.data);
        if (event.type === 'git_status_changed') {
          setUncommittedFiles(event.uncommitted || []);
        } else if (event.type === 'system_status') {
          setSystemStatusEvents((prev) => [...prev, event]);
        } else if (event.type === 'chat_token') {
          setStreamingMessage((prev) => prev + event.delta);
        } else if (event.type === 'chat_message_complete') {
          setStreamingMessage('');
          fetchMessages(pID);
        } else if (event.type === 'job_update') {
          if (event.status === 'succeeded' || event.status === 'failed') {
            fetchGitStatus(pID);
            fetchMessages(pID);
          }
        }
      } catch (err) {
        console.error('Error parsing WS frame:', err);
      }
    };
  };

  const handleSendMessage = async (content, agentPayload = null) => {
    const optUserMsg = { id: 'temp-' + Date.now(), role: 'user', content };
    setMessages((prev) => [...prev, optUserMsg]);

    try {
      const res = await fetch(`/api/v1/projects/${activeProjectID}/messages`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
        body: JSON.stringify({ content, agent_payload: agentPayload }),
      });

      if (res.ok) {
        fetchMessages(activeProjectID);
      }
    } catch (e) {
      console.error('Error sending message:', e);
    }
  };

  const handleUpdateProjectRemote = async (gitRemoteURL) => {
    try {
      const res = await fetch(`/api/v1/projects/${activeProjectID}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
        body: JSON.stringify({ git_remote_url: gitRemoteURL }),
      });
      if (res.ok) {
        const updated = await res.json();
        setActiveProject(updated);
      }
    } catch (e) {
      console.error('Error updating project remote:', e);
    }
  };

  const handlePushGit = async () => {
    try {
      const res = await fetch(`/api/v1/projects/${activeProjectID}/git/push`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
        body: JSON.stringify({ commit_message: 'Update from SaaS Agentic Platform' }),
      });

      if (res.status === 428) {
        setSecretType('github_pat');
        setSecretModalOpen(true);
      } else if (res.ok) {
        fetchGitStatus(activeProjectID);
      }
    } catch (e) {
      console.error('Error pushing git:', e);
    }
  };

  return (
    <div className="flex h-screen w-screen overflow-hidden bg-[#1b1b1b]">
      {/* Fixed Left Sidebar */}
      {sidebarOpen && (
        <Sidebar
          projects={projects}
          activeProjectID={activeProjectID}
          activeProject={activeProject}
          uncommittedFiles={uncommittedFiles}
          onSelectProject={(pID) => setActiveProjectID(pID)}
          onCreateNewChat={handleCreateNewChat}
          onDeleteProject={handleDeleteProject}
          onOpenModal={(modalName) => setOpenModal(modalName)}
          onViewDiff={(filePath) => setDiffFilePath(filePath)}
          onUpdateProjectRemote={handleUpdateProjectRemote}
          onPushGit={handlePushGit}
        />
      )}

      {/* Main Chat Panel */}
      <ChatPanel
        activeProject={activeProject}
        messages={messages}
        onSendMessage={handleSendMessage}
        streamingMessage={streamingMessage}
        systemStatusEvents={systemStatusEvents}
        onNewChat={handleCreateNewChat}
        onToggleSidebar={() => setSidebarOpen((prev) => !prev)}
        onOpenTaskWizard={() => setOpenModal('wizard')}
        onOpenConfig={() => setOpenModal('config')}
        onOpenLivePreview={() => setOpenModal('preview')}
      />

      {/* Modals */}
      <SkillsModal
        isOpen={openModal === 'skills'}
        onClose={() => setOpenModal(null)}
      />

      <ProjectsModal
        isOpen={openModal === 'projects'}
        onClose={() => { setOpenModal(null); fetchProjects(); }}
        activeProjectID={activeProjectID}
        onSelectProject={(pID) => { setActiveProjectID(pID); fetchProjects(); }}
      />

      <TerminalModal
        isOpen={openModal === 'terminal'}
        onClose={() => setOpenModal(null)}
        activeProjectID={activeProjectID}
      />

      <FileExplorerModal
        isOpen={openModal === 'files'}
        onClose={() => setOpenModal(null)}
        activeProjectID={activeProjectID}
      />

      <GitDiffModal
        isOpen={!!diffFilePath}
        onClose={() => setDiffFilePath(null)}
        activeProjectID={activeProjectID}
        filePath={diffFilePath}
      />

      <ConfigModal
        isOpen={openModal === 'config'}
        onClose={() => setOpenModal(null)}
      />

      <TaskWizardModal
        isOpen={openModal === 'wizard'}
        onClose={() => setOpenModal(null)}
        activeProject={activeProject}
        onExecuteAgentTask={(promptText, payload) => {
          handleSendMessage(promptText, payload);
        }}
      />

      <LivePreviewModal
        isOpen={openModal === 'preview'}
        onClose={() => setOpenModal(null)}
        activeProjectID={activeProjectID}
        activeProject={activeProject}
      />

      <SecretPromptModal
        isOpen={secretModalOpen}
        onClose={() => setSecretModalOpen(false)}
        activeProjectID={activeProjectID}
        secretType={secretType}
        onSuccess={() => {
          handlePushGit();
        }}
      />
    </div>
  );
}
