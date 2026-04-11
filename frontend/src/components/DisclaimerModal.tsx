import { useEffect, useState } from 'react';
import { AcceptDisclaimer, DeclineDisclaimer, GetDisclaimerText } from '../../wailsjs/go/main/App';

interface Props {
  onAccepted: () => void;
}

// Render a small subset of Markdown: headings, bold, bullet lists, paragraphs.
function renderMarkdown(md: string): React.ReactNode[] {
  const lines = md.split('\n');
  const nodes: React.ReactNode[] = [];
  let listItems: React.ReactNode[] = [];
  let key = 0;

  const flushList = () => {
    if (listItems.length > 0) {
      nodes.push(<ul key={key++}>{listItems}</ul>);
      listItems = [];
    }
  };

  // Inline: replace **text** with <strong>
  const inline = (text: string): React.ReactNode => {
    const parts = text.split(/\*\*(.*?)\*\*/g);
    return parts.map((p, i) => i % 2 === 1 ? <strong key={i}>{p}</strong> : p);
  };

  for (const line of lines) {
    if (/^# /.test(line)) {
      flushList();
      nodes.push(<h1 key={key++}>{inline(line.slice(2))}</h1>);
    } else if (/^## /.test(line)) {
      flushList();
      nodes.push(<h2 key={key++}>{inline(line.slice(3))}</h2>);
    } else if (/^- /.test(line)) {
      listItems.push(<li key={key++}>{inline(line.slice(2))}</li>);
    } else if (line.trim() === '') {
      flushList();
      nodes.push(<br key={key++} />);
    } else {
      flushList();
      nodes.push(<p key={key++}>{inline(line)}</p>);
    }
  }
  flushList();
  return nodes;
}

export default function DisclaimerModal({ onAccepted }: Props) {
  const [text, setText] = useState('Loading...');
  const [accepting, setAccepting] = useState(false);

  useEffect(() => {
    GetDisclaimerText().then((t) => setText(t));
  }, []);

  const handleAccept = async () => {
    setAccepting(true);
    await AcceptDisclaimer();
    onAccepted();
  };

  const handleDecline = () => {
    DeclineDisclaimer();
  };

  return (
    <div className="disclaimer-overlay">
      <div className="disclaimer-dialog">
        <div className="disclaimer-header">
          <span className="disclaimer-icon">⚠️</span>
          <h2>Disclaimer</h2>
        </div>
        <div className="disclaimer-body">
          <div className="disclaimer-text">{renderMarkdown(text)}</div>
        </div>
        <div className="disclaimer-footer">
          <button className="disclaimer-btn decline" onClick={handleDecline}>
            Decline
          </button>
          <button className="disclaimer-btn accept" onClick={handleAccept} disabled={accepting}>
            {accepting ? 'Please wait…' : 'Accept'}
          </button>
        </div>
      </div>
    </div>
  );
}
