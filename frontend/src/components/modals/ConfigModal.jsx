import React, { useState, useEffect } from 'react';
import { X, Settings, Key, Server, Cloud, Database, Check, Plug, Compass, RefreshCw, AlertCircle, CheckCircle2 } from 'lucide-react';

export default function ConfigModal({ isOpen, onClose }) {
  const [activeTab, setActiveTab] = useState('llm');
  const [saved, setSaved] = useState(false);

  const [config, setConfig] = useState({
    anthropicKey: localStorage.getItem('cfg_anthropic_key') || 'sk-ant-api03-••••••••••••••••',
    openaiKey: localStorage.getItem('cfg_openai_key') || 'sk-proj-••••••••••••••••',
    deepseekKey: localStorage.getItem('cfg_deepseek_key') || 'sk-ds-••••••••••••••••',
    customOpenAIBaseURL: localStorage.getItem('cfg_custom_openai_base_url') || 'http://localhost:8000/v1',
    customOpenAIKey: localStorage.getItem('cfg_custom_openai_key') || '',
    customOpenAIModel: localStorage.getItem('cfg_custom_openai_model') || 'custom-llm-v1',
    daytonaServer: localStorage.getItem('cfg_daytona_server') || 'https://app.daytona.io/api',
    daytonaKey: localStorage.getItem('cfg_daytona_key') || 'daytona_pat_••••••••••••••••',
    azureSubId: localStorage.getItem('cfg_azure_sub_id') || '00000000-0000-0000-0000-000000000000',
    azureTenantId: localStorage.getItem('cfg_azure_tenant_id') || 'tenant-id-8831-4a99',
    azureClientId: localStorage.getItem('cfg_azure_client_id') || 'client-id-1122-3344',
    azureClientSecret: localStorage.getItem('cfg_azure_secret') || '••••••••••••••••',
    redisAddr: localStorage.getItem('cfg_redis_addr') || 'localhost:6379',
  });

  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState(null);

  const [discovering, setDiscovering] = useState(false);
  const [discoveredModels, setDiscoveredModels] = useState(() => {
    try {
      const saved = localStorage.getItem('cfg_discovered_models');
      return saved ? JSON.parse(saved) : [];
    } catch (e) {
      return [];
    }
  });

  if (!isOpen) return null;

  const handleChange = (field, val) => {
    setConfig((prev) => ({ ...prev, [field]: val }));
  };

  const handleSave = async (e) => {
    e.preventDefault();
    try {
      const token = localStorage.getItem('auth_token');
      await fetch('/api/v1/config', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { 'Authorization': `Bearer ${token}` } : {})
        },
        body: JSON.stringify(config),
      });
    } catch (e) {
      console.error('Error saving config to server:', e);
    }
    setSaved(true);
    setTimeout(() => {
      setSaved(false);
      onClose();
    }, 1200);
  };

  const handleTestConnection = async () => {
    setTesting(true);
    setTestResult(null);
    try {
      const token = localStorage.getItem('auth_token');
      const res = await fetch('/api/v1/providers/test', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { 'Authorization': `Bearer ${token}` } : {})
        },
        body: JSON.stringify({
          base_url: config.customOpenAIBaseURL,
          api_key: config.customOpenAIKey,
          model: config.customOpenAIModel,
        }),
      });
      const data = await res.json();
      setTestResult(data);
    } catch (err) {
      setTestResult({
        status: 'failed',
        error: `Connection error: ${err.message}`,
        latency: 0,
      });
    } finally {
      setTesting(false);
    }
  };

  const handleDiscoverModels = async () => {
    setDiscovering(true);
    try {
      const token = localStorage.getItem('auth_token');
      const res = await fetch('/api/v1/providers/discover', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { 'Authorization': `Bearer ${token}` } : {})
        },
        body: JSON.stringify({
          base_url: config.customOpenAIBaseURL,
          api_key: config.customOpenAIKey,
        }),
      });
      const data = await res.json();
      if (data.models && data.models.length > 0) {
        setDiscoveredModels(data.models);
        localStorage.setItem('cfg_discovered_models', JSON.stringify(data.models));
        if (!data.models.includes(config.customOpenAIModel)) {
          handleChange('customOpenAIModel', data.models[0]);
        }
      }
    } catch (err) {
      console.error('Model discovery error:', err);
    } finally {
      setDiscovering(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-xs p-4">
      <div className="w-full max-w-2xl bg-[#222222] border border-[#333333] rounded-xl shadow-2xl overflow-hidden flex flex-col max-h-[85vh]">
        {/* Modal Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[#333333] bg-[#1a1a1a]">
          <div className="flex items-center space-x-2 text-[#d97757]">
            <Settings className="w-5 h-5" />
            <h2 className="text-lg font-semibold text-white">Platform Configuration</h2>
          </div>
          <button onClick={onClose} className="text-gray-400 hover:text-white p-1 rounded-md hover:bg-[#333333]">
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Tab Navigation */}
        <div className="flex border-b border-[#333333] bg-[#1e1e1e] px-6">
          <button
            onClick={() => setActiveTab('llm')}
            className={`flex items-center space-x-2 py-3 px-4 text-xs font-semibold border-b-2 transition-colors ${
              activeTab === 'llm'
                ? 'border-[#d97757] text-[#d97757]'
                : 'border-transparent text-gray-400 hover:text-white'
            }`}
          >
            <Key className="w-4 h-4" />
            <span>LLM Providers</span>
          </button>

          <button
            onClick={() => setActiveTab('daytona')}
            className={`flex items-center space-x-2 py-3 px-4 text-xs font-semibold border-b-2 transition-colors ${
              activeTab === 'daytona'
                ? 'border-[#d97757] text-[#d97757]'
                : 'border-transparent text-gray-400 hover:text-white'
            }`}
          >
            <Server className="w-4 h-4" />
            <span>Daytona Sandbox</span>
          </button>

          <button
            onClick={() => setActiveTab('azure')}
            className={`flex items-center space-x-2 py-3 px-4 text-xs font-semibold border-b-2 transition-colors ${
              activeTab === 'azure'
                ? 'border-[#d97757] text-[#d97757]'
                : 'border-transparent text-gray-400 hover:text-white'
            }`}
          >
            <Cloud className="w-4 h-4" />
            <span>Azure Cloud</span>
          </button>

          <button
            onClick={() => setActiveTab('redis')}
            className={`flex items-center space-x-2 py-3 px-4 text-xs font-semibold border-b-2 transition-colors ${
              activeTab === 'redis'
                ? 'border-[#d97757] text-[#d97757]'
                : 'border-transparent text-gray-400 hover:text-white'
            }`}
          >
            <Database className="w-4 h-4" />
            <span>Queue & Store</span>
          </button>
        </div>

        {/* Form Body */}
        <form onSubmit={handleSave} className="p-6 overflow-y-auto space-y-4 flex-1">
          {activeTab === 'llm' && (
            <div className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-gray-300 mb-1">
                  Anthropic API Key (Claude Models)
                </label>
                <input
                  type="password"
                  value={config.anthropicKey}
                  onChange={(e) => handleChange('anthropicKey', e.target.value)}
                  className="w-full bg-[#181818] border border-[#3a3a3a] rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-[#d97757]"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-300 mb-1">
                  OpenAI API Key (GPT-4o, O1)
                </label>
                <input
                  type="password"
                  value={config.openaiKey}
                  onChange={(e) => handleChange('openaiKey', e.target.value)}
                  className="w-full bg-[#181818] border border-[#3a3a3a] rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-[#d97757]"
                />
              </div>

              {/* Custom OpenAI-Compatible Provider Setup */}
              <div className="p-4 rounded-lg bg-[#1a1a1a] border border-[#333333] space-y-3 pt-3">
                <div className="flex items-center justify-between">
                  <div className="text-xs font-semibold text-[#d97757] uppercase tracking-wider">
                    Custom OpenAI-Compatible Provider (vLLM / Ollama / LocalAI)
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-medium text-gray-300 mb-1">
                    API Base URL (e.g. http://localhost:8000/v1 or https://api.together.xyz/v1)
                  </label>
                  <input
                    type="text"
                    placeholder="http://localhost:8000/v1"
                    value={config.customOpenAIBaseURL}
                    onChange={(e) => handleChange('customOpenAIBaseURL', e.target.value)}
                    className="w-full bg-[#242424] border border-[#3a3a3a] rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-[#d97757]"
                  />
                </div>

                <div>
                  <label className="block text-xs font-medium text-gray-300 mb-1">
                    API Key (Optional for local servers like Ollama/vLLM)
                  </label>
                  <input
                    type="password"
                    placeholder="sk-custom-..."
                    value={config.customOpenAIKey}
                    onChange={(e) => handleChange('customOpenAIKey', e.target.value)}
                    className="w-full bg-[#242424] border border-[#3a3a3a] rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-[#d97757]"
                  />
                </div>

                <div>
                  <label className="block text-xs font-medium text-gray-300 mb-1">
                    Custom Model ID / Name
                  </label>
                  <input
                    type="text"
                    placeholder="deepseek-ai/DeepSeek-R1"
                    value={config.customOpenAIModel}
                    onChange={(e) => handleChange('customOpenAIModel', e.target.value)}
                    className="w-full bg-[#242424] border border-[#3a3a3a] rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-[#d97757]"
                  />
                </div>

                {/* Connection & Discovery Action Controls */}
                <div className="flex items-center space-x-2 pt-2">
                  <button
                    type="button"
                    onClick={handleTestConnection}
                    disabled={testing}
                    className="flex-1 flex items-center justify-center space-x-1.5 px-3 py-2 bg-[#2c2c2c] hover:bg-[#383838] text-xs font-medium text-gray-200 rounded-lg border border-[#444444] transition-colors disabled:opacity-50"
                  >
                    {testing ? (
                      <RefreshCw className="w-3.5 h-3.5 animate-spin text-amber-400" />
                    ) : (
                      <Plug className="w-3.5 h-3.5 text-emerald-400" />
                    )}
                    <span>{testing ? 'Testing...' : 'Test Connection'}</span>
                  </button>

                  <button
                    type="button"
                    onClick={handleDiscoverModels}
                    disabled={discovering}
                    className="flex-1 flex items-center justify-center space-x-1.5 px-3 py-2 bg-[#2c2c2c] hover:bg-[#383838] text-xs font-medium text-gray-200 rounded-lg border border-[#444444] transition-colors disabled:opacity-50"
                  >
                    {discovering ? (
                      <RefreshCw className="w-3.5 h-3.5 animate-spin text-[#d97757]" />
                    ) : (
                      <Compass className="w-3.5 h-3.5 text-[#d97757]" />
                    )}
                    <span>{discovering ? 'Discovering...' : 'Discover Models'}</span>
                  </button>
                </div>

                {/* Connection Test Feedback Badge */}
                {testResult && (
                  <div className={`p-2.5 rounded-lg border text-xs flex items-center space-x-2 ${
                    testResult.status === 'success'
                      ? 'bg-emerald-950/40 border-emerald-800/60 text-emerald-300'
                      : 'bg-rose-950/40 border-rose-800/60 text-rose-300'
                  }`}>
                    {testResult.status === 'success' ? (
                      <CheckCircle2 className="w-4 h-4 shrink-0 text-emerald-400" />
                    ) : (
                      <AlertCircle className="w-4 h-4 shrink-0 text-rose-400" />
                    )}
                    <div className="flex-1 font-mono text-[11px]">
                      <span>{testResult.message || testResult.error}</span>
                      {testResult.latency > 0 && (
                        <span className="ml-2 font-semibold">({testResult.latency}ms)</span>
                      )}
                    </div>
                  </div>
                )}

                {/* Discovered Models Tags */}
                {discoveredModels.length > 0 && (
                  <div className="space-y-1.5 pt-1">
                    <span className="text-[11px] font-medium text-gray-400">
                      Discovered Models ({discoveredModels.length}):
                    </span>
                    <div className="flex flex-wrap gap-1.5 max-h-24 overflow-y-auto">
                      {discoveredModels.map((m) => (
                        <button
                          key={m}
                          type="button"
                          onClick={() => handleChange('customOpenAIModel', m)}
                          className={`px-2 py-0.5 rounded text-[11px] font-mono transition-colors ${
                            config.customOpenAIModel === m
                              ? 'bg-[#d97757] text-white font-semibold'
                              : 'bg-[#2a2a2a] text-gray-300 hover:bg-[#383838] hover:text-white'
                          }`}
                        >
                          {m}
                        </button>
                      ))}
                    </div>
                  </div>
                )}

              </div>
            </div>
          )}

          {activeTab === 'daytona' && (
            <div className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-gray-300 mb-1">
                  Daytona Server URL
                </label>
                <input
                  type="text"
                  value={config.daytonaServer}
                  onChange={(e) => handleChange('daytonaServer', e.target.value)}
                  className="w-full bg-[#181818] border border-[#3a3a3a] rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-[#d97757]"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-300 mb-1">
                  Daytona Personal Access Token (PAT)
                </label>
                <input
                  type="password"
                  value={config.daytonaKey}
                  onChange={(e) => handleChange('daytonaKey', e.target.value)}
                  className="w-full bg-[#181818] border border-[#3a3a3a] rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-[#d97757]"
                />
              </div>
            </div>
          )}

          {activeTab === 'azure' && (
            <div className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-gray-300 mb-1">
                  Azure Subscription ID
                </label>
                <input
                  type="text"
                  value={config.azureSubId}
                  onChange={(e) => handleChange('azureSubId', e.target.value)}
                  className="w-full bg-[#181818] border border-[#3a3a3a] rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-[#d97757]"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-300 mb-1">
                  Azure Tenant ID
                </label>
                <input
                  type="text"
                  value={config.azureTenantId}
                  onChange={(e) => handleChange('azureTenantId', e.target.value)}
                  className="w-full bg-[#181818] border border-[#3a3a3a] rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-[#d97757]"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-300 mb-1">
                  Azure Client ID (Service Principal)
                </label>
                <input
                  type="text"
                  value={config.azureClientId}
                  onChange={(e) => handleChange('azureClientId', e.target.value)}
                  className="w-full bg-[#181818] border border-[#3a3a3a] rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-[#d97757]"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-300 mb-1">
                  Azure Client Secret
                </label>
                <input
                  type="password"
                  value={config.azureClientSecret}
                  onChange={(e) => handleChange('azureClientSecret', e.target.value)}
                  className="w-full bg-[#181818] border border-[#3a3a3a] rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-[#d97757]"
                />
              </div>
            </div>
          )}

          {activeTab === 'redis' && (
            <div className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-gray-300 mb-1">
                  Redis Queue Address
                </label>
                <input
                  type="text"
                  value={config.redisAddr}
                  onChange={(e) => handleChange('redisAddr', e.target.value)}
                  className="w-full bg-[#181818] border border-[#3a3a3a] rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-[#d97757]"
                />
              </div>
            </div>
          )}

          {/* Action Footer */}
          <div className="flex items-center justify-end space-x-3 pt-4 border-t border-[#333333]">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 bg-[#2a2a2a] hover:bg-[#333333] text-gray-300 text-xs font-medium rounded-lg"
            >
              Cancel
            </button>
            <button
              type="submit"
              className="px-5 py-2 bg-[#d97757] hover:bg-[#c66849] text-white text-xs font-semibold rounded-lg flex items-center space-x-1.5 shadow-md"
            >
              {saved ? (
                <>
                  <Check className="w-4 h-4 text-emerald-400" />
                  <span>Saved!</span>
                </>
              ) : (
                <span>Save Configuration</span>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
