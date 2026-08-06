import React, { useState } from 'react';
import { X, Key, ShieldCheck } from 'lucide-react';

export default function SecretPromptModal({ isOpen, onClose, activeProjectID, secretType = 'github_pat', onSuccess }) {
  const [value, setValue] = useState('');
  const [saving, setSaving] = useState(false);

  if (!isOpen) return null;

  const getTitle = () => {
    switch (secretType) {
      case 'github_pat':
        return 'GitHub Personal Access Token';
      case 'azure_credentials':
        return 'Azure Service Principal Credentials';
      case 'huggingface_token':
        return 'Hugging Face Hub User Token';
      case 'nvidia_nim_token':
        return 'NVIDIA NGC / NIM API Key';
      default:
        return 'Enter Secret Credential';
    }
  };

  const handleSave = async (e) => {
    e.preventDefault();
    if (!value) return;
    try {
      setSaving(true);
      const res = await fetch(`/api/v1/projects/${activeProjectID}/secrets`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ type: secretType, value }),
      });
      if (res.ok) {
        setValue('');
        onSuccess && onSuccess();
        onClose();
      }
    } catch (e) {
      console.error('Error saving secret:', e);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-xs p-6">
      <div className="w-full max-w-md bg-[#222222] border border-[#333333] rounded-xl shadow-2xl overflow-hidden flex flex-col">
        <div className="flex items-center justify-between px-6 py-4 border-b border-[#333333] bg-[#1a1a1a]">
          <div className="flex items-center space-x-2 text-[#d97757]">
            <ShieldCheck className="w-5 h-5" />
            <h2 className="text-sm font-semibold text-white">{getTitle()}</h2>
          </div>
          <button onClick={onClose} className="text-gray-400 hover:text-white p-1 rounded-md hover:bg-[#333333]">
            <X className="w-5 h-5" />
          </button>
        </div>

        <form onSubmit={handleSave} className="p-6 space-y-4">
          <p className="text-xs text-gray-400">
            Credentials are sent directly to Azure Key Vault and never logged or stored in plaintext.
          </p>
          <div>
            <label className="block text-xs font-medium text-gray-300 mb-1">Secret Key / Token</label>
            <input
              type="password"
              placeholder="ghp_xxxxxxxxxxxxxxxxxxxx"
              value={value}
              onChange={(e) => setValue(e.target.value)}
              className="w-full bg-[#1a1a1a] border border-[#3a3a3a] rounded-lg px-3 py-2 text-sm text-white placeholder-gray-600 focus:outline-none focus:border-[#d97757]"
              required
              autoFocus
            />
          </div>
          <div className="flex justify-end space-x-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 bg-[#2c2c2c] hover:bg-[#383838] text-white text-xs font-medium rounded-lg"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={saving}
              className="px-4 py-2 bg-[#d97757] hover:bg-[#c66849] text-white text-xs font-medium rounded-lg transition-colors"
            >
              {saving ? 'Encrypting & Saving...' : 'Save Secret'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
