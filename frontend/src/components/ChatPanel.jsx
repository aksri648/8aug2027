import React, { useState, useEffect, useRef } from 'react';
import { Menu, Star, SlidersHorizontal, ChevronDown, Plus, ArrowUp, Sparkles, Cpu, Wrench, Eye, Bot } from 'lucide-react';
import MarkdownRenderer from './MarkdownRenderer';

export default function ChatPanel({
  activeProject,
  messages,
  onSendMessage,
  streamingMessage,
  systemStatusEvents,
  onNewChat,
  onToggleSidebar,
  onOpenTaskWizard,
  onOpenConfig,
  onOpenLivePreview,
}) {
  const [prompt, setPrompt] = useState('');
  const [selectedModel, setSelectedModel] = useState('Claude 3.5 Sonnet');
  const [modelDropdownOpen, setModelDropdownOpen] = useState(false);
  const messagesEndRef = useRef(null);
  const textareaRef = useRef(null);

  const customModelName = localStorage.getItem('cfg_custom_openai_model') || 'Custom OpenAI Endpoint';
  const modelsList = [
    'Claude 3.5 Sonnet',
    'Claude 3 Opus',
    'DeepSeek R1 (vLLM)',
    'Meta Llama 3.3 70B (NIM)',
    `Custom (${customModelName})`,
  ];

  const suggestionChips = [
    'Write a function in Go',
    'Explain a concept',
    'Debug this error',
    'Summarize this document',
  ];

  const getGreetingTimeOfDay = () => {
    const hour = new Date().getHours();
    if (hour < 12) return 'morning';
    if (hour < 17) return 'afternoon';
    return 'evening';
  };

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, streamingMessage, systemStatusEvents]);

  const handleSend = (e) => {
    e?.preventDefault();
    if (!prompt.trim()) return;
    onSendMessage(prompt);
    setPrompt('');
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const handleTextareaChange = (e) => {
    setPrompt(e.target.value);
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
      textareaRef.current.style.height = `${Math.min(textareaRef.current.scrollHeight, 200)}px`;
    }
  };

  const [isStarred, setIsStarred] = useState(false);

  return (
    <div className="flex-1 h-full bg-[#232323] flex flex-col justify-between overflow-hidden relative text-gray-100">
      {/* Top Bar matching PNG exactly */}
      <header className="h-14 px-6 border-b border-[#2d2d2d] flex items-center justify-between bg-[#1e1e1e] shrink-0">
        <div className="flex items-center space-x-4">
          <button
            onClick={onToggleSidebar}
            className="text-gray-300 hover:text-white p-1.5 rounded-md hover:bg-[#2c2c2c] transition-colors"
            title="Toggle Sidebar"
          >
            <Menu className="w-5 h-5" />
          </button>

          {/* Backend Model Badge */}
          <div className="flex items-center space-x-1.5 text-xs font-medium text-emerald-400 bg-emerald-500/10 border border-emerald-500/20 px-2.5 py-1 rounded-lg">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
            <span>Backend Model (.env)</span>
          </div>
        </div>

        <div className="flex items-center space-x-3">
          <button
            onClick={onOpenLivePreview}
            className="flex items-center space-x-1.5 px-3 py-1.5 text-xs font-semibold text-white bg-emerald-700/80 hover:bg-emerald-600 rounded-lg transition-colors shadow-xs"
            title="Open Daytona Sandbox Live Preview"
          >
            <Eye className="w-3.5 h-3.5 text-emerald-300" />
            <span>Live Preview</span>
          </button>
          <button
            onClick={() => setIsStarred(!isStarred)}
            className="p-1.5 rounded-lg hover:bg-[#2c2c2c] transition-colors"
            title={isStarred ? "Unstar Chat Session" : "Star Chat Session"}
          >
            <Star className={`w-4 h-4 ${isStarred ? 'fill-amber-400 text-amber-400' : 'text-gray-400 hover:text-white'}`} />
          </button>
          <button
            onClick={onOpenTaskWizard}
            className="text-gray-400 hover:text-white p-1.5 rounded-lg hover:bg-[#2c2c2c] transition-colors"
            title="Agent Task Execution Wizard"
          >
            <Wrench className="w-4 h-4" />
          </button>
          <button
            onClick={onOpenConfig}
            className="text-gray-400 hover:text-white p-1.5 rounded-lg hover:bg-[#2c2c2c] transition-colors"
            title="Platform Configuration Setup"
          >
            <SlidersHorizontal className="w-4 h-4" />
          </button>
          {/* User Avatar Circle */}
          <div className="w-8 h-8 rounded-full bg-[#3b82f6] text-white text-xs font-semibold flex items-center justify-center shadow-inner">
            A
          </div>
        </div>
      </header>

      {/* Main Chat Canvas Area */}
      <div className="flex-1 overflow-y-auto px-4 md:px-12 py-6 flex flex-col items-center">
        {messages.length === 0 ? (
          /* Empty State View matching PNG exactly */
          <div className="w-full max-w-2xl flex flex-col items-center justify-center my-auto text-center space-y-3 pt-6">
            {/* Claude Orange Asterisk/Starburst Mark */}
            <div className="text-[#d97757] mb-1">
              <svg className="w-12 h-12 fill-current" viewBox="0 0 24 24">
                <path d="M12 2L13.8 8.2L20 10L13.8 11.8L12 18L10.2 11.8L4 10L10.2 8.2L12 2Z" />
              </svg>
            </div>

            <h1 className="text-3xl md:text-4xl font-serif text-white tracking-tight">
              Good {getGreetingTimeOfDay()}, Akshat
            </h1>
            <p className="text-sm text-gray-400">
              How can Claude help you today?
            </p>
          </div>
        ) : (
          /* Message List */
          <div className="w-full max-w-3xl space-y-6 pb-6">
            {messages.map((msg) => {
              if (msg.role === 'system-status') {
                return (
                  <div key={msg.id} className="flex items-center space-x-2 px-4 py-2 rounded-lg bg-[#2c2c2c] border border-[#3a3a3a] text-xs text-gray-300">
                    <Wrench className="w-4 h-4 text-[#d97757]" />
                    <span className="font-mono">{msg.content}</span>
                  </div>
                );
              }

              const isUser = msg.role === 'user';

              return (
                <div
                  key={msg.id}
                  className={`flex flex-col ${isUser ? 'items-end' : 'items-start'}`}
                >
                  {isUser ? (
                    /* User Chat Bubble */
                    <div className="bg-[#2d3039] text-white px-5 py-3.5 rounded-2xl rounded-tr-none shadow-md border border-[#3b3e4a] max-w-[85%] text-sm leading-relaxed">
                      {msg.content}
                    </div>
                  ) : (
                    /* AI Chat Bubble */
                    <div className="bg-[#1e2026] text-gray-100 px-5 py-4 rounded-2xl rounded-tl-none shadow-md border border-[#2b2d36] max-w-[92%] w-full space-y-3">
                      <div className="flex items-center justify-between border-b border-[#2b2d36] pb-2 mb-1">
                        <div className="flex items-center space-x-2">
                          <Sparkles className="w-4 h-4 text-[#d97757]" />
                          <span className="text-xs font-bold tracking-wide text-[#d97757]">AI Assistant</span>
                        </div>
                        <span className="text-[10px] text-gray-500 font-mono">ADK Agent Engine</span>
                      </div>

                      <div className="prose prose-invert max-w-none font-sans text-sm leading-relaxed text-gray-200">
                        <MarkdownRenderer content={msg.content} />
                      </div>

                      {/* Formatted UI Action & Question Card Component */}
                      {msg.content && (msg.content.toLowerCase().includes("how can i assist") || msg.content.toLowerCase().includes("option") || msg.content.toLowerCase().includes("select")) && (
                        <div className="mt-3 pt-3 border-t border-[#2b2d36] space-y-2">
                          <div className="text-[11px] font-semibold text-gray-300 flex items-center space-x-1.5">
                            <Bot className="w-3.5 h-3.5 text-[#d97757]" />
                            <span>Quick Task Actions & Choices</span>
                          </div>
                          <div className="grid grid-cols-2 gap-2">
                            <button
                              onClick={() => onSendMessage("Build a Go 1.22 REST API microservice with Docker containerization")}
                              className="text-left p-2.5 bg-[#262830] hover:bg-[#2d303b] border border-[#363845] rounded-xl text-xs text-gray-200 transition-colors"
                            >
                              <span className="font-semibold block text-[#d97757]">⚡ Build Go REST API</span>
                              <span className="text-[11px] text-gray-400">Generate Go 1.22 codebase</span>
                            </button>
                            <button
                              onClick={() => onSendMessage("Build a React + Vite + Tailwind CSS web dashboard application")}
                              className="text-left p-2.5 bg-[#262830] hover:bg-[#2d303b] border border-[#363845] rounded-xl text-xs text-gray-200 transition-colors"
                            >
                              <span className="font-semibold block text-emerald-400">🎨 Build React App</span>
                              <span className="text-[11px] text-gray-400">Generate React SPA</span>
                            </button>
                            <button
                              onClick={() => onSendMessage("Provision Azure Virtual Machine infrastructure for application deployment")}
                              className="text-left p-2.5 bg-[#262830] hover:bg-[#2d303b] border border-[#363845] rounded-xl text-xs text-gray-200 transition-colors"
                            >
                              <span className="font-semibold block text-blue-400">☁️ Azure Cloud Deploy</span>
                              <span className="text-[11px] text-gray-400">Provision Azure VM</span>
                            </button>
                            <button
                              onClick={() => onSendMessage("Diagnose and fix reported application errors in sandbox workspace")}
                              className="text-left p-2.5 bg-[#262830] hover:bg-[#2d303b] border border-[#363845] rounded-xl text-xs text-gray-200 transition-colors"
                            >
                              <span className="font-semibold block text-purple-400">🔧 Fix Application Bug</span>
                              <span className="text-[11px] text-gray-400">App Maintainer patch</span>
                            </button>
                          </div>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              );
            })}

            {/* System Status Events */}
            {systemStatusEvents.map((evt, idx) => (
              <div key={idx} className="flex items-center space-x-2 px-4 py-2 rounded-lg bg-[#2a2a2a] border border-[#383838] text-xs text-amber-300 font-mono animate-pulse">
                <Cpu className="w-4 h-4 text-[#d97757]" />
                <span>{evt.text}</span>
              </div>
            ))}

            {/* Streaming Token Message Bubble */}
            {streamingMessage && (
              <div className="bg-[#1e2026] text-gray-100 px-5 py-4 rounded-2xl rounded-tl-none shadow-md border border-[#2b2d36] max-w-[92%] w-full space-y-2">
                <div className="flex items-center space-x-2 mb-1">
                  <Sparkles className="w-4 h-4 text-[#d97757] animate-pulse" />
                  <span className="text-xs font-semibold text-[#d97757]">AI Assistant (Streaming Token Output...)</span>
                </div>
                <div className="prose prose-invert max-w-none font-sans text-sm leading-relaxed text-gray-200 bg-[#252730] p-3.5 rounded-xl border border-[#333642]">
                  <MarkdownRenderer content={streamingMessage} />
                  <span className="inline-block w-2 h-4 ml-1 bg-[#d97757] animate-pulse align-middle" />
                </div>
              </div>
            )}
            <div ref={messagesEndRef} />
          </div>
        )}
      </div>

      {/* Composer Container matching PNG */}
      <div className="w-full max-w-3xl mx-auto px-4 pb-6 pt-2 shrink-0">
        {/* Composer Rounded Rectangle Box */}
        <div className="bg-[#2c2c2c] border border-[#3a3a3a] rounded-2xl p-4 shadow-xl flex flex-col space-y-3 focus-within:border-[#555555] transition-colors">
          <textarea
            ref={textareaRef}
            rows={2}
            value={prompt}
            onChange={handleTextareaChange}
            onKeyDown={handleKeyDown}
            placeholder="Message Claude or launch agent..."
            className="w-full bg-transparent text-sm text-white placeholder-gray-500 focus:outline-none resize-none px-1"
          />

          <div className="flex items-center justify-between pt-1">
            <div className="flex items-center space-x-2">
              <button
                onClick={onOpenTaskWizard}
                className="p-1.5 text-gray-400 hover:text-white rounded-lg hover:bg-[#383838] transition-colors"
                title="Launch Agent Task Execution Wizard"
              >
                <Plus className="w-4 h-4" />
              </button>
              <button
                onClick={onOpenConfig}
                className="p-1.5 text-gray-400 hover:text-white rounded-lg hover:bg-[#383838] transition-colors"
                title="Platform Configuration Setup"
              >
                <SlidersHorizontal className="w-4 h-4" />
              </button>
            </div>

            <div className="flex items-center space-x-3">
              <button
                onClick={handleSend}
                disabled={!prompt.trim()}
                className={`p-2 rounded-full transition-all ${
                  prompt.trim()
                    ? 'bg-[#d97757] hover:bg-[#c66849] text-white shadow-md'
                    : 'bg-[#3a3a3a] text-gray-600 cursor-not-allowed'
                }`}
              >
                <ArrowUp className="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>

        {/* Suggestion Chips Row (directly below composer) */}
        <div className="flex flex-wrap items-center justify-center gap-2 mt-4">
          {suggestionChips.map((chip, i) => (
            <button
              key={i}
              onClick={() => onSendMessage(chip)}
              className="px-4 py-2 bg-[#2c2c2c] hover:bg-[#383838] border border-[#3a3a3a] rounded-full text-xs text-gray-300 hover:text-white transition-all shadow-xs"
            >
              {chip}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}
