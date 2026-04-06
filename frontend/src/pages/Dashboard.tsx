import { useEffect, useState } from 'react';
import {
  GetBackupStatus,
  GetSnapshots,
  RunBackupNow,
  GetDiskInfo,
  GetConfig,
  GetNextBackup,
} from '../../wailsjs/go/main/App';

interface BackupStatus {
  running: boolean;
  lastSnapshot: string;
  lastTime: string;
  progress: number;
  currentFile: string;
  filesTotal: number;
  filesDone: number;
  error: string;
}

interface SnapshotMeta {
  id: string;
  timestamp: string;
  fileCount: number;
  totalSize: number;
  linkedSize: number;
  copiedSize: number;
  duration: string;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

function formatDate(ts: string): string {
  if (!ts) return 'Never';
  const d = new Date(ts);
  return d.toLocaleDateString() + ' ' + d.toLocaleTimeString();
}

export default function Dashboard() {
  const [status, setStatus] = useState<BackupStatus | null>(null);
  const [snapshots, setSnapshots] = useState<SnapshotMeta[]>([]);
  const [diskInfo, setDiskInfo] = useState<{ totalBytes: number; freeBytes: number; usedBytes: number } | null>(null);
  const [configured, setConfigured] = useState(false);
  const [nextBackup, setNextBackup] = useState<string>('');

  const refresh = () => {
    GetBackupStatus().then(setStatus);
    GetSnapshots().then((s) => setSnapshots(s || []));
    GetNextBackup().then((t) => setNextBackup(t || ''));
    GetConfig().then((cfg: any) => {
      const isSmbConfigured = cfg.targetType === 'smb' && cfg.smbTarget?.server && cfg.smbTarget?.share && cfg.smbTarget?.drive;
      const isLocalConfigured = cfg.targetType !== 'smb' && !!cfg.targetDir;
      setConfigured(isSmbConfigured || isLocalConfigured);

      const diskPath = cfg.targetType === 'smb'
        ? (cfg.targetDir || cfg.smbTarget?.drive + '\\')
        : cfg.targetDir;
      if (diskPath) {
        GetDiskInfo(diskPath).then((d: any) => setDiskInfo(d)).catch(() => {});
      }
    });
  };

  useEffect(() => {
    refresh();
    const interval = setInterval(refresh, 2000);
    return () => clearInterval(interval);
  }, []);

  const handleBackupNow = () => {
    RunBackupNow().then(refresh);
  };

  const latestSnapshot = snapshots.length > 0 ? snapshots[0] : null;

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h1 className="page-title">Dashboard</h1>
        <button className="primary" onClick={handleBackupNow} disabled={status?.running || !configured}>
          {status?.running ? 'Backing up...' : 'Back Up Now'}
        </button>
      </div>

      {!configured && (
        <div className="card" style={{ borderColor: 'var(--warning)' }}>
          <p style={{ color: 'var(--warning)' }}>
            ⚠ No target directory configured. Go to <strong>Settings</strong> to set up your backup.
          </p>
        </div>
      )}

      {status?.running && (
        <div className="card">
          <h3>Backup in Progress</h3>
          <p style={{ fontSize: '13px', color: 'var(--text-secondary)', marginBottom: 8 }}>
            {status.currentFile || 'Preparing...'}
          </p>
          <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, marginBottom: 4 }}>
            <span>{status.filesDone} / {status.filesTotal} files</span>
            <span>{Math.round(status.progress * 100)}%</span>
          </div>
          <div className="progress-bar">
            <div className="progress-fill" style={{ width: `${status.progress * 100}%` }} />
          </div>
        </div>
      )}

      {status?.error && (
        <div className="card" style={{ borderColor: 'var(--danger)' }}>
          <p style={{ color: 'var(--danger)' }}>❌ {status.error}</p>
        </div>
      )}

      <div className="stat-grid">
        <div className="stat-card">
          <div className="stat-label">Total Snapshots</div>
          <div className="stat-value">{snapshots.length}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Last Backup</div>
          <div className="stat-value" style={{ fontSize: 16 }}>
            {latestSnapshot ? formatDate(latestSnapshot.timestamp) : 'Never'}
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Next Backup</div>
          <div className="stat-value" style={{ fontSize: 16 }}>
            {nextBackup ? formatDate(nextBackup) : '—'}
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Files in Last Snapshot</div>
          <div className="stat-value">{latestSnapshot?.fileCount ?? '—'}</div>
        </div>
        {diskInfo && (
          <div className="stat-card">
            <div className="stat-label">Target Disk Space</div>
            <div className="stat-value" style={{ fontSize: 16 }}>
              {formatBytes(diskInfo.freeBytes)} free / {formatBytes(diskInfo.totalBytes)}
            </div>
            <div className="progress-bar" style={{ marginTop: 8 }}>
              <div
                className="progress-fill"
                style={{
                  width: `${((diskInfo.totalBytes - diskInfo.freeBytes) / diskInfo.totalBytes) * 100}%`,
                  background: diskInfo.freeBytes / diskInfo.totalBytes < 0.1 ? 'var(--danger)' : 'var(--accent)',
                }}
              />
            </div>
          </div>
        )}
      </div>

      {snapshots.length > 0 && (
        <div className="card">
          <h3>Recent Snapshots</h3>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
            <thead>
              <tr style={{ textAlign: 'left', color: 'var(--text-secondary)', borderBottom: '1px solid var(--border)' }}>
                <th style={{ padding: '8px 12px' }}>Date</th>
                <th style={{ padding: '8px 12px' }}>Files</th>
                <th style={{ padding: '8px 12px' }}>Size</th>
                <th style={{ padding: '8px 12px' }}>Linked</th>
                <th style={{ padding: '8px 12px' }}>Duration</th>
              </tr>
            </thead>
            <tbody>
              {snapshots.slice(0, 10).map((s) => (
                <tr key={s.id} style={{ borderBottom: '1px solid var(--border)' }}>
                  <td style={{ padding: '8px 12px' }}>{formatDate(s.timestamp)}</td>
                  <td style={{ padding: '8px 12px' }}>{s.fileCount}</td>
                  <td style={{ padding: '8px 12px' }}>{formatBytes(s.totalSize)}</td>
                  <td style={{ padding: '8px 12px' }}>{formatBytes(s.linkedSize)}</td>
                  <td style={{ padding: '8px 12px' }}>{s.duration}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
