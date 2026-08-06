import React, { useState } from 'react';
import { X, Code2, CloudUpload, Cpu, Wrench, Play, CheckCircle2, GitBranch } from 'lucide-react';

export default function TaskWizardModal({ isOpen, onClose, activeProject, onExecuteAgentTask }) {
  const [activeAgent, setActiveAgent] = useState('appdeveloper'); // 'appdeveloper' | 'appdeployer' | 'llmdeployer' | 'appmaintainer'

  // App Developer State
  const [devPrompt, setDevPrompt] = useState('Build a Go REST API for user authentication with JWT');
  const [devStack, setDevStack] = useState('Go 1.22 REST API + Docker');
  const [autoDeploy, setAutoDeploy] = useState(false);

  // App Deployer State
  const [deploySource, setDeploySource] = useState('current'); // 'current' | 'github'
  const [githubUrl, setGithubUrl] = useState('https://github.com/user/payment-microservice.git');
  const [vmSize, setVmSize] = useState('Standard_B2s');
  const [azureRegion, setAzureRegion] = useState('eastus');

  // LLM Deployer State
  const [modelRepo, setModelRepo] = useState('meta-llama/Llama-3-8B-Instruct');
  const [servingFramework, setServingFramework] = useState('vLLM (Azure VM)');
  const [gpuType, setGpuType] = useState('NVIDIA A10G (24GB)');

  // App Maintainer State
  const [maintSource, setMaintSource] = useState('current'); // 'current' | 'github'
  const [maintGithubUrl, setMaintGithubUrl] = useState('https://github.com/user/ecommerce-api.git');
  const [maintPrompt, setMaintPrompt] = useState('Fix panic: nil pointer dereference on /checkout endpoint under load');
  const [runSandboxTests, setRunSandboxTests] = useState(true);

  if (!isOpen) return null;

  const handleSubmit = (e) => {
    e.preventDefault();

    let fullPrompt = '';
    let agentPayload = {};

    if (activeAgent === 'appdeveloper') {
      fullPrompt = `[App Developer Agent] ${devPrompt}. Preferred stack: ${devStack}.${autoDeploy ? ' Also deploy to Azure VM upon completion.' : ''}`;
      agentPayload = { agent: 'appdeveloper', prompt: devPrompt, stack: devStack, autoDeploy };
    } else if (activeAgent === 'appdeployer') {
      const repo = deploySource === 'github' ? githubUrl : activeProject?.git_remote_url || 'current sandbox codebase';
      fullPrompt = `[App Deployer Agent] Deploy microservice repo ${repo} to Azure VM size ${vmSize} in region ${azureRegion}.`;
      agentPayload = { agent: 'appdeployer', repo, vmSize, azureRegion };
    } else if (activeAgent === 'llmdeployer') {
      fullPrompt = `[LLM Deployer Agent] Deploy HuggingFace LLM model ${modelRepo} using framework ${servingFramework} on GPU ${gpuType}.`;
      agentPayload = { agent: 'llmdeployer', modelRepo, servingFramework, gpuType };
    } else if (activeAgent === 'appmaintainer') {
      const repo = maintSource === 'github' ? maintGithubUrl : activeProject?.git_remote_url || 'current sandbox codebase';
      fullPrompt = `[App Maintainer Agent] Clone repository ${repo} into disposable Daytona sandbox. Diagnose & fix issue: "${maintPrompt}".${runSandboxTests ? ' Run sandbox integration tests before completing.' : ''}`;
      agentPayload = { agent: 'appmaintainer', repo, prompt: maintPrompt, runSandboxTests };
    }

    onExecuteAgentTask(fullPrompt, agentPayload);
    onClose();
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-xs p-4">
      <div className="w-full max-w-3xl bg-[#222222] border border-[#333333] rounded-xl shadow-2xl overflow-hidden flex flex-col max-h-[90vh]">
        
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[#333333] bg-[#1a1a1a]">
          <div className="flex items-center space-x-2 text-[#d97757]">
            <Play className="w-5 h-5 fill-current" />
            <h2 className="text-lg font-semibold text-white">Agent Task Execution Wizard</h2>
          </div>
          <button onClick={onClose} className="text-gray-400 hover:text-white p-1 rounded-md hover:bg-[#333333]">
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Agent Selector Grid */}
        <div className="grid grid-cols-4 border-b border-[#333333] bg-[#1a1a1a]">
          <button
            onClick={() => setActiveAgent('appdeveloper')}
            className={`p-3 flex flex-col items-center space-y-1.5 border-b-2 text-xs font-semibold transition-colors ${
              activeAgent === 'appdeveloper'
                ? 'border-[#d97757] text-[#d97757] bg-[#25201d]'
                : 'border-transparent text-gray-400 hover:text-white hover:bg-[#222222]'
            }`}
          >
            <Code2 className="w-5 h-5" />
            <span>App Developer</span>
          </button>

          <button
            onClick={() => setActiveAgent('appdeployer')}
            className={`p-3 flex flex-col items-center space-y-1.5 border-b-2 text-xs font-semibold transition-colors ${
              activeAgent === 'appdeployer'
                ? 'border-[#d97757] text-[#d97757] bg-[#25201d]'
                : 'border-transparent text-gray-400 hover:text-white hover:bg-[#222222]'
            }`}
          >
            <CloudUpload className="w-5 h-5" />
            <span>App Deployer</span>
          </button>

          <button
            onClick={() => setActiveAgent('llmdeployer')}
            className={`p-3 flex flex-col items-center space-y-1.5 border-b-2 text-xs font-semibold transition-colors ${
              activeAgent === 'llmdeployer'
                ? 'border-[#d97757] text-[#d97757] bg-[#25201d]'
                : 'border-transparent text-gray-400 hover:text-white hover:bg-[#222222]'
            }`}
          >
            <Cpu className="w-5 h-5" />
            <span>LLM Deployer</span>
          </button>

          <button
            onClick={() => setActiveAgent('appmaintainer')}
            className={`p-3 flex flex-col items-center space-y-1.5 border-b-2 text-xs font-semibold transition-colors ${
              activeAgent === 'appmaintainer'
                ? 'border-[#d97757] text-[#d97757] bg-[#25201d]'
                : 'border-transparent text-gray-400 hover:text-white hover:bg-[#222222]'
            }`}
          >
            <Wrench className="w-5 h-5" />
            <span>App Maintainer</span>
          </button>
        </div>

        {/* Wizard Body Form */}
        <form onSubmit={handleSubmit} className="p-6 overflow-y-auto space-y-5 flex-1 text-sm text-gray-200">

          {/* APP DEVELOPER WIZARD */}
          {activeAgent === 'appdeveloper' && (
            <div className="space-y-4">
              <div className="p-3 rounded-lg bg-[#1a1a1a] border border-[#333333] text-xs text-gray-300">
                <span className="font-semibold text-white">App Developer Agent:</span> Generates complete application codebases inside the Daytona sandbox based on prompt requirements and stack specifications.
              </div>

              <div>
                <label className="block text-xs font-semibold text-gray-300 mb-1">
                  Application Prompt & Requirements
                </label>
                <textarea
                  rows={3}
                  value={devPrompt}
                  onChange={(e) => setDevPrompt(e.target.value)}
                  placeholder="Describe the application, API endpoints, microservices, or database requirements..."
                  className="w-full bg-[#181818] border border-[#3a3a3a] rounded-lg p-3 text-xs text-white focus:outline-none focus:border-[#d97757]"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-gray-300 mb-1">
                  Target Tech Stack
                </label>
                <select
                  value={devStack}
                  onChange={(e) => setDevStack(e.target.value)}
                  className="w-full bg-[#181818] border border-[#3a3a3a] rounded-lg p-2.5 text-xs text-white focus:outline-none focus:border-[#d97757]"
                >
                  <option value="Go 1.22 REST API + Docker">Go 1.22 REST API + Docker</option>
                  <option value="React + Vite + Tailwind CSS">React + Vite + Tailwind CSS</option>
                  <option value="Python FastAPI + PostgreSQL">Python FastAPI + PostgreSQL</option>
                  <option value="Next.js Fullstack (App Router)">Next.js Fullstack (App Router)</option>
                </select>
              </div>

              <div className="flex items-center space-x-2 pt-2">
                <input
                  type="checkbox"
                  id="autoDeploy"
                  checked={autoDeploy}
                  onChange={(e) => setAutoDeploy(e.target.checked)}
                  className="rounded border-[#3a3a3a] bg-[#181818] text-[#d97757] focus:ring-0"
                />
                <label htmlFor="autoDeploy" className="text-xs text-gray-300 cursor-pointer">
                  Auto-trigger <span className="font-semibold text-white">App Deployer Agent</span> to provision Azure VM upon completion
                </label>
              </div>
            </div>
          )}

          {/* APP DEPLOYER WIZARD */}
          {activeAgent === 'appdeployer' && (
            <div className="space-y-4">
              <div className="p-3 rounded-lg bg-[#1a1a1a] border border-[#333333] text-xs text-gray-300">
                <span className="font-semibold text-white">App Deployer Agent:</span> Provisions Azure cloud infrastructure, builds container images, and deploys applications to live servers.
              </div>

              <div>
                <label className="block text-xs font-semibold text-gray-300 mb-2">
                  Deployment Codebase Source
                </label>
                <div className="grid grid-cols-2 gap-3">
                  <button
                    type="button"
                    onClick={() => setDeploySource('current')}
                    className={`p-3 rounded-lg border text-xs font-medium text-left flex items-center justify-between ${
                      deploySource === 'current'
                        ? 'bg-[#2a221f] border-[#d97757] text-white'
                        : 'bg-[#181818] border-[#333333] text-gray-400 hover:text-white'
                    }`}
                  >
                    <span>Current Sandbox Workspace</span>
                    {deploySource === 'current' && <CheckCircle2 className="w-4 h-4 text-[#d97757]" />}
                  </button>

                  <button
                    type="button"
                    onClick={() => setDeploySource('github')}
                    className={`p-3 rounded-lg border text-xs font-medium text-left flex items-center justify-between ${
                      deploySource === 'github'
                        ? 'bg-[#2a221f] border-[#d97757] text-white'
                        : 'bg-[#181818] border-[#333333] text-gray-400 hover:text-white'
                    }`}
                  >
                    <span>External GitHub Remote Repo</span>
                    {deploySource === 'github' && <CheckCircle2 className="w-4 h-4 text-[#d97757]" />}
                  </button>
                </div>
              </div>

              {deploySource === 'github' && (
                <div>
                  <label className="block text-xs font-semibold text-gray-300 mb-1 flex items-center space-x-1">
                    <GitBranch className="w-3.5 h-3.5 text-gray-400" />
                    <span>GitHub Repository Remote URL</span>
                  </label>
                  <input
                    type="text"
                    value={githubUrl}
                    onChange={(e) => setGithubUrl(e.target.value)}
                    placeholder="https://github.com/username/repository.git"
                    className="w-full bg-[#181818] border border-[#3a3a3a] rounded-lg p-2.5 text-xs text-white focus:outline-none focus:border-[#d97757]"
                  />
                </div>
              )}

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-semibold text-gray-300 mb-1">
                    Azure Compute Instance Size
                  </label>
                  <select
                    value={vmSize}
                    onChange={(e) => setVmSize(e.target.value)}
                    className="w-full bg-[#181818] border border-[#3a3a3a] rounded-lg p-2.5 text-xs text-white focus:outline-none focus:border-[#d97757]"
                  >
                    <option value="Standard_B2s">Standard_B2s (2 vCPU, 4GB RAM)</option>
                    <option value="Standard_D4s_v5">Standard_D4s_v5 (4 vCPU, 16GB RAM)</option>
                    <option value="Standard_NV36ads_A10_v5">Standard_NV36ads_A10_v5 (GPU)</option>
                  </select>
                </div>

                <div>
                  <label className="block text-xs font-semibold text-gray-300 mb-1">
                    Azure Region
                  </label>
                  <select
                    value={azureRegion}
                    onChange={(e) => setAzureRegion(e.target.value)}
                    className="w-full bg-[#181818] border border-[#3a3a3a] rounded-lg p-2.5 text-xs text-white focus:outline-none focus:border-[#d97757]"
                  >
                    <option value="eastus">East US (Virginia)</option>
                    <option value="westeurope">West Europe (Amsterdam)</option>
                    <option value="southeastasia">Southeast Asia (Singapore)</option>
                  </select>
                </div>
              </div>
            </div>
          )}

          {/* LLM DEPLOYER WIZARD */}
          {activeAgent === 'llmdeployer' && (
            <div className="space-y-4">
              <div className="p-3 rounded-lg bg-[#1a1a1a] border border-[#333333] text-xs text-gray-300">
                <span className="font-semibold text-white">LLM Deployer Agent:</span> Self-hosts open-weight LLMs (Llama 3, DeepSeek R1, Mistral) on dedicated Azure GPU instances using vLLM or NVIDIA NIM.
              </div>

              <div>
                <label className="block text-xs font-semibold text-gray-300 mb-1">
                  Hugging Face Model Repo ID
                </label>
                <input
                  type="text"
                  value={modelRepo}
                  onChange={(e) => setModelRepo(e.target.value)}
                  placeholder="meta-llama/Llama-3-8B-Instruct"
                  className="w-full bg-[#181818] border border-[#3a3a3a] rounded-lg p-2.5 text-xs text-white focus:outline-none focus:border-[#d97757]"
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-semibold text-gray-300 mb-1">
                    Serving Framework & Topology
                  </label>
                  <select
                    value={servingFramework}
                    onChange={(e) => setServingFramework(e.target.value)}
                    className="w-full bg-[#181818] border border-[#3a3a3a] rounded-lg p-2.5 text-xs text-white focus:outline-none focus:border-[#d97757]"
                  >
                    <option value="vLLM (Azure VM)">vLLM (High Throughput PagedAttention)</option>
                    <option value="NVIDIA NIM Container">NVIDIA NIM (TensorRT-LLM Microservice)</option>
                    <option value="Ollama Cluster">Ollama Container Instance</option>
                  </select>
                </div>

                <div>
                  <label className="block text-xs font-semibold text-gray-300 mb-1">
                    Target GPU Accelerator
                  </label>
                  <select
                    value={gpuType}
                    onChange={(e) => setGpuType(e.target.value)}
                    className="w-full bg-[#181818] border border-[#3a3a3a] rounded-lg p-2.5 text-xs text-white focus:outline-none focus:border-[#d97757]"
                  >
                    <option value="NVIDIA A10G (24GB)">NVIDIA A10G (24GB VRAM)</option>
                    <option value="NVIDIA T4 (16GB)">NVIDIA T4 (16GB VRAM)</option>
                    <option value="NVIDIA A100 (80GB)">NVIDIA A100 (80GB VRAM)</option>
                  </select>
                </div>
              </div>
            </div>
          )}

          {/* APP MAINTAINER WIZARD */}
          {activeAgent === 'appmaintainer' && (
            <div className="space-y-4">
              <div className="p-3 rounded-lg bg-[#1a1a1a] border border-[#333333] text-xs text-gray-300">
                <span className="font-semibold text-white">App Maintainer Agent:</span> Clones target repository into an isolated Daytona sandbox, executes tests/logs to reproduce bugs, applies code fixes, and verifies patches.
              </div>

              <div>
                <label className="block text-xs font-semibold text-gray-300 mb-2">
                  Target Maintenance Codebase
                </label>
                <div className="grid grid-cols-2 gap-3">
                  <button
                    type="button"
                    onClick={() => setMaintSource('current')}
                    className={`p-3 rounded-lg border text-xs font-medium text-left flex items-center justify-between ${
                      maintSource === 'current'
                        ? 'bg-[#2a221f] border-[#d97757] text-white'
                        : 'bg-[#181818] border-[#333333] text-gray-400 hover:text-white'
                    }`}
                  >
                    <span>Current Sandbox Workspace</span>
                    {maintSource === 'current' && <CheckCircle2 className="w-4 h-4 text-[#d97757]" />}
                  </button>

                  <button
                    type="button"
                    onClick={() => setMaintSource('github')}
                    className={`p-3 rounded-lg border text-xs font-medium text-left flex items-center justify-between ${
                      maintSource === 'github'
                        ? 'bg-[#2a221f] border-[#d97757] text-white'
                        : 'bg-[#181818] border-[#333333] text-gray-400 hover:text-white'
                    }`}
                  >
                    <span>External GitHub Remote Repo</span>
                    {maintSource === 'github' && <CheckCircle2 className="w-4 h-4 text-[#d97757]" />}
                  </button>
                </div>
              </div>

              {maintSource === 'github' && (
                <div>
                  <label className="block text-xs font-semibold text-gray-300 mb-1 flex items-center space-x-1">
                    <GitBranch className="w-3.5 h-3.5 text-gray-400" />
                    <span>GitHub Repository Remote URL</span>
                  </label>
                  <input
                    type="text"
                    value={maintGithubUrl}
                    onChange={(e) => setMaintGithubUrl(e.target.value)}
                    placeholder="https://github.com/username/repository.git"
                    className="w-full bg-[#181818] border border-[#3a3a3a] rounded-lg p-2.5 text-xs text-white focus:outline-none focus:border-[#d97757]"
                  />
                </div>
              )}

              <div>
                <label className="block text-xs font-semibold text-gray-300 mb-1">
                  Issue Description / Bug Trace / Refactoring Prompt
                </label>
                <textarea
                  rows={3}
                  value={maintPrompt}
                  onChange={(e) => setMaintPrompt(e.target.value)}
                  placeholder="Describe the bug, stack trace, failing test case, or refactoring goal..."
                  className="w-full bg-[#181818] border border-[#3a3a3a] rounded-lg p-3 text-xs text-white focus:outline-none focus:border-[#d97757]"
                />
              </div>

              <div className="flex items-center space-x-2 pt-1">
                <input
                  type="checkbox"
                  id="runSandboxTests"
                  checked={runSandboxTests}
                  onChange={(e) => setRunSandboxTests(e.target.checked)}
                  className="rounded border-[#3a3a3a] bg-[#181818] text-[#d97757] focus:ring-0"
                />
                <label htmlFor="runSandboxTests" className="text-xs text-gray-300 cursor-pointer">
                  Run sandbox unit & integration tests before finalizing bug fix patch
                </label>
              </div>
            </div>
          )}

          {/* Footer Action */}
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
              className="px-5 py-2.5 bg-[#d97757] hover:bg-[#c66849] text-white text-xs font-semibold rounded-lg flex items-center space-x-2 shadow-lg transition-colors"
            >
              <Play className="w-4 h-4 fill-current" />
              <span>Launch Agent Execution</span>
            </button>
          </div>

        </form>
      </div>
    </div>
  );
}
