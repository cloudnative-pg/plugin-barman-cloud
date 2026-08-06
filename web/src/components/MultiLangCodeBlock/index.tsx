import React, {ReactElement, useState} from 'react';
import CodeBlock from '@theme/CodeBlock';

export type Snippet = {
  label: string;      // visible tab label (e.g. "Shell", "Helm", "Go")
  language: string;   // language prop for CodeBlock (e.g. "sh", "yaml", "go")
  code: string;       // the snippet to display
};

type Props = {
  snippets: Snippet[];
  defaultIndex?: number;
  className?: string;
};

export function MultiLangCodeBlock({snippets, defaultIndex = 0, className}: Props): ReactElement | null {
  const safeDefault = Math.max(0, Math.min(defaultIndex, snippets.length - 1));
  const [active, setActive] = useState<number>(snippets.length > 0 ? safeDefault : -1);

  if (snippets.length === 0) return null;

  const tabStyle: React.CSSProperties = {
    padding: '0.3rem 0.5rem',
    cursor: 'pointer',
    border: '0',
    borderRadius: 'var(--ifm-code-border-radius)',
    margin: '0 0 0 0',
    fontSize: '0.8125rem',
    color: 'inherit',
  };

  const activeTabStyle: React.CSSProperties = {
    ...tabStyle,
    fontWeight: 700,
  };

  const tabsContainerStyle: React.CSSProperties = {
    display: 'flex',
    alignItems: 'flex-end',
    justifyContent: 'flex-end',
    gap: '0.25rem',
    margin: '0 0 -0.5rem',
    padding: '0.5rem 0.5rem 1rem',
    flexWrap: 'wrap',
  };

  const wrapperStyle: React.CSSProperties = {
    overflow: 'hidden',
    padding: '0',
  };

  return (
    <div className={className} style={wrapperStyle}>
      <div role="tablist" aria-label="Code snippets" style={tabsContainerStyle}>
        {snippets.map((ex, idx) => (
          <button
            key={ex.label + idx}
            role="tab"
            aria-selected={active === idx}
            aria-controls={`code-panel-${idx}`}
            id={`code-tab-${idx}`}
            onClick={() => setActive(idx)}
            style={active === idx ? activeTabStyle : tabStyle}
          >
            {ex.label}
          </button>
        ))}
      </div>

      <div role="tabpanel" id={`code-panel-${active}`} aria-labelledby={`code-tab-${active}`}>
        <CodeBlock language={snippets[active].language}>
{snippets[active].code}
        </CodeBlock>
      </div>
    </div>
  );
}

export default MultiLangCodeBlock;
