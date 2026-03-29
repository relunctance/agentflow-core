<?php
/**
 * AgentFlow Monitor - PHP 监控页面
 * 读取 SQLite 数据库，展示 Agent 状态与任务进度
 *
 * 使用方式：
 *   php -S 0.0.0.0:8081 internal/monitor/monitor.php
 * 或部署到任意 PHP 环境
 */

define('DB_PATH', __DIR__ . '/../../data/agentflow.db');
define('REFRESH_INTERVAL', 10); // 秒

// ---------- 工具函数 ----------

function get_db(): PDO {
    $dsn = 'sqlite:' . DB_PATH;
    $pdo = new PDO($dsn);
    $pdo->setAttribute(PDO::ATTR_ERRMODE, PDO::ERRMODE_EXCEPTION);
    $pdo->setAttribute(PDO::ATTR_DEFAULT_FETCH_MODE, PDO::FETCH_ASSOC);
    return $pdo;
}

function h($s): string {
    return htmlspecialchars((string)$s, ENT_QUOTES, 'UTF-8');
}

function time_ago(string $t): string {
    $diff = time() - strtotime($t);
    if ($diff < 60)  return "{$diff}s ago";
    if ($diff < 3600) return floor($diff/60) . "m ago";
    if ($diff < 86400) return floor($diff/3600) . "h ago";
    return floor($diff/86400) . "d ago";
}

function status_badge(string $status): string {
    $cls = match ($status) {
        'active'  => 'badge-green',
        'online'  => 'badge-green',
        'idle'    => 'badge-yellow',
        'offline' => 'badge-red',
        'pending' => 'badge-yellow',
        'running' => 'badge-blue',
        'done', 'completed' => 'badge-green',
        'failed'  => 'badge-red',
        default   => 'badge-gray',
    };
    return "<span class=\"badge {$cls}\">" . h($status) . "</span>";
}

function task_status_cls(string $status): string {
    return match ($status) {
        'running' => 'row-running',
        'pending' => 'row-pending',
        'done', 'completed' => 'row-done',
        'failed'  => 'row-failed',
        default   => '',
    };
}

// ---------- 数据获取 ----------

function get_agents(PDO $db): array {
    $stmt = $db->query("SELECT * FROM agents ORDER BY updated_at DESC");
    return $stmt->fetchAll();
}

function get_tasks(PDO $db, ?string $agentId = null): array {
    if ($agentId) {
        $stmt = $db->prepare("SELECT * FROM tasks WHERE agent_id = ? ORDER BY created_at DESC LIMIT 50");
        $stmt->execute([$agentId]);
    } else {
        $stmt = $db->query("SELECT * FROM tasks ORDER BY created_at DESC LIMIT 100");
    }
    return $stmt->fetchAll();
}

function get_recent_events(PDO $db, int $limit = 30): array {
    $stmt = $db->query("SELECT * FROM events ORDER BY created_at DESC LIMIT {$limit}");
    return $stmt->fetchAll();
}

function get_stats(PDO $db): array {
    $agents  = $db->query("SELECT COUNT(*) FROM agents")->fetchColumn();
    $tasks   = $db->query("SELECT COUNT(*) FROM tasks")->fetchColumn();
    $running = $db->query("SELECT COUNT(*) FROM tasks WHERE status = 'running'")->fetchColumn();
    $done    = $db->query("SELECT COUNT(*) FROM tasks WHERE status IN ('done','completed')")->fetchColumn();
    $failed  = $db->query("SELECT COUNT(*) FROM tasks WHERE status = 'failed'")->fetchColumn();
    $events  = $db->query("SELECT COUNT(*) FROM events")->fetchColumn();
    return compact('agents','tasks','running','done','failed','events');
}

// ---------- 视图 ----------

