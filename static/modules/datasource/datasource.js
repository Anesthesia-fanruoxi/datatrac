// 数据源核心逻辑
(function() {
    'use strict';
    
    window.DataSourceCore = {
        currentId: null,
        eventSource: null,
        
        // 加载数据源列表
        loadList: async function() {
            try {
                const result = await HttpUtils.get('/api/v1/datasources');
                const tbody = document.getElementById('datasourceList');
                
                if (!result.data || result.data.length === 0) {
                    tbody.innerHTML = '<tr><td colspan="5" class="text-center text-muted">暂无数据源</td></tr>';
                    return;
                }
                
                tbody.innerHTML = result.data.map(ds => `
                    <tr>
                        <td>
                            <i class="bi bi-${ds.type === 'mysql' ? 'database-fill' : 'search'} me-2" style="color: #667eea;"></i>
                            <strong>${ds.name}</strong>
                        </td>
                        <td><span class="badge bg-primary">${ds.type === 'mysql' ? 'MySQL' : 'Elasticsearch'}</span></td>
                        <td><code>${ds.host}:${ds.port}</code></td>
                        <td id="status-${ds.id}">
                            <span class="text-muted">
                                <i class="bi bi-hourglass-split"></i> 等待测试...
                            </span>
                        </td>
                        <td>
                            <button class="btn btn-sm btn-outline-primary" onclick="DataSourceCore.edit('${ds.id}')">
                                <i class="bi bi-pencil"></i> 编辑
                            </button>
                            <button class="btn btn-sm btn-outline-danger" onclick="DataSourceCore.delete('${ds.id}', '${ds.name}')">
                                <i class="bi bi-trash"></i> 删除
                            </button>
                        </td>
                    </tr>
                `).join('');
                
                // 启动 SSE 测试
                if (window.DataSourceSSE) {
                    window.DataSourceSSE.start();
                }
            } catch (error) {
                console.error('加载数据源失败:', error);
                Toast.error('加载数据源失败: ' + error.message);
            }
        },
        

        // 显示新建表单
        showAdd: function() {
            document.getElementById('datasourceModal').style.display = 'block';
            document.getElementById('datasourceForm').reset();
            document.getElementById('datasourceId').value = '';
            document.getElementById('modalTitle').textContent = '新建数据源';
            document.getElementById('dsPassword').required = true;
            document.getElementById('dsPassword').placeholder = '请输入密码';
            
            // 添加ESC键关闭功能
            this.modalEscHandler = (e) => {
                if (e.key === 'Escape') {
                    this.closeModal();
                }
            };
            document.addEventListener('keydown', this.modalEscHandler);
        },
        
        // 编辑数据源
        edit: async function(id) {
            try {
                const result = await HttpUtils.get(`/api/v1/datasources/${id}`);
                
                if (result.code === 200) {
                    const ds = result.data;
                    document.getElementById('datasourceId').value = ds.id;
                    document.getElementById('dsName').value = ds.name;
                    document.getElementById('dsType').value = ds.type;
                    document.getElementById('dsHost').value = ds.host;
                    document.getElementById('dsPort').value = ds.port;
                    document.getElementById('dsUsername').value = ds.username;
                    document.getElementById('dsDatabase').value = ds.database_name || '';
                    document.getElementById('dsPassword').value = '';
                    document.getElementById('dsPassword').placeholder = '留空表示不修改';
                    document.getElementById('dsPassword').required = false;
                    
                    document.getElementById('dsType').dispatchEvent(new Event('change'));
                    document.getElementById('modalTitle').textContent = '编辑数据源';
                    document.getElementById('datasourceModal').style.display = 'block';
                    
                    // 添加ESC键关闭功能
                    this.modalEscHandler = (e) => {
                        if (e.key === 'Escape') {
                            this.closeModal();
                        }
                    };
                    document.addEventListener('keydown', this.modalEscHandler);
                } else {
                    Toast.error('加载失败: ' + result.message);
                }
            } catch (error) {
                Toast.error('加载失败: ' + error.message);
            }
        },
        
        // 删除数据源
        delete: function(id, name) {
            Modal.confirm(
                `确定要删除数据源"${name}"吗？<br><span class="text-danger">此操作不可恢复！</span>`,
                async () => {
                    try {
                        const result = await HttpUtils.delete(`/api/v1/datasources/${id}`);
                        
                        if (result.code === 200) {
                            Toast.success('删除成功！');
                            this.loadList();
                        } else {
                            Toast.error('删除失败: ' + result.message);
                        }
                    } catch (error) {
                        Toast.error('删除失败: ' + error.message);
                    }
                }
            );
        },
        
        // 关闭模态框
        closeModal: function() {
            document.getElementById('datasourceModal').style.display = 'none';
            // 移除ESC键监听
            if (this.modalEscHandler) {
                document.removeEventListener('keydown', this.modalEscHandler);
                this.modalEscHandler = null;
            }
        },
        
        // 测试连接
        testConnection: async function() {
            const data = {
                type: document.getElementById('dsType').value,
                host: document.getElementById('dsHost').value,
                port: parseInt(document.getElementById('dsPort').value),
                username: document.getElementById('dsUsername').value,
                password: document.getElementById('dsPassword').value,
                database_name: document.getElementById('dsDatabase').value
            };

            if (!data.type || !data.host || !data.port || !data.username || !data.password) {
                Toast.warning('请填写完整信息');
                return;
            }

            try {
                Toast.info('正在测试连接...');
                const result = await HttpUtils.post('/api/v1/datasources/test', data);
                
                if (result.code === 200) {
                    Toast.success('连接成功！版本: ' + (result.data.version || '未知'));
                } else {
                    Toast.error('连接失败: ' + result.message);
                }
            } catch (error) {
                Toast.error('连接失败: ' + error.message);
            }
        },
        
        // 提交表单
        submitForm: async function(e) {
            e.preventDefault();
            
            const id = document.getElementById('datasourceId').value;
            const data = {
                name: document.getElementById('dsName').value,
                type: document.getElementById('dsType').value,
                host: document.getElementById('dsHost').value,
                port: parseInt(document.getElementById('dsPort').value),
                username: document.getElementById('dsUsername').value,
                password: document.getElementById('dsPassword').value,
                database_name: document.getElementById('dsDatabase').value
            };

            try {
                const url = id ? `/api/v1/datasources/${id}` : '/api/v1/datasources';
                const result = id ? await HttpUtils.put(url, data) : await HttpUtils.post(url, data);
                
                if (result.code === 200) {
                    Toast.success(id ? '更新成功！' : '创建成功！');
                    this.closeModal();
                    this.loadList();
                } else {
                    Toast.error('操作失败: ' + result.message);
                }
            } catch (error) {
                Toast.error('操作失败: ' + error.message);
            }
        }
    };
})();
