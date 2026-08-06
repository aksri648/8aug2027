import React, { useState, useEffect, useRef } from 'react';
import { Menu, Star, SlidersHorizontal, ChevronDown, Plus, ArrowUp, Sparkles, Cpu, Wrench } from 'lucide-react';

export default function ChatPanel({
  activeProject,
  messages,
  onSendMessage,
  streamingMessage,
  systemStatusEvents,
  onNewChat,
}) {
  const [prompt, setPrompt] = useState('');
  const [selectedModel, setSelectedModel] = useState('Claude 3.5 Sonnet');
  const [modelDropdownOpen, setModelDropdownOpen] = useState(false);
  const messagesEndRef = useRef(null);
  const textareaRef = useRef(null);

  const modelsList = [
    'Claude 3.5 Sonnet',
    'Claude 3 Opus',
    'DeepSeek R1 (vLLM)',
    'Meta Llama 3.3 70B (NIM)',
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

  return (
    <div className="flex-1 h-full bg-[#232323] flex flex-col justify-between overflow-hidden relative text-gray-100">
      {/* Top Bar matching PNG exactly */}
      <header className="h-14 px-6 border-b border-[#2d2d2d] flex items-center justify-between bg-[#1e1e1e] shrink-0">
        <div className="flex items-center space-x-4">
          <button className="text-gray-300 hover:text-white p-1 rounded-md hover:bg-[#2c2c2c] transition-colors">
            <Menu className="w-5 h-5" />
          </button>
          
          <button
            onClick={onNewChat}
            className="text-sm font-medium text-gray-200 hover:text-white hover:bg-[#2c2c2c] px-3 py-1.5 rounded-lg transition-colors"
          >
            New Chat
          </button>

          {/* Model Selector Dropdown */}
          <div className="relative">
            <button
              onClick={() => setModelDropdownOpen(!modelDropdownOpen)}
              className="flex items-center space-x-1 text-sm font-medium text-gray-200 hover:text-white px-2.5 py-1.5 rounded-lg hover:bg-[#2c2c2c] transition-colors"
            >
              <span>Claude</span>
              <ChevronDown className="w-4 h-4 text-gray-400" />
            </button>

            {modelDropdownOpen && (
              <div className="absolute left-0 mt-1 w-56 bg-[#2a2a2a] border border-[#3a3a3a] rounded-xl shadow-2xl py-1 z-30">
                {modelsList.map((m) => (
                  <button
                    key={m}
                    onClick={() => {
                      setSelectedModel(m);
                      setModelDropdownOpen(false);
                    }}
                    className="w-full text-left px-4 py-2 text-xs font-medium text-gray-200 hover:bg-[#383838] hover:text-white flex items-center justify-between"
                  >
                    <span>{m}</span>
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>

        <div className="flex items-center space-x-3">
          <button className="text-gray-400 hover:text-white p-1.5 rounded-lg hover:bg-[#2c2c2c] transition-colors">
            <Star className="w-4 h-4" />
          </button>
          <button className="text-gray-400 hover:text-white p-1.5 rounded-lg hover:bg-[#2c2c2c] transition-colors">
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

              return (
                <div
                  key={msg.id}
                  className={`flex flex-col ${msg.role === 'user' ? 'items-end' : 'items-start'}`}
                >
                  {msg.role === 'user' ? (
                    <div className="bg-[#343434] text-white px-5 py-3 rounded-2xl max-w-[85%] text-sm leading-relaxed shadow-xs">
                      {msg.content}
                    </div>
                  ) : (
                    <div className="w-full text-gray-100 text-sm leading-relaxed space-y-2 pl-2">
                      <div className="flex items-center space-x-2 mb-1">
                        <Sparkles className="w-4 h-4 text-[#d97757]" />
                        <span className="text-xs font-semibold text-[#d97757]">Claude Assistant</span>
                      </div>
                      <div className="prose prose-invert max-w-none whitespace-pre-wrap font-sans">
                        {msg.content}
                      </div>
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

            {/* Streaming Token Message */}
            {streamingMessage && (
              <div className="text-gray-100 text-sm pl-2 space-y-1">
                <div className="flex items-center space-x-2">
                  <Sparkles className="w-4 h-4 text-[#d97757] animate-spin" />
                  <span className="text-xs font-semibold text-[#d97757]">Claude Assistant</span>
                </div>
                <div className="whitespace-pre-wrap">{streamingMessage}</div>
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
            placeholder="Message Claude..."
            className="w-full bg-transparent text-sm text-white placeholder-gray-500 focus:outline-none resize-none px-1"
          />

          <div className="flex items-center justify-between pt-1">
            <div className="flex items-center space-x-2">
              <button className="p-1.5 text-gray-400 hover:text-white rounded-lg hover:bg-[#383838] transition-colors" title="Attach file">
                <Plus className="w-4 h-4" />
              </button>
              <button className="p-1.5 text-gray-400 hover:text-white rounded-lg hover:bg-[#383838] transition-colors" title="Agent settings">
                <SlidersHorizontal className="w-4 h-4" />
              </button>
            </div>

            <div className="flex items-center space-x-3">
              <button
                onClick={() => setModelDropdownOpen(!modelDropdownOpen)}
                className="flex items-center space-x-1 text-xs text-gray-300 hover:text-white font-medium px-2.5 py-1 rounded-lg bg-[#383838] hover:bg-[#404040]"
              >
                <span>{selectedModel}</span>
                <ChevronDown className="w-3.5 h-3.5 text-gray-400" />
              </button>
              
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
