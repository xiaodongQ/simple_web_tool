// 数据库配置表单提交
document.getElementById('dbConfigForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const formData = new FormData(e.target);
    const config = {
        host: formData.get('host'),
        port: formData.get('port'),
        user: formData.get('user'),
        password: formData.get('password'),
        dbname: 'test'
    };

    try {
        const response = await fetch('/api/configure-db', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(config)
        });
        const data = await response.json();
        alert(data.message || data.error);
    } catch (error) {
        alert('配置失败：' + error.message);
    }
});

// 搜索表单提交
document.getElementById('searchForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const formData = new FormData(e.target);
    const bid = formData.get('bid');
    const bname = formData.get('bname');

    try {
        const response = await fetch(`/api/query?bid=${bid}&bname=${bname}`);
        const data = await response.json();
        
        if (data.error) {
            alert(data.error);
            return;
        }

        // 显示结果
        const resultsDiv = document.getElementById('results');
        let html = '<h3>查询结果</h3>';
        
        // 显示主表数据
        html += '<h4>主表数据</h4>';
        html += '<table border="1"><tr><th>BID</th><th>BName</th><th>User</th><th>Partition</th></tr>';
        data.main_data.forEach(item => {
            html += `<tr><td>${item.bid}</td><td>${item.bname}</td><td>${item.user}</td><td>${item.partition}</td></tr>`;
        });
        html += '</table>';

        // 显示详情数据
        html += '<h4>详情数据</h4>';
        html += '<table border="1"><tr><th>FID</th><th>FName</th><th>BID</th><th>FSize</th></tr>';
        data.details.forEach(item => {
            html += `<tr><td>${item.fid}</td><td>${item.fname}</td><td>${item.bid}</td><td>${item.fsize}</td></tr>`;
        });
        html += '</table>';

        resultsDiv.innerHTML = html;
    } catch (error) {
        alert('查询失败：' + error.message);
    }
});