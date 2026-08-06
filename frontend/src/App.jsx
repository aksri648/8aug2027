import React, { useState, useEffect, useRef } from 'react';
import Sidebar from './components/Sidebar';
import ChatPanel from './components/ChatPanel';
import SkillsModal from './components/modals/SkillsModal';
import ProjectsModal from './components/modals/ProjectsModal';
import TerminalModal from './components/modals/TerminalModal';
import FileExplorerModal from './components/modals/FileExplorerModal';
import GitDiffModal from './components/modals/GitDiffModal';
import SecretPromptModal from './components/modals/SecretPromptModal';

export default function App() {
  const [activeProjectID, setActiveProjectID] = useState('proj-default');
  const [activeProject, setActiveProject] = useState(null);
  const [messages, setMessages] = useState([]);
  const [uncommittedFiles, setUncommittedFiles] = useState([]);
  const [openModal, setOpenModal] = useState(null); // 'skills' | 'projects' | 'terminal' | 'files'
  const [diffFilePath, setDiffFilePath] = useState(null);
  const [secretModalOpen, setSecretModalOpen] = useState(false);
  const [secretType, setSecretType] = useState('github_pat');
  const [systemStatusEvents, setSystemStatusEvents] = useState([]);
  const [streamingMessage, setStreamingMessage] = useState('');
  const wsRef = useRef(null);

  useEffect(() => {
    fetchProjectDetail(activeProjectID);
    fetchMessages(activeProjectID);
    fetchGitStatus(activeProjectID);
    connectWebSocket(activeProjectID);

    return () => {
      if (wsRef.current) wsRef.current.close();
    };
  }, [activeProjectID]);

  const fetchProjectDetail = async (pID) => {
    try {
      const res = await fetch(`/api/v1/projects/${pID}`);
      if (res.ok) {
        const data = await res.json();
        setActiveProject(data);
      }
    } catch (e) {
      console.error('Failed to fetch project:', e);
    }
  };

  const fetchMessages = async (pID) => {
    try {
      const res = await fetch(`/api/v1/projects/${pID}/messages`);
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
      const res = await fetch(`/api/v1/projects/${pID}/git/status`);
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

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const wsUrl = `${protocol}//${host}/api/v1/projects/${pID}/stream`;

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
          if (event.status === 'succeeded') {
            fetchGitStatus(pID);
            fetchMessages(pID);
          }
        }
      } catch (err) {
        console.error('Error parsing WS frame:', err);
      }
    };
  };

  const handleSendMessage = async (content) => {
    // Append optimistic user message
    const optUserMsg = { id: 'temp-' + Date.now(), role: 'user', content };
    setMessages((prev) => [...prev, optUserMsg]);

    try {
      const res = await fetch(`/api/v1/projects/${activeProjectID}/messages`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content }),
      });

      if (res.ok) {
        const data = await res.json();
        // Re-fetch server state
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
        headers: { 'Content-Type': 'application/json' },
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
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ commit_message: 'Update from SaaS Agentic Platform' }),
      });

      if (res.status === 428) {
        // Precondition Required (PAT secret missing)
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
      <Sidebar
        activeProject={activeProject}
        uncommittedFiles={uncommittedFiles}
        onOpenModal={(modalName) => setOpenModal(modalName)}
        onViewDiff={(filePath) => setDiffFilePath(filePath)}
        onUpdateProjectRemote={handleUpdateProjectRemote}
        onPushGit={handlePushGit}
      />

      {/* Main Chat Panel */}
      <ChatPanel
        activeProject={activeProject}
        messages={messages}
        onSendMessage={handleSendMessage}
        streamingMessage={streamingMessage}
        systemStatusEvents={systemStatusEvents}
      />

      {/* Modals */}
      <SkillsModal
        isOpen={openModal === 'skills'}
        onClose={() => setOpenModal(null)}
      />

      <ProjectsModal
        isOpen={openModal === 'projects'}
        onClose={() => setOpenModal(null)}
        activeProjectID={activeProjectID}
        onSelectProject={(pID) => setActiveProjectID(pID)}
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
