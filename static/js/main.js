document.addEventListener('DOMContentLoaded', () => {
  // 加载用户统计数据
  fetch('/api/users')
    .then(res => res.json())
    .then(data => {
      const tbody = document.getElementById('userList');
      tbody.innerHTML = data.map(user => `
        <tr onclick="loadPartitions(${user.id})">
          <td>${user.name}</td>
          <td>${user.total_files}</td>
          <td>${formatSize(user.total_size)}</td>
          <td>${user.partitions.join(', ')}</td>
        </tr>
      `).join('');
    });

  // 数据库配置表单提交
  document.getElementById('dbConfigForm').addEventListener('submit', e => {
    e.preventDefault();
    const formData = new FormData(e.target);
    fetch('/api/configure-db', {
      method: 'POST',
      body: JSON.stringify(Object.fromEntries(formData)),
      headers: {'Content-Type': 'application/json'}
    }).then(handleResponse);
  });
});

function loadPartitions(userId) {
  fetch(`/api/partitions/${userId}`)
    .then(res => res.json())
    .then(renderPartitionGrid);
}

function renderPartitionGrid(partitions) {
  const container = document.getElementById('partitionDetails');
  container.innerHTML = `<div class="partition-grid">
    ${Array(256).fill().map((_,i) => {
      const p = partitions.find(v => v.partition === i.toString(16).padStart(2,'0'));
      return `<div class="partition-cell bg-${p ? 'success' : 'secondary'}">${i.toString(16).padStart(2,'0')}</div>`;
    }).join('')}
  </div>`;
}

function formatSize(bytes) {
  const units = ['B','KB','MB','GB'];
  let size = bytes, unit = 0;
  while (size >= 1024 && unit < 3) { size /= 1024; unit++; }
  return `${size.toFixed(2)} ${units[unit]}`;
}