// 任务配置核心逻辑
(function() {
    'use strict';
    
    window.TaskConfigCore = {
        currentTaskId: null,
        
        // 加载任务列表
        loadList: async function() {
            try {
                const result = await HttpUtils.get('/api/v1/tasks');
                const tbody = document.getElementById('taskConfigList');
                
                if (!result.data || result.data.length === 0) {
                    tbody.innerHTML = '<tr><td colspan="5" class="text-center text-muted">暂无任务，请点击右上角"新建同步任务"按钮创建</td></tr>';
                    return;
                }
                
                tbody.innerHTML = result.data.map(task => {
                    const direction = `${task.source_type === 'mysql' ? 'MySQL' : 'ES'} → ${task.target_type === 'mysql' ? 'MySQL' : 'ES'}`;
                    const statusBadge = this.getStatusBadge(task.status);
                    
                    return `
                        <tr>
                            <td><strong>${task.name}</strong></td>
                            <td>
                                <i class="bi bi-arrow-right-circle me-1" style="color: #667eea;"></i>
                                ${direction}
                            </td>
                            <td>${statusBadge}</td>
                            <td>${FormatUtils.formatDate(task.created_at)}</td>
                            <td>
                                <button class="btn btn-sm btn-outline-primary" onclick="TaskConfigCore.configure('${task.id}')">
                                    <i class="bi bi-gear"></i> ${task.status === 'idle' ? '配置' : '修改配置'}
                                </button>
                                ${task.status !== 'idle' ? `
                                    <button class="btn btn-sm btn-outline-info" onclick="TaskConfigCore.view('${task.id}')">
                                        <i class="bi bi-eye"></i> 查看
                                    </button>
                                ` : ''}
                                <button class="btn btn-sm btn-outline-danger" onclick="TaskConfigCore.delete('${task.id}', '${task.name}')">
                                    <i class="bi bi-trash"></i> 删除
                                </button>
                            </td>
                        </tr>
                    `;
                }).join('');
            } catch (error) {
                console.error('加载任务列表失败:', error);
                Toast.error('加载任务列表失败: ' + error.message);
            }
        },
        
        // 获取状态徽章
        getStatusBadge: function(status) {
            const badges = {
                'idle': '<span class="badge bg-secondary">未配置</span>',
                'configured': '<span class="badge bg-info">已配置</span>',
                'running': '<span class="badge bg-success">运行中</span>',
                'paused': '<span class="badge bg-warning">已暂停</span>',
                'completed': '<span class="badge bg-primary">已完成</span>',
                'failed': '<span class="badge bg-danger">失败</span>'
            };
            return badges[status] || '<span class="badge bg-secondary">未知</span>';
        },
        
        // 显示新建任务模态框
        showCreate: function() {
            document.getElementById('createTaskForm').reset();
            document.getElementById('createTaskModal').style.display = 'block';
        },
        
        // 关闭新建任务模态框
        closeCreate: function() {
            document.getElementById('createTaskModal').style.display = 'none';
        },
        
        // 提交新建任务
        submitCreate: async function(e) {
            e.preventDefault();
            
            const data = {
                name: document.getElementById('taskName').value,
                source_type: document.getElementById('taskSourceType').value,
                target_type: document.getElementById('taskTargetType').value,
                remark: document.getElementById('taskRemark').value
            };
            
            try {
                const result = await HttpUtils.post('/api/v1/tasks', data);
                
                if (result.code === 200) {
                    Toast.success('任务创建成功！');
                    this.closeCreate();
                    // 跳转到配置页面
                    this.configure(result.data.id);
                } else {
                    Toast.error('创建失败: ' + result.message);
                }
            } catch (error) {
                Toast.error('创建失败: ' + error.message);
            }
        },
        
        // 配置任务
        configure: function(taskId) {
            if (window.TaskWizard) {
                window.TaskWizard.show(taskId);
            } else {
                Toast.error('配置向导未加载');
            }
        },
        
        // 查看任务
        view: async function(taskId) {
            try {
                const result = await HttpUtils.get(`/api/v1/tasks/${taskId}`);
                
                if (result.code !== 200) {
                    Toast.error('加载任务失败: ' + result.message);
                    return;
                }
                
                const task = result.data;
                
                // 解析配置
                let config = {};
                try {
                    config = JSON.parse(task.config || '{}');
                } catch (e) {
                    console.error('配置解析失败:', e);
                    Toast.error('配置解析失败');
                    return;
                }
                
                // 显示查看模态框
                this.showViewModal(task, config);
                
            } catch (error) {
                Toast.error('加载任务失败: ' + error.message);
            }
        },
        
        // 显示查看模态框
        showViewModal: function(task, config) {
            const direction = `${task.source_type === 'mysql' ? 'MySQL' : 'Elasticsearch'} → ${task.target_type === 'mysql' ? 'MySQL' : 'Elasticsearch'}`;
            const totalTables = config.selected_databases?.reduce((sum, db) => sum + (db.tables?.length || 0), 0) || 0;
            
            const modalHtml = `
            <div id="viewTaskModal" style="display:block; position:fixed; top:0; left:0; width:100%; height:100%; background:rgba(0,0,0,0.6); z-index:1001; backdrop-filter: blur(5px);">
                <div style="position:absolute; top:50%; left:50%; transform:translate(-50%,-50%); background:white; padding:0; border-radius:15px; width:70%; max-width:1000px; max-height:80vh; display: flex; flex-direction: column; box-shadow: 0 20px 60px rgba(0,0,0,0.3);">
                    <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 20px; border-radius: 15px 15px 0 0; flex-shrink: 0;">
                        <h5 style="color: white; margin: 0;"><i class="bi bi-eye me-2"></i>查看任务配置</h5>
                        <div style="color: rgba(255,255,255,0.9); font-size: 14px; margin-top: 5px;">
                            任务名称: <strong>${task.name}</strong> | 同步方向: <strong>${direction}</strong>
                        </div>
                    </div>
                    <div style="padding: 30px; flex: 1; overflow-y: auto;">
                        <div class="row">
                            <div class="col-md-6">
                                <h6>配置信息</h6>
                                <table class="table table-sm">
                                    <tr><td>源数据源</td><td>${task.source_conn?.name || '-'}</td></tr>
                                    <tr><td>目标数据源</td><td>${task.target_conn?.name || '-'}</td></tr>
                                    <tr><td>同步表数量</td><td>${totalTables} 张</td></tr>
                                    <tr><td>同步模式</td><td>${config.sync_config?.sync_mode === 'full' ? '全量同步' : '增量同步'}</td></tr>
                                    <tr><td>批次大小</td><td>${config.sync_config?.batch_size || '-'} 条</td></tr>
                                    <tr><td>并发线程数</td><td>${config.sync_config?.thread_count || '-'} 个</td></tr>
                                </table>
                            </div>
                            <div class="col-md-6">
                                <h6>选中的表</h6>
                                <div style="max-height: 300px; overflow-y: auto;">
                                    ${this.renderTableList(config.selected_databases)}
                                </div>
                            </div>
                        </div>
                    </div>
                    <div style="padding: 20px; border-top: 1px solid #e2e8f0; background: #f7fafc; border-radius: 0 0 15px 15px; flex-shrink: 0;">
                        <div class="d-flex justify-content-between">
                            <button type="button" class="btn btn-secondary" onclick="TaskConfigCore.closeViewModal()">关闭</button>
                            <button type="button" class="btn btn-primary" onclick="TaskConfigCore.closeViewModal(); TaskConfigCore.configure('${task.id}')">
                                <i class="bi bi-pencil"></i> 修改配置
                            </button>
                        </div>
                    </div>
                </div>
            </div>
            `;
            
            document.body.insertAdjacentHTML('beforeend', modalHtml);
        },
        
        // 渲染表列表
        renderTableList: function(databases) {
            if (!databases || databases.length === 0) {
                return '<div class="text-muted">暂无表配置</div>';
            }
            
            return databases.map(db => {
                const tables = db.tables?.map(t => `<li>${t.source_table}</li>`).join('') || '';
                return `
                    <div class="mb-2">
                        <strong><i class="bi bi-database me-1"></i>${db.database}</strong>
                        <ul class="mb-0">${tables}</ul>
                    </div>
                `;
            }).join('');
        },
        
        // 关闭查看模态框
        closeViewModal: function() {
            const modal = document.getElementById('viewTaskModal');
            if (modal) {
                modal.remove();
            }
        },
        
        // 删除任务
        delete: function(id, name) {
            Modal.confirm(
                `确定要删除任务"${name}"吗？<br><span class="text-danger">此操作不可恢复！</span>`,
                async () => {
                    try {
                        const result = await HttpUtils.delete(`/api/v1/tasks/${id}`);
                        
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
        }
    };
})();
