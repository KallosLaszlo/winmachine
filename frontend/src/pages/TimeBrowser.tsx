import { useEffect, useState, useMemo, useRef, useCallback } from 'react';
import {
  GetSnapshots,
  GetSnapshotFiles,
  RestoreFile,
  SelectTargetDirectory,
  MountSnapshot,
  UnmountSnapshot,
  GetMountedSnapshot,
} from '../../wailsjs/go/main/App';

interface SnapshotMeta {
  id: string;
  timestamp: string;
  fileCount: number;
  totalSize: number;
  duration: string;
}

interface FileInfo {
  name: string;
  relPath: string;
  size: number;
  modTime: number;
  isDir: boolean;
}

function fmtBytes(b: number): string {
  if (b === 0) return '0 B';
  const k = 1024, s = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(b) / Math.log(k));
  return parseFloat((b / Math.pow(k, i)).toFixed(1)) + ' ' + s[i];
}

function fmtDate(ts: string): string {
  return new Date(ts).toLocaleString();
}

function fmtShort(ts: string): string {
  const d = new Date(ts);
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
    + ' ' + d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
}

function fmtModTime(ns: number): string {
  if (!ns) return '';
  return new Date(ns / 1e6).toLocaleString();
}

function fIcon(name: string, isDir: boolean): string {
  if (isDir) return '📁';
  const ext = name.split('.').pop()?.toLowerCase() || '';
  if (['jpg','jpeg','png','gif','webp','bmp','svg'].includes(ext)) return '🖼️';
  if (['mp3','wav','flac','ogg','aac'].includes(ext)) return '🎵';
  if (['mp4','avi','mkv','mov','wmv'].includes(ext)) return '🎬';
  if (['zip','rar','7z','tar','gz'].includes(ext)) return '📦';
  if (['exe','msi'].includes(ext)) return '⚙️';
  if (['doc','docx','odt','pdf'].includes(ext)) return '📝';
  if (['xls','xlsx'].includes(ext)) return '📊';
  return '📄';
}

const VISIBLE_BEHIND = 8;  // older snapshots stacked behind active