function render_page(): void {
    $autoRefresh = isset($_GET['refresh']);

    try {
        $db = get_db();
        $agents = get_agents($db);
        $tasks  = get_tasks($db);
        $events = get_recent_events($db);
        $stats  = get_stats($db);
    } catch (PDOException $e) {
        echo "<div class='error'>数据库连接失败: " . h($e->getMessage()) . "</div>";
        return;
    }
?>
<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>AgentFlow Monitor</title>
  <style>
    :root {
      --bg: #0f1117;
      --card: #1a1d27;
      --card2: #21253a;
      --border: #2a2d3e;
      --text: #e2e4ea;
      --muted: #8b8fa8;
      --green: #3dd68c;
      --blue: #4a9eff;
      --yellow: #ffc542;
      --red: #ff5f5f;
      --gray: #5c6080;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background: var(--bg);
      color: var(--text);
      font-family: -apple-system, 'PingFang SC', 'Microsoft YaHei', sans-serif;
      font-size: 14px;
      padding: 24px;
    }
    a { color: var(--blue); text-decoration: none; }
    a:hover { text-decoration: underline; }

    .header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 24px;
    }
    .header h1 { font-size: 22px; font-weight: 600; color: var(--text); }
    .header .actions { display: flex; gap: 10px; align-items: center; }
    .header .time { color: var(--muted); font-size: 12px; }

    .btn {
      display: inline-block;
      padding: 6px 14px;
      border-radius: 6px;
      font-size: 13px;
      cursor: pointer;
      border: 1px solid var(--border);
      background: var(--card2);
      color: var(--text);
      transition: background 0.2s;
    }
    .btn:hover { background: var(--border); }
    .btn-primary { background: var(--blue); border-color: var(--blue); color: #fff; }
    .btn-primary:hover { background: #3a8ee5; }

    /* Stats */
    .stats {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
      gap: 12px;
      margin-bottom: 28px;
    }
    .stat-card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 10px;
      padding: 16px 20px;
    }
    .stat-card .label { font-size: 12px; color: var(--muted); margin-bottom: 6px; }
    .stat-card .value { font-size: 26px; font-weight: 700; }
    .stat-card .value.green { color: var(--green); }
    .stat-card .value.blue  { color: var(--blue); }
    .stat-card .value.yellow{ color: var(--yellow); }
    .stat-card .value.red   { color: var(--red); }

    /* Sections */
    .section { margin-bottom: 36px; }
    .section-title {
      font-size: 15px;
      font-weight: 600;
      color: var(--text);
      margin-bottom: 12px;
      display: flex;
      align-items: center;
      gap: 8px;
    }
    .section-title span { color: var(--muted); font-weight: 400; font-size: 13px; }

    /* Agent cards */
    .agent-grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
      gap: 14px;
    }
    .agent-card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 10px;
      padding: 16px;
    }
    .agent-card .agent-name { font-size: 15px; font-weight: 600; margin-bottom: 6px; }
    .agent-card .agent-meta { font-size: 12px; color: var(--muted); line-height: 1.8; }
    .agent-card .agent-meta strong { color: var(--text); }

    /* Tables */
    table { width: 100%; border-collapse: collapse; }
    th {
      text-align: left;
      padding: 8px 12px;
      font-size: 11px;
      font-weight: 600;
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: 0.05em;
      border-bottom: 1px solid var(--border);
    }
    td { padding: 10px 12px; border-bottom: 1px solid var(--border); font-size: 13px; }
    tr:last-child td { border-bottom: none; }
    tr:hover td { background: var(--card2); }
    .truncate { max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

    /* Badges */
    .badge {
      display: inline-block;
      padding: 2px 8px;
      border-radius: 4px;
      font-size: 11px;
      font-weight: 600;
    }
    .badge-green  { background: rgba(61,214,140,0.15); color: var(--green); }
    .badge-blue   { background: rgba(74,158,255,0.15); color: var(--blue); }
    .badge-yellow { background: rgba(255,197,66,0.15); color: var(--yellow); }
    .badge-red    { background: rgba(255,95,95,0.15); color: var(--red); }
    .badge-gray   { background: rgba(139,143,168,0.15); color: var(--gray); }

    .card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 10px;
      overflow: hidden;
    }
    .error { background: rgba(255,95,95,0.1); border: 1px solid var(--red); border-radius: 8px; padding: 16px; color: var(--red); margin: 20px 0; }

    /* Tabs */
    .tabs { display: flex; gap: 2px; margin-bottom: 0; }
    .tab {
      padding: 8px 18px;
      cursor: pointer;
      border-radius: 8px 8px 0 0;
      font-size: 13px;
      color: var(--muted);
      border: 1px solid transparent;
      transition: all 0.2s;
    }
    .tab.active { background: var(--card); color: var(--text); border-color: var(--border); border-bottom-color: var(--card); }
    .tab:hover:not(.active) { color: var(--text); }

    .tab-content { display: none; }
    .tab-content.active { display: block; }

    .mono { font-family: 'Fira Code', 'Cascadia Code', monospace; font-size: 12px; }
    .text-muted { color: var(--muted); }
  </style>
</head>
<body>

<div class="header">
  <h1>🕹 AgentFlow Monitor</h1>
  <div class="actions">
    <span class="time" id="clock"></span>
    <button class="btn btn-primary" onclick="location.reload()">🔄 刷新</button>
  </div>
</div>

<?php if (isset($error)): ?>
  <div class="error"><?= h($error) ?></div>
<?php else: ?>

<!-- Stats -->
<div class="stats">
  <div class="stat-card">
    <div class="label">Agent 总数</div>
    <div class="value blue"><?= $stats['agents'] ?></div>
  </div>
  <div class="stat-card">
    <div class="label">任务总数</div>
    <div class="value"><?= $stats['tasks'] ?></div>
  </div>
  <div class="stat-card">
    <div class="label">运行中</div>
    <div class="value yellow"><?= $stats['running'] ?></div>
  </div>
  <div class="stat-card">
    <div class="label">已完成</div>
    <div class="value green"><?= $stats['done'] ?></div>
  </div>
  <div class="stat-card">
    <div class="label">失败</div>
    <div class="value red"><?= $stats['failed'] ?></div>
  </div>
  <div class="stat-card">
    <div class="label">事件日志</div>
    <div class="value"><?= $stats['events'] ?></div>
  </div>
