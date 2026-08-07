import React, { useState } from 'react';
import { Copy, Check, Code, ChevronRight } from 'lucide-react';

export default function MarkdownRenderer({ content }) {
  if (!content) return null;

  // Split text by fenced code blocks: ```lang ... ```
  const codeBlockRegex = /```([a-zA-Z0-9_\-]*)\n([\s+S]*?)```/g;
  const parts = [];
  let lastIndex = 0;
  let match;

  while ((match = codeBlockRegex.exec(content)) !== null) {
    // Text before code block
    if (match.index > lastIndex) {
      parts.push({
        type: 'text',
        value: content.slice(lastIndex, match.index),
      });
    }

    parts.push({
      type: 'code',
      language: match[1] || 'code',
      value: match[2].trim(),
    });

    lastIndex = codeBlockRegex.lastIndex;
  }

  if (lastIndex < content.length) {
    parts.push({
      type: 'text',
      value: content.slice(lastIndex),
    });
  }

  return (
    <div className="space-y-3 font-sans text-sm text-gray-200 leading-relaxed">
      {parts.map((part, idx) => {
        if (part.type === 'code') {
          return (
            <CodeBlock key={idx} language={part.language} code={part.value} />
          );
        }

        return (
          <div key={idx} className="space-y-2">
            {parseInlineMarkdown(part.value)}
          </div>
        );
      })}
    </div>
  );
}

function CodeBlock({ language, code }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="my-3 rounded-xl border border-[#333644] bg-[#16181f] overflow-hidden shadow-lg">
      {/* Code Block Header */}
      <div className="flex items-center justify-between px-3.5 py-2 bg-[#20232e] border-b border-[#2d303f]">
        <div className="flex items-center space-x-2">
          <Code className="w-3.5 h-3.5 text-[#d97757]" />
          <span className="text-[11px] font-mono font-semibold uppercase text-gray-300 tracking-wider">
            {language}
          </span>
        </div>
        <button
          onClick={handleCopy}
          className="flex items-center space-x-1 text-[11px] text-gray-400 hover:text-white px-2 py-0.5 rounded bg-[#2a2d3b] hover:bg-[#343849] transition-colors"
        >
          {copied ? (
            <>
              <Check className="w-3 h-3 text-emerald-400" />
              <span className="text-emerald-400 font-medium">Copied!</span>
            </>
          ) : (
            <>
              <Copy className="w-3 h-3 text-gray-400" />
              <span>Copy code</span>
            </>
          )}
        </button>
      </div>

      {/* Code Content */}
      <pre className="p-4 text-xs font-mono text-emerald-300 overflow-x-auto leading-relaxed select-text">
        <code>{code}</code>
      </pre>
    </div>
  );
}

function parseInlineMarkdown(text) {
  if (!text) return null;

  const lines = text.split('\n');
  return lines.map((line, lIdx) => {
    const trimmed = line.trim();

    if (!trimmed) {
      return <div key={lIdx} className="h-1.5" />;
    }

    // Headers: #, ##, ###
    if (trimmed.startsWith('# ')) {
      return (
        <h1 key={lIdx} className="text-xl font-bold text-white tracking-tight mt-3 mb-1 flex items-center space-x-2">
          <span className="text-[#d97757]">#</span>
          <span>{renderFormattedInline(trimmed.slice(2))}</span>
        </h1>
      );
    }
    if (trimmed.startsWith('## ')) {
      return (
        <h2 key={lIdx} className="text-lg font-bold text-white tracking-tight mt-2.5 mb-1 flex items-center space-x-2">
          <span className="text-[#d97757]">##</span>
          <span>{renderFormattedInline(trimmed.slice(3))}</span>
        </h2>
      );
    }
    if (trimmed.startsWith('### ')) {
      return (
        <h3 key={lIdx} className="text-base font-semibold text-gray-100 mt-2 mb-1 flex items-center space-x-2">
          <span className="text-[#d97757]">###</span>
          <span>{renderFormattedInline(trimmed.slice(4))}</span>
        </h3>
      );
    }

    // Lists: - item or * item
    if (trimmed.startsWith('- ') || trimmed.startsWith('* ')) {
      return (
        <div key={lIdx} className="flex items-start space-x-2 ml-2 my-1">
          <span className="text-[#d97757] font-bold text-sm shrink-0 mt-0.5">•</span>
          <span className="text-gray-200">{renderFormattedInline(trimmed.slice(2))}</span>
        </div>
      );
    }

    // Numbered lists: 1. item
    const numMatch = trimmed.match(/^(\d+)\.\s+(.*)/);
    if (numMatch) {
      return (
        <div key={lIdx} className="flex items-start space-x-2 ml-2 my-1">
          <span className="text-[#d97757] font-mono text-xs font-bold shrink-0 mt-0.5">{numMatch[1]}.</span>
          <span className="text-gray-200">{renderFormattedInline(numMatch[2])}</span>
        </div>
      );
    }

    // Blockquotes: > quote
    if (trimmed.startsWith('> ')) {
      return (
        <blockquote key={lIdx} className="border-l-2 border-[#d97757] pl-3 py-1.5 italic text-gray-300 bg-[#252834] rounded-r-lg my-1.5">
          {renderFormattedInline(trimmed.slice(2))}
        </blockquote>
      );
    }

    // Regular line
    return (
      <p key={lIdx} className="text-gray-200 my-0.5">
        {renderFormattedInline(line)}
      </p>
    );
  });
}

function renderFormattedInline(str) {
  if (!str) return '';

  // Regex for inline elements: `code`, **bold**, *italic*
  const tokens = [];
  let remaining = str;

  while (remaining.length > 0) {
    // Inline code: `code`
    const codeMatch = remaining.match(/^`([^`]+)`/);
    if (codeMatch) {
      tokens.push(
        <code key={tokens.length} className="font-mono text-xs text-amber-300 bg-[#2a2d39] px-1.5 py-0.5 rounded border border-[#383c4c]">
          {codeMatch[1]}
        </code>
      );
      remaining = remaining.slice(codeMatch[0].length);
      continue;
    }

    // Bold text: **text**
    const boldMatch = remaining.match(/^\*\*([^*]+)\*\*/);
    if (boldMatch) {
      tokens.push(
        <strong key={tokens.length} className="font-bold text-white">
          {boldMatch[1]}
        </strong>
      );
      remaining = remaining.slice(boldMatch[0].length);
      continue;
    }

    // Italic text: *text*
    const italicMatch = remaining.match(/^\*([^*]+)\*/);
    if (italicMatch) {
      tokens.push(
        <em key={tokens.length} className="italic text-gray-300">
          {italicMatch[1]}
        </em>
      );
      remaining = remaining.slice(italicMatch[0].length);
      continue;
    }

    // Regular character chunk up to next special character
    const nextSpecial = remaining.search(/[`*]/);
    if (nextSpecial === -1) {
      tokens.push(remaining);
      break;
    } else if (nextSpecial > 0) {
      tokens.push(remaining.slice(0, nextSpecial));
      remaining = remaining.slice(nextSpecial);
    } else {
      // Unmatched single special char
      tokens.push(remaining[0]);
      remaining = remaining.slice(1);
    }
  }

  return tokens;
}