export default function TimeBrowser() {
  const [snapshots, setSnapshots] = useState<SnapshotMeta[]>([]);
  // activeIdx: 0 = oldest, last = newest. We display newest at bottom/front.
  const [activeIdx, setActiveIdx] = useState(0);
  const [files, setFiles] = useState<FileInfo[]>([]);
  const [path, setPath] = useState('');
  const [toast, setToast] = useState<{ msg: string; type: 'success' | 'error' } | null>(null);
  const [mountedDrive, setMountedDrive] = useState('');
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [dragging, setDragging] = useState(false);
  const [pendingPath, setPendingPath] = useState<string | null>(null);
  const trackRef = useRef<HTMLDivElement>(null);
  const pageRef = useRef<HTMLDivElement>(null);

  // Snapshots are stored oldest-first. Reverse so index 0 = newest.
  const snaps = useMemo(() => [...snapshots].reverse(), [snapshots]);
  const snap = snaps[activeIdx] ?? null;

  useEffect(() => {
    GetSnapshots().then((s) => {
      setSnapshots(s || []);
      // Start at newest snapshot (highest index after reverse)
      if (s && s.length > 0) setActiveIdx(s.length - 1);
    });
    GetMountedSnapshot().then((d) => setMountedDrive(d || ''));
    return () => { UnmountSnapshot(); };
  }, []);

  useEffect(() => {
    if (snap) {
      // If we have a pending path from snapshot switch, try to use it
      const targetPath = pendingPath !== null ? pendingPath : path;
      
      GetSnapshotFiles(snap.id, targetPath || '.').then((f) => {
        setFiles(f || []);
        // Successfully loaded - if this was a pending path, commit it
        if (pendingPath !== null) {
          setPath(pendingPath);
          setPendingPath(null);
        }
      }).catch(() => {
        // Path doesn't exist in this snapshot - try fallback
        if (targetPath) {
          const parentPath = targetPath.split('/').slice(0, -1).join('/');
          // Recursively try parent paths
          const tryPath = (p: string) => {
            GetSnapshotFiles(snap.id, p || '.').then((f) => {
              setFiles(f || []);
              setPath(p);
              setPendingPath(null);
            }).catch(() => {
              if (p) {
                tryPath(p.split('/').slice(0, -1).join('/'));
              } else {
                setFiles([]);
                setPath('');
                setPendingPath(null);
              }
            });
          };
          tryPath(parentPath);
        } else {
          setFiles([]);
          setPendingPath(null);
        }
      });
    }
  }, [snap?.id, path, pendingPath]);

  const crumbs = useMemo(() => path ? path.split('/').filter(Boolean) : [], [path]);

  const showToast = (msg: string, type: 'success' | 'error') => {
    setToast({ msg, type });
    setTimeout(() => setToast(null), 3000);
  };

  const goSnap = useCallback((idx: number) => {
    const clamped = Math.max(0, Math.min(snaps.length - 1, idx));
    if (clamped === activeIdx) return;
    if (mountedDrive) UnmountSnapshot().then(() => setMountedDrive(''));
    // Store current path for validation in new snapshot
    setPendingPath(path);
    setActiveIdx(clamped);
    setSelectedFile(null);
  }, [activeIdx, snaps.length, mountedDrive, path]);

  const navigate = (f: FileInfo) => {
    if (f.isDir) { setPath(f.relPath); setSelectedFile(null); }
  };

  const goUp = () => {
    const p = crumbs.slice(0, -1).join('/');
    setPath(p);
  };

  const handleRestore = (f: FileInfo) => {
    SelectTargetDirectory().then((dest) => {
      if (dest && snap) {
        RestoreFile(snap.id, f.relPath, dest + '\\' + f.name)
          .then(() => showToast(`Restored ${f.name}`, 'success'))
          .catch((err) => showToast(`${err}`, 'error'));
      }
    });
  };

  const handleMount = () => {
    if (mountedDrive) {
      UnmountSnapshot().then(() => setMountedDrive(''));
    } else if (snap) {
      MountSnapshot(snap.id)
        .then((d) => setMountedDrive(d || ''))
        .catch((err) => showToast(`${err}`, 'error'));
    }
  };

  // --- Mouse wheel anywhere on page to change snapshot ---
  useEffect(() => {
    const page = pageRef.current;
    if (!page) return;
    const onWheel = (e: WheelEvent) => {
      // Don't intercept scroll inside the file list
      const target = e.target as HTMLElement;
      if (target.closest('.tm-filebody')) return;
      e.preventDefault();
      // Scroll up (negative deltaY) = newer = lower idx, scroll down = older = higher idx
      if (e.deltaY < 0) goSnap(activeIdx - 1);
      else if (e.deltaY > 0) goSnap(activeIdx + 1);
    };
    page.addEventListener('wheel', onWheel, { passive: false });
    return () => page.removeEventListener('wheel', onWheel);
  }, [goSnap, activeIdx]);

  // --- Drag logic for timeline thumb ---
  const idxFromY = useCallback((clientY: number) => {
    const track = trackRef.current;
    if (!track || snaps.length < 2) return 0;
    const rect = track.getBoundingClientRect();
    // Top of track = newest (idx 0), bottom = oldest (last idx)
    const ratio = Math.max(0, Math.min(1, (clientY - rect.top) / rect.height));
    return Math.round(ratio * (snaps.length - 1));
  }, [snaps.length]);

  const onThumbDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setDragging(true);
    const onMove = (ev: MouseEvent) => {
      goSnap(idxFromY(ev.clientY));
    };
    const onUp = () => {
      setDragging(false);
      window.removeEventListener('mousemove', onMove);
      window.removeEventListener('mouseup', onUp);
    };
    window.addEventListener('mousemove', onMove);
    window.addEventListener('mouseup', onUp);
  }, [goSnap, idxFromY]);

  const onTrackClick = useCallback((e: React.MouseEvent) => {
    goSnap(idxFromY(e.clientY));
  }, [goSnap, idxFromY]);

  // --- Timeline tick positions ---
  // Show up to 15 labeled ticks
  const maxTicks = 15;
  const tickStep = snaps.length <= maxTicks ? 1 : Math.ceil(snaps.length / maxTicks);
  const ticks: number[] = [];
  for (let i = 0; i < snaps.length; i += tickStep) ticks.push(i);
  if (ticks.length > 0 && ticks[ticks.length - 1] !== snaps.length - 1) ticks.push(snaps.length - 1);

  // Thumb position: 0% = top (newest, idx 0), 100% = bottom (oldest)
  const thumbPct = snaps.length > 1 ? (activeIdx / (snaps.length - 1)) * 100 : 0;

  // --- 3D window stack (macOS Time Machine card-deck style) ---
  // offset 0 = active (front), +1..N = behind (older = lower index in snaps)
  const renderWindow = (offset: number) => {
    const snapIdx = activeIdx - offset;
    if (snapIdx < 0 || snapIdx >= snaps.length) return null;
    const s = snaps[snapIdx];
    const isActive = offset === 0;

    // Each behind card shifts UP by ~28px (just enough to show its chrome/titlebar)
    // and recedes BACK with slight scale reduction — like a card deck
    const translateY = -offset * 28;
    const translateZ = -offset * 40;
    const scale = 1 - offset * 0.012;
    const opacity = isActive ? 1 : Math.max(0.15, 0.85 - offset * 0.08);
    const blur = isActive ? 0 : Math.min(offset * 0.5, 2);

    return (
      <div
        key={s.id}
        className={`tm-window ${isActive ? 'active' : 'behind'}`}
        style={{
          transform: `translate3d(0, ${translateY}px, ${translateZ}px) scale(${scale})`,
          opacity,
          zIndex: 100 - offset,
          filter: blur > 0 ? `blur(${blur}px)` : undefined,
        }}
        onClick={!isActive ? () => goSnap(snapIdx) : undefined}
      >
        {/* Chrome: toolbar + address bar */}
        <div className="tm-chrome">
          <div className="tm-toolbar">
            <div className="tm-nav-arrows">
              <button className="tm-nav-arrow" disabled={!path} onClick={isActive ? goUp : undefined}>←</button>
              <button className="tm-nav-arrow" disabled>→</button>
            </div>
            <div className="tm-addressbar">
              <button className="tm-addr-segment" onClick={isActive ? () => setPath('') : undefined}>💻</button>
              <span className="tm-addr-sep">›</span>
              <button className="tm-addr-segment" onClick={isActive ? () => setPath('') : undefined}>
                {isActive ? 'Snapshot' : fmtShort(s.timestamp)}
              </button>
              {isActive && crumbs.map((c, i) => (
                <span key={i}>
                  <span className="tm-addr-sep">›</span>
                  <button className="tm-addr-segment"
                    onClick={() => setPath(crumbs.slice(0, i + 1).join('/'))}
                  >{c}</button>
                </span>
              ))}
            </div>
            {isActive && (
              <div className="tm-toolbar-right">
                <button className={`tm-tool-btn ${mountedDrive ? 'mounted' : ''}`} onClick={handleMount}>
                  {mountedDrive ? `⏏ ${mountedDrive}` : '📂 Explorer'}
                </button>
              </div>
            )}
          </div>
        </div>

        {/* Column headers */}
        <div className="tm-colheader">
          <span>Name</span>
          <span>Date Modified</span>
          <span>Size</span>
          <span></span>
        </div>

        {/* File body */}
        <div className="tm-filebody">
          {isActive ? (
            <>
              {path && (
                <div className="tm-filerow tm-filerow-up" onClick={goUp}>
                  <div className="tm-fname"><span className="tm-ficon">⬆️</span><span>..</span></div>
                  <div className="tm-fdate"></div><div className="tm-fsize"></div><div className="tm-factions"></div>
                </div>
              )}
              {files.length > 0 ? files.map((f) => (
                <div key={f.relPath}
                  className={`tm-filerow ${selectedFile === f.relPath ? 'selected' : ''}`}
                  onClick={() => { setSelectedFile(f.relPath); navigate(f); }}
                >
                  <div className="tm-fname">
                    <span className="tm-ficon">{fIcon(f.name, f.isDir)}</span>
                    <span>{f.name}</span>
                  </div>
                  <div className="tm-fdate">{fmtModTime(f.modTime)}</div>
                  <div className="tm-fsize">{f.isDir ? '' : fmtBytes(f.size)}</div>
                  <div className="tm-factions">
                    <button className="tm-restore-btn"
                      onClick={(e) => { e.stopPropagation(); handleRestore(f); }}
                    >Restore</button>
                  </div>
                </div>
              )) : (
                <div className="tm-empty">Empty folder</div>
              )}
            </>
          ) : (
            <div className="tm-ghost-rows">
              {[...Array(5)].map((_, i) => <div key={i} className="tm-ghost-row" />)}
            </div>
          )}
        </div>

        {/* Status bar */}
        <div className="tm-statusbar">
          {isActive ? `${files.length} items` : `${s.fileCount} files · ${fmtBytes(s.totalSize)}`}
        </div>
      </div>
    );
  };

  // Build stack: render back-to-front (furthest behind first, active last)
  const stack = [];
  const behindCount = Math.min(VISIBLE_BEHIND, activeIdx);

  for (let off = behindCount; off >= 0; off--) {
    const el = renderWindow(off);
    if (el) stack.push(el);
  }

  if (snaps.length === 0) {
    return (
      <div className="tm-page">
        <div className="tm-empty-state">
          <div className="icon">🕰️</div>
          <p>No snapshots yet. Run your first backup to travel through time.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="tm-page" ref={pageRef}>
      {/* Main area */}
      <div className="tm-main">
        {/* 3D Stage */}
        <div className="tm-stage">
          {stack}
        </div>

        {/* Vertical timeline */}
        <div className="tm-timeline">
          <button className="tm-tl-arrow" onClick={() => goSnap(activeIdx - 1)}
            disabled={activeIdx <= 0}>▲</button>

          {/* Track with ticks + draggable thumb */}
          <div className="tm-tl-track" ref={trackRef} onClick={onTrackClick}>
            {/* Tick marks + labels */}
            {ticks.map((idx) => {
              const pct = snaps.length > 1 ? (idx / (snaps.length - 1)) * 100 : 50;
              return (
                <div key={idx}>
                  <div
                    className={`tm-tl-tick ${idx === activeIdx ? 'active' : ''}`}
                    style={{ top: `${pct}%` }}
                    onClick={(e) => { e.stopPropagation(); goSnap(idx); }}
                  />
                  <div
                    className={`tm-tl-label ${idx === activeIdx ? 'active' : ''}`}
                    style={{ top: `${pct}%` }}
                  >
                    {fmtShort(snaps[idx].timestamp)}
                  </div>
                </div>
              );
            })}

            {/* Draggable thumb */}
            <div
              className="tm-tl-thumb"
              style={{ top: `${thumbPct}%` }}
              onMouseDown={onThumbDown}
            />
          </div>

          <div className="tm-tl-current">
            {snap ? fmtShort(snap.timestamp) : ''}
          </div>

          <div className="tm-tl-now">Now</div>

          <button className="tm-tl-arrow" onClick={() => goSnap(activeIdx + 1)}
            disabled={activeIdx >= snaps.length - 1}>▼</button>
        </div>
      </div>

      {/* Bottom bar */}
      <div className="tm-bottombar">
        <button className="tm-btn-restore"
          disabled={!selectedFile}
          onClick={() => {
            const f = files.find(f => f.relPath === selectedFile);
            if (f) handleRestore(f);
          }}
        >
          Restore
        </button>
      </div>

      {/* Toast */}
      {toast && <div className={`tm-toast ${toast.type}`}>{toast.msg}</div>}
    </div>
  );
}