</div>

<!-- Tabs -->
<div class="tabs">
  <div class="tab active" onclick="switchTab('agents')">🤖 Agent 状态</div>
  <div class="tab" onclick="switchTab('tasks')">📋 任务列表</div>
  <div class="tab" onclick="switchTab('events')">📜 事件日志</div>
</div>

<div id="tab-agents" class="tab-content active">
  <div class="section">
    <?php if (empty($agents)): ?>
      <div class="card" style="padding:24px;text-align:center;color:var(--muted)">暂无 Agent 数据</div>
    <?php else: ?>
    <div class="agent-grid">
      <?php foreach ($agents as $a): ?>
      <div class="agent-card">
        <div class="agent-name">
          <?= h($a['name']) ?>
          <?= status_badge($a['status']) ?>
        </div>
        <div class="agent-meta">
          <div><strong>ID:</strong> <span class="mono"><?= h(substr($a['id'], 0, 8)) ?>...</span></div>
          <div><strong>Model:</strong> <?= h($a['model']) ?></div>
          <div><strong>Provider:</strong> <?= h($a['provider']) ?></div>
          <div><strong>更新于:</strong> <?= time_ago($a['updated_at']) ?></div>
        </div>
      </div>
      <?php endforeach; ?>
    </div>
    <?php endif; ?>
  </div>
</div>

<div id="tab-tasks" class="tab-content">
  <div class="section">
    <?php if (empty($tasks)): ?>
      <div class="card" style="padding:24px;text-align:center;color:var(--muted)">暂无任务数据</div>
    <?php else: ?>
    <div class="card">
      <table>
        <thead>
          <tr>
            <th>任务</th>
            <th>Agent</th>
            <th>状态</th>
            <th>优先级</th>
            <th>创建时间</th>
            <th>更新时间</th>
          </tr>
        </thead>
        <tbody>
          <?php foreach ($tasks as $t): ?>
          <tr>
            <td class="truncate" title="<?= h($t['title']) ?>"><?= h($t['title']) ?></td>
            <td><span class="mono"><?= h(substr($t['agent_id'], 0, 8)) ?>...</span></td>
            <td><?= status_badge($t['status']) ?></td>
            <td><?= (int)$t['priority'] ?></td>
            <td class="text-muted"><?= time_ago($t['created_at']) ?></td>
            <td class="text-muted"><?= time_ago($t['updated_at']) ?></td>
          </tr>
          <?php endforeach; ?>
        </tbody>
      </table>
    </div>
    <?php endif; ?>
  </div>
</div>

<div id="tab-events" class="tab-content">
  <div class="section">
    <?php if (empty($events)): ?>
      <div class="card" style="padding:24px;text-align:center;color:var(--muted)">暂无事件数据</div>
    <?php else: ?>
    <div class="card">
      <table>
        <thead>
          <tr>
            <th>类型</th>
            <th>Agent</th>
            <th>任务</th>
            <th>消息</th>
            <th>时间</th>
          </tr>
        </thead>
        <tbody>
          <?php foreach ($events as $e): ?>
          <tr>
            <td><?= status_badge($e['type']) ?></td>
            <td><span class="mono"><?= h(substr($e['agent_id'], 0, 8)) ?>...</span></td>
            <td><span class="mono text-muted"><?= h(substr($e['task_id'], 0, 8)) ?>...</span></td>
            <td class="truncate" title="<?= h($e['message']) ?>"><?= h($e['message']) ?></td>
            <td class="text-muted"><?= time_ago($e['created_at']) ?></td>
          </tr>
          <?php endforeach; ?>
        </tbody>
      </table>
    </div>
    <?php endif; ?>
  </div>
</div>

<?php endif; ?>

<script>
  function switchTab(name) {
    document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
    document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
    document.querySelector('.tab:nth-child(' + ['agents','tasks','events'].indexOf(name) + 1 + ')').classList.add('active');
    document.getElementById('tab-' + name).classList.add('active');
  }
  function updateClock() {
    const el = document.getElementById('clock');
    if (el) {
      const now = new Date();
      el.textContent = now.toLocaleTimeString('zh-CN', {hour:'2-digit',minute:'2-digit',second:'2-digit'});
    }
  }
  updateClock();
  setInterval(updateClock, 1000);
</script>

</body>
</html>
<?php
}

// 自动刷新
if (isset($_GET['refresh'])) {
    header("Refresh: " . REFRESH_INTERVAL);
}
render_page();
