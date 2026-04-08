import { useEffect, useState } from 'react';
import {
  GetConfig,
  SaveConfig,
  SelectDirectory,
  SelectTargetDirectory,
  IsAutoStartEnabled,
  TestSMBConnection,
  GetAvailableDrives,
  PurgeSourceDirBackups,
} from '../../wailsjs/go/main/App';

interface RetentionPolicy {
  hourlyForHours: number;
  dailyForDays: number;
  weeklyForWeeks: number;
  monthlyForMonths: number;
}

interface SMBShareConfig {
  server: string;
  share: string;
  username: string;
  password: string;
  domain: string;
  drive: string;
}

interface Config {
  sourceDirs: string[];
  targetDir: string;
  targetType: string;
  smbTarget: SMBShareConfig;
  scheduleInterval: string;
  retention: RetentionPolicy;
  autoStart: boolean;
  excludePatterns: string[];
}

const scheduleOptions = [
  { label: 'Every 15 minutes', value: '@every 15m' },
  { label: 'Every 30 minutes', value: '@every 30m' },
  { label: 'Every hour', value: '@every 1h' },
  { label: 'Every 2 hours', value: '@every 2h' },
  { label: 'Every 6 hours', value: '@every 6h' },
  { label: 'Every 12 hours', value: '@every 12h' },
  { label: 'Daily', value: '@every 24h' },
];

