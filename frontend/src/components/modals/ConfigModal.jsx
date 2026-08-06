import React, { useState, useEffect } from 'react';
import { X, Settings, Key, Server, Cloud, Database, Check } from 'lucide-react';

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

  if (!isOpen) return null;

  const handleChange = (field, val) => {
    setConfig((prev) => ({ ...prev, [field]: val }));
  };

  const handleSave = (e) => {
    e.preventDefault();
    Object.entries(config).forEach(([k, v]) => {
      localStorage.setItem(`cfg_${k}`, v);
    });
    setSaved(true);
    setTimeout(() => {
      setSaved(false);
      onClose();
    }, 1200);
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

              <div>
                <label className="block text-xs font-medium text-gray-300 mb-1">
                  DeepSeek / vLLM Endpoint Key
                </label>
                <input
                  type="password"
                  value={config.deepseekKey}
                  onChange={(e) => handleChange('deepseekKey', e.target.value)}
                  className="w-full bg-[#181818] border border-[#3a3a3a] rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-[#d97757]"
                />
              </div>

              {/* Custom OpenAI-Compatible Provider Setup */}
              <div className="p-4 rounded-lg bg-[#1a1a1a] border border-[#333333] space-y-3 pt-3">
                <div className="text-xs font-semibold text-[#d97757] uppercase tracking-wider">
                  Custom OpenAI-Compatible Provider (vLLM / Ollama / LocalAI / LM Studio)
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