export default function Settings() {
  const [cfg, setCfg] = useState<Config | null>(null);
  const [saved, setSaved] = useState(false);
  const [excludeInput, setExcludeInput] = useState('');
  const [smbTesting, setSmbTesting] = useState(false);
  const [smbTestResult, setSmbTestResult] = useState<{ ok: boolean; msg: string } | null>(null);
  const [availableDrives, setAvailableDrives] = useState<string[]>([]);
  const [showPassword, setShowPassword] = useState(false);

  useEffect(() => {
    GetConfig().then((c: any) => {
      // Ensure smbTarget exists with defaults
      if (!c.smbTarget) {
        c.smbTarget = { server: '', share: '', username: '', password: '', domain: '', drive: 'Z:' };
      }
      if (!c.targetType) {
        c.targetType = 'local';
      }
      setCfg(c);
    });
    GetAvailableDrives().then(setAvailableDrives);
  }, []);

  if (!cfg) return <div>Loading...</div>;

  const save = (updated: Config) => {
    setCfg(updated);
    SaveConfig(
      updated.sourceDirs,
      updated.targetDir,
      updated.targetType,
      updated.scheduleInterval,
      updated.smbTarget,
      updated.retention,
      updated.autoStart,
      updated.excludePatterns,
    ).then(() => {
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    });
  };

  const updateSMB = (field: keyof SMBShareConfig, value: string) => {
    save({ ...cfg, smbTarget: { ...cfg.smbTarget, [field]: value } });
  };

  const testSMB = () => {
    setSmbTesting(true);
    setSmbTestResult(null);
    TestSMBConnection(cfg.smbTarget)
      .then(() => {
        setSmbTestResult({ ok: true, msg: 'Connection successful!' });
      })
      .catch((err: any) => {
        setSmbTestResult({ ok: false, msg: String(err) });
      })
      .finally(() => setSmbTesting(false));
  };

  const addSourceDir = () => {
    SelectDirectory().then((dir) => {
      if (dir && !cfg.sourceDirs.includes(dir)) {
        save({ ...cfg, sourceDirs: [...cfg.sourceDirs, dir] });
      }
    });
  };

  const removeSourceDir = (dir: string) => {
    const deleteBackups = window.confirm(
      `Remove "${dir}" from backup sources?\n\nDo you also want to delete all existing backups for this folder?`
    );
    save({ ...cfg, sourceDirs: cfg.sourceDirs.filter((d) => d !== dir) });
    if (deleteBackups) {
      PurgeSourceDirBackups(dir).catch((err: any) =>
        console.error('Failed to purge backups:', err)
      );
    }
  };

  const selectTarget = () => {
    SelectTargetDirectory().then((dir) => {
      if (dir) {
        save({ ...cfg, targetDir: dir });
      }
    });
  };

  const addExcludePattern = () => {
    const pattern = excludeInput.trim();
    if (pattern && !cfg.excludePatterns.includes(pattern)) {
      save({ ...cfg, excludePatterns: [...cfg.excludePatterns, pattern] });
      setExcludeInput('');
    }
  };

  const removeExcludePattern = (pattern: string) => {
    save({ ...cfg, excludePatterns: cfg.excludePatterns.filter((p) => p !== pattern) });
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h1 className="page-title">Settings</h1>
        {saved && <span className="badge success">Saved ✓</span>}
      </div>

      {/* Source Dirs */}
      <div className="card">
        <h3>Source Folders</h3>
        <p style={{ fontSize: 13, color: 'var(--text-secondary)', marginBottom: 12 }}>
          Select the folders you want to back up.
        </p>
        <div className="tag-list">
          {cfg.sourceDirs.map((dir) => (
            <div className="tag" key={dir}>
              📁 {dir}
              <button className="tag-remove" onClick={() => removeSourceDir(dir)}>×</button>
            </div>
          ))}
        </div>
        <button className="secondary" style={{ marginTop: 12 }} onClick={addSourceDir}>
          + Add Folder
        </button>
      </div>

      {/* Target Type */}
      <div className="card">
        <h3>Backup Target</h3>
        <p style={{ fontSize: 13, color: 'var(--text-secondary)', marginBottom: 12 }}>
          Choose where to store your backups — a local folder or a network (SMB) share.
        </p>
        <div style={{ display: 'flex', gap: 8, marginBottom: 16 }}>
          <button
            className={cfg.targetType === 'local' ? 'primary' : 'secondary'}
            onClick={() => save({ ...cfg, targetType: 'local' })}
          >
            📁 Local Folder
          </button>
          <button
            className={cfg.targetType === 'smb' ? 'primary' : 'secondary'}
            onClick={() => save({ ...cfg, targetType: 'smb' })}
          >
            🌐 SMB Network Share
          </button>
        </div>

        {cfg.targetType === 'local' && (
          <div>
            <p style={{ fontSize: 13, color: 'var(--text-secondary)', marginBottom: 8 }}>
              Must be on an NTFS volume for hard link support.
            </p>
            <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
              <input
                readOnly
                value={cfg.targetDir || 'Not set'}
                style={{ flex: 1, cursor: 'pointer', opacity: cfg.targetDir ? 1 : 0.5 }}
                onClick={selectTarget}
              />
              <button className="secondary" onClick={selectTarget}>Browse</button>
            </div>
          </div>
        )}

        {cfg.targetType === 'smb' && (
          <div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
              <div className="field">
                <label>Server (IP or hostname)</label>
                <input
                  value={cfg.smbTarget.server}
                  onChange={(e) => updateSMB('server', e.target.value)}
                  placeholder="192.168.1.100 or nas.local"
                />
              </div>
              <div className="field">
                <label>Share name</label>
                <input
                  value={cfg.smbTarget.share}
                  onChange={(e) => updateSMB('share', e.target.value)}
                  placeholder="Backups"
                />
              </div>
              <div className="field">
                <label>Domain (optional)</label>
                <input
                  value={cfg.smbTarget.domain}
                  onChange={(e) => updateSMB('domain', e.target.value)}
                  placeholder="WORKGROUP"
                />
              </div>
              <div className="field">
                <label>Mount as drive</label>
                <select
                  value={cfg.smbTarget.drive}
                  onChange={(e) => updateSMB('drive', e.target.value)}
                >
                  {/* Keep current drive if already set */}
                  {cfg.smbTarget.drive && !availableDrives.includes(cfg.smbTarget.drive) && (
                    <option value={cfg.smbTarget.drive}>{cfg.smbTarget.drive} (current)</option>
                  )}
                  {availableDrives.map((d) => (
                    <option key={d} value={d}>{d}</option>
                  ))}
                </select>
              </div>
              <div className="field">
                <label>Username</label>
                <input
                  value={cfg.smbTarget.username}
                  onChange={(e) => updateSMB('username', e.target.value)}
                  placeholder="user"
                  autoComplete="off"
                />
              </div>
              <div className="field">
                <label>Password</label>
                <div style={{ display: 'flex', gap: 4 }}>
                  <input
                    type={showPassword ? 'text' : 'password'}
                    value={cfg.smbTarget.password}
                    onChange={(e) => updateSMB('password', e.target.value)}
                    placeholder="••••••••"
                    autoComplete="off"
                    style={{ flex: 1 }}
                  />
                  <button
                    className="secondary"
                    style={{ padding: '6px 10px', fontSize: 12 }}
                    onClick={() => setShowPassword(!showPassword)}
                    title={showPassword ? 'Hide' : 'Show'}
                  >
                    {showPassword ? '🙈' : '👁'}
                  </button>
                </div>
              </div>
            </div>

            <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginTop: 8 }}>
              <p style={{ fontSize: 12, color: 'var(--text-secondary)', flex: 1 }}>
                UNC: <code style={{ color: 'var(--text-primary)' }}>\\{cfg.smbTarget.server || '...'}\{cfg.smbTarget.share || '...'}</code> → {cfg.smbTarget.drive || '?:'}
              </p>
            </div>

            <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginTop: 12 }}>
              <button className="primary" onClick={testSMB} disabled={smbTesting || !cfg.smbTarget.server || !cfg.smbTarget.share}>
                {smbTesting ? 'Testing...' : '🔌 Test Connection'}
              </button>
              {smbTestResult && (
                <span className={`badge ${smbTestResult.ok ? 'success' : 'danger'}`}>
                  {smbTestResult.msg}
                </span>
              )}
            </div>

            {/* Optional: subdirectory on the share for backups */}
            <div className="field" style={{ marginTop: 16 }}>
              <label>Subdirectory on share (optional)</label>
              <input
                value={cfg.targetDir}
                onChange={(e) => save({ ...cfg, targetDir: e.target.value })}
                placeholder={`${cfg.smbTarget.drive}\\WinMachineBackups`}
              />
              <p style={{ fontSize: 11, color: 'var(--text-secondary)', marginTop: 4 }}>
                Leave empty to use the root of the share. Path relative to the mounted drive.
              </p>
            </div>
          </div>
        )}
      </div>

      {/* Schedule */}
      <div className="card">
        <h3>Backup Schedule</h3>
        <div className="field">
          <label>Interval</label>
          <select
            value={cfg.scheduleInterval}
            onChange={(e) => save({ ...cfg, scheduleInterval: e.target.value })}
          >
            {scheduleOptions.map((opt) => (
              <option key={opt.value} value={opt.value}>{opt.label}</option>
            ))}
          </select>
        </div>
      </div>

      {/* Retention */}
      <div className="card">
        <h3>Retention Policy</h3>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <div className="field">
            <label>Keep hourly for (hours)</label>
            <input
              type="number" min={1}
              value={cfg.retention.hourlyForHours}
              onChange={(e) => save({ ...cfg, retention: { ...cfg.retention, hourlyForHours: +e.target.value } })}
            />
          </div>
          <div className="field">
            <label>Keep daily for (days)</label>
            <input
              type="number" min={1}
              value={cfg.retention.dailyForDays}
              onChange={(e) => save({ ...cfg, retention: { ...cfg.retention, dailyForDays: +e.target.value } })}
            />
          </div>
          <div className="field">
            <label>Keep weekly for (weeks)</label>
            <input
              type="number" min={1}
              value={cfg.retention.weeklyForWeeks}
              onChange={(e) => save({ ...cfg, retention: { ...cfg.retention, weeklyForWeeks: +e.target.value } })}
            />
          </div>
          <div className="field">
            <label>Keep monthly for (months)</label>
            <input
              type="number" min={1}
              value={cfg.retention.monthlyForMonths}
              onChange={(e) => save({ ...cfg, retention: { ...cfg.retention, monthlyForMonths: +e.target.value } })}
            />
          </div>
        </div>
      </div>

      {/* Exclude Patterns */}
      <div className="card">
        <h3>Exclude Patterns</h3>
        <p style={{ fontSize: 13, color: 'var(--text-secondary)', marginBottom: 12 }}>
          Glob patterns for files and folders to skip (e.g., *.tmp, node_modules).
        </p>
        <div className="tag-list">
          {cfg.excludePatterns.map((p) => (
            <div className="tag" key={p}>
              {p}
              <button className="tag-remove" onClick={() => removeExcludePattern(p)}>×</button>
            </div>
          ))}
        </div>
        <div style={{ display: 'flex', gap: 8, marginTop: 12 }}>
          <input
            value={excludeInput}
            onChange={(e) => setExcludeInput(e.target.value)}
            placeholder="e.g., *.log"
            onKeyDown={(e) => e.key === 'Enter' && addExcludePattern()}
          />
          <button className="secondary" onClick={addExcludePattern}>Add</button>
        </div>
      </div>

      {/* Auto-start */}
      <div className="card">
        <h3>Startup</h3>
        <p style={{ fontSize: 12, color: 'var(--text-secondary)', marginBottom: 12, background: 'rgba(255,180,0,0.1)', border: '1px solid rgba(255,180,0,0.3)', borderRadius: 6, padding: '8px 12px' }}>
          {/* TODO: Future feature — save the current exe path in the config file and on next startup, if it has changed, automatically update the Windows registry entry */}
          ⚠️ This is a portable application. If you move the .exe file to a different location, you must disable and re-enable this option to update the registered path in Windows.
        </p>
        <label style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer' }}>
          <input
            type="checkbox"
            checked={cfg.autoStart}
            onChange={(e) => save({ ...cfg, autoStart: e.target.checked })}
            style={{ width: 'auto', accentColor: 'var(--accent)' }}
          />
          Start WinMachine automatically when Windows starts
        </label>
      </div>
    </div>
  );
}
