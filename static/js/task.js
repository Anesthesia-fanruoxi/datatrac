// 任务管理相关JavaScript

// 全局变量
let currentTaskId = null;

// 加载任务列表
async function loadTaskConfigs() {
    try {
        const response = await fetch('/api/v1/tasks');
        const result = await response.json();
        const tbody = document.getElementById('taskConfigList');
        
        if (!result.data || result.data.length === 0) {
            tbody.innerHTML = '<tr><td colspan="5" class="text-center text-muted">暂无任务，请点击右上角"新建同步任务"按钮创建</td></tr>';
            return;
        }
        
        tbody.innerHTML = result.data.map(task => {
            const direction = `${task.source_type === 'mysql' ? 'MySQL' : 'ES'} → ${task.target_type === 'mysql' ? 'MySQL' : 'ES'}`;
            const statusBadge = {
                'idle': '<span class="badge bg-secondary">未配置</span>',
                'configured': '<span class="badge bg-info">已配置</span>',
                'running': '<span class="badge bg-success">运行中</span>',
                'paused': '<span class="badge bg-warning">已暂停</span>',
                'completed': '<span class="badge bg-primary">已完成</span>',
                'failed': '<span class="badge bg-danger">失败</span>'
            }[task.status] || '<span class="badge bg-secondary">未知</span>';
            
            return `
                <tr>
                    <td><strong>${task.name}</strong></td>
                    <td>
                        <i class="bi bi-arrow-right-circle me-1" style="color: #667eea;"></i>
                        ${direction}
                    </td>
                    <td>${statusBadge}</td>
                    <td>${formatDate(task.created_at)}</td>
                    <td>
                        <button class="btn btn-sm btn-outline-primary" onclick="configureTask('${task.id}')">
                            <i class="bi bi-gear"></i> ${task.status === 'idle' ? '配置' : '修改配置'}
                        </button>
                        ${task.status !== 'idle' ? `
                            <button class="btn btn-sm btn-outline-info" onclick="viewTask('${task.id}')">
                                <i class="bi bi-eye"></i> 查看
                            </button>
                        ` : ''}
                        <button class="btn btn-sm btn-outline-danger" onclick="deleteTask('${task.id}', '${task.name}')">
                            <i class="bi bi-trash"></i> 删除
                        </button>
                    </td>
                </tr>
            `;
        }).join('');
    } catch (error) {
        console.error('加载任务列表失败:', error);
    }
}

// 显示新建任务模态框
function showCreateTaskModal() {
    document.getElementById('createTaskForm').reset();
    document.getElementById('createTaskModal').style.display = 'block';
}

// 关闭新建任务模态框
function closeCreateTaskModal() {
    document.getElementById('createTaskModal').style.display = 'none';
}

// 提交新建任务表单
document.addEventListener('DOMContentLoaded', function() {
    const form = document.getElementById('createTaskForm');
    if (form) {
        form.addEventListener('submit', async function(e) {
            e.preventDefault();
            
            const data = {
                name: document.getElementById('taskName').value,
                source_type: document.getElementById('taskSourceType').value,
                target_type: document.getElementById('taskTargetType').value,
                remark: document.getElementById('taskRemark').value
            };
            
            try {
                const response = await fetch('/api/v1/tasks', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(data)
                });
                const result = await response.json();
                
                if (result.code === 200) {
                    showToast('任务创建成功！', 'success');
                    closeCreateTaskModal();
                    // 跳转到配置页面
                    configureTask(result.data.id);
                } else {
                    showToast('创建失败: ' + result.message, 'error');
                }
            } catch (error) {
                showToast('创建失败: ' + error.message, 'error');
            }
        });
    }
});

// 配置任务
function configureTask(taskId) {
    currentTaskId = taskId;
    // 显示配置向导
    showConfigWizard(taskId);
}

// 查看任务
async function viewTask(taskId) {
    try {
        const response = await fetch(`/api/v1/tasks/${taskId}`);
        const result = await response.json();
        
        if (result.code !== 200) {
            showToast('加载任务失败: ' + result.message, 'error');
            return;
        }
        
        const task = result.data;
        
        // 解析配置
        let config = {};
        try {
            config = JSON.parse(task.config || '{}');
        } catch (e) {
            console.error('配置解析失败:', e);
            showToast('配置解析失败', 'error');
            return;
        }
        
        // 构建查看内容
        const direction = `${task.source_type === 'mysql' ? 'MySQL' : 'Elasticsearch'} → ${task.target_type === 'mysql' ? 'MySQL' : 'Elasticsearch'}`;
        
        // 统计信息
        const totalTables = config.selected_databases?.reduce((sum, db) => sum + (db.tables?.length || 0), 0) || 0;
        const modifiedTables = config.selected_databases?.reduce((sum, db) => {
            return sum + (db.tables?.filter(t => t.is_modified || t.source_table !== t.target_table).length || 0);
        }, 0) || 0;
        const modifiedDatabases = config.selected_databases?.filter(db => db.is_database_modified).length || 0;
        
        // 表列表HTML
        const tableListHtml = config.selected_databases?.map(db => {
            const isDatabaseModified = db.is_database_modified || (db.source_database && db.source_database !== db.database);
            const databaseDisplayName = isDatabaseModified 
                ? `<span style="color: #3182ce;">${db.database}</span><span style="color: #718096; font-size: 12px;">(原名：${db.source_database || db.database})</span>`
                : db.database;
            
            const tablesHtml = db.tables?.map(tableConfig => {
                const isModified = tableConfig.is_modified || tableConfig.source_table !== tableConfig.target_table;
                if (isModified) {
                    return `
                    <div style="padding: 4px 0 4px 40px; font-size: 13px;">
                        <i class="bi bi-table me-1" style="color: #764ba2; font-size: 12px;"></i>
                        <span style="color: #3182ce;">${tableConfig.target_table}</span><span style="color: #718096; font-size: 12px;">(原名：${tableConfig.source_table})</span>
                    </div>`;
                } else {
                    return `
                    <div style="padding: 4px 0 4px 40px; font-size: 13px; color: #4a5568;">
                        <i class="bi bi-table me-1" style="color: #764ba2; font-size: 12px;"></i>${tableConfig.source_table}
                    </div>`;
                }
            }).join('') || '';
            
            return `
            <div style="border-bottom: 1px solid #e9ecef;">
                <div style="padding: 10px; background: #f8f9fa; cursor: pointer; display: flex; align-items: center;" onclick="toggleViewDatabase('${db.database}')">
                    <i class="bi bi-chevron-right me-2" id="view-chevron-${db.database}" style="transition: transform 0.3s; color: #667eea; font-size: 12px;"></i>
                    <i class="bi bi-database me-2" style="color: #667eea; font-size: 14px;"></i>
                    <span style="font-weight: 600; font-size: 14px;">${databaseDisplayName}</span>
                    <span style="color: #718096; font-size: 12px; margin-left: 8px;">(${db.tables?.length || 0})</span>
                </div>
                <div id="view-tables-${db.database}" style="display: none;">
                    ${tablesHtml}
                </div>
            </div>
            `;
        }).join('') || '<div class="text-muted p-3">暂无表配置</div>';
        
        const errorStrategyText = {
            'skip': '跳过错误继续',
            'pause': '遇错暂停'
        };
        const tableStrategyText = {
            'drop': '删除重建',
            'truncate': '清空数据',
            'backup': '备份后重建'
        };
        const syncModeText = {
            'full': '全量同步',
            'incremental': '增量同步'
        };
        
        // 显示模态框
        const modalHtml = `
        <div id="viewTaskModal" style="display:block; position:fixed; top:0; left:0; width:100%; height:100%; background:rgba(0,0,0,0.6); z-index:1001; backdrop-filter: blur(5px);">
            <div style="position:absolute; top:50%; left:50%; transform:translate(-50%,-50%); background:white; padding:0; border-radius:15px; width:70%; max-width:1200px; min-width:800px; height:85vh; max-height:850px; display: flex; flex-direction: column; box-shadow: 0 20px 60px rgba(0,0,0,0.3);">
                <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 20px; border-radius: 15px 15px 0 0; flex-shrink: 0;">
                    <h5 style="color: white; margin: 0;"><i class="bi bi-eye me-2"></i>查看任务配置</h5>
                    <div style="color: rgba(255,255,255,0.9); font-size: 14px; margin-top: 5px;">
                        任务名称: <strong>${task.name}</strong> | 同步方向: <strong>${direction}</strong>
                    </div>
                </div>
                <div style="padding: 30px; flex: 1; overflow-y: auto; display: flex; flex-direction: column;">
                    <div class="row" style="flex: 1;">
                        <div class="col-md-6" style="height: 100%; display: flex; flex-direction: column;">
                            <div class="card" style="flex: 1; display: flex; flex-direction: column;">
                                <div class="card-header bg-light" style="flex-shrink: 0;">
                                    <strong><i class="bi bi-gear me-2"></i>配置信息</strong>
                                </div>
                                <div class="card-body" style="flex: 1; overflow-y: auto; padding: 20px;">
                                    <div class="row mb-3">
                                        <div class="col-5" style="color: #718096; font-size: 14px;">源数据源</div>
                                        <div class="col-7" style="color: #2d3748; font-size: 14px; font-weight: 500;">${task.source_conn?.name || '-'} (${task.source_conn?.host || '-'}:${task.source_conn?.port || '-'})</div>
                                    </div>
                                    <div class="row mb-3">
                                        <div class="col-5" style="color: #718096; font-size: 14px;">目标数据源</div>
                                        <div class="col-7" style="color: #2d3748; font-size: 14px; font-weight: 500;">${task.target_conn?.name || '-'} (${task.target_conn?.host || '-'}:${task.target_conn?.port || '-'})</div>
                                    </div>
                                    <div class="row mb-3">
                                        <div class="col-5" style="color: #718096; font-size: 14px;">同步表数量</div>
                                        <div class="col-7" style="color: #667eea; font-size: 14px; font-weight: 600;">${totalTables} 张表${modifiedTables > 0 ? ` <span style="color: #3182ce;">(${modifiedTables} 个已修改)</span>` : ''}</div>
                                    </div>
                                    <div class="row mb-3">
                                        <div class="col-5" style="color: #718096; font-size: 14px;">数据库名修改</div>
                                        <div class="col-7" style="color: #2d3748; font-size: 14px; font-weight: 500;">${modifiedDatabases > 0 ? `<span style="color: #3182ce;">${modifiedDatabases} 个</span>` : '无'}</div>
                                    </div>
                                    <div class="row mb-3">
                                        <div class="col-5" style="color: #718096; font-size: 14px;">同步模式</div>
                                        <div class="col-7" style="color: #2d3748; font-size: 14px; font-weight: 500;">${syncModeText[config.sync_config?.sync_mode] || '-'}</div>
                                    </div>
                                    <div class="row mb-3">
                                        <div class="col-5" style="color: #718096; font-size: 14px;">批次大小</div>
                                        <div class="col-7" style="color: #2d3748; font-size: 14px;"><span style="font-weight: 600; color: #667eea;">${config.sync_config?.batch_size || '-'}</span> <span style="color: #718096;">条/批次</span></div>
                                    </div>
                                    <div class="row mb-3">
                                        <div class="col-5" style="color: #718096; font-size: 14px;">并发线程数</div>
                                        <div class="col-7" style="color: #2d3748; font-size: 14px;"><span style="font-weight: 600; color: #667eea;">${config.sync_config?.thread_count || '-'}</span> <span style="color: #718096;">个线程</span></div>
                                    </div>
                                    <div class="row mb-3">
                                        <div class="col-5" style="color: #718096; font-size: 14px;">错误处理策略</div>
                                        <div class="col-7" style="color: #2d3748; font-size: 14px; font-weight: 500;">${errorStrategyText[config.sync_config?.error_strategy] || '-'}</div>
                                    </div>
                                    <div class="row mb-0">
                                        <div class="col-5" style="color: #718096; font-size: 14px;">目标表存在策略</div>
                                        <div class="col-7" style="color: #2d3748; font-size: 14px; font-weight: 500;">${tableStrategyText[config.sync_config?.table_exists_strategy] || '-'}</div>
                                    </div>
                                </div>
                            </div>
                        </div>
                        <div class="col-md-6" style="height: 100%; display: flex; flex-direction: column;">
                            <div class="card" style="flex: 1; display: flex; flex-direction: column;">
                                <div class="card-header bg-light" style="flex-shrink: 0;">
                                    <strong><i class="bi bi-table me-2"></i>选中的表</strong>
                                </div>
                                <div class="card-body p-2" style="flex: 1; overflow-y: auto;">
                                    ${tableListHtml}
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
                <div style="padding: 20px; border-top: 1px solid #e2e8f0; background: #f7fafc; border-radius: 0 0 15px 15px; flex-shrink: 0;">
                    <div class="d-flex justify-content-between">
                        <button type="button" class="btn btn-secondary" onclick="closeViewTaskModal()">关闭</button>
                        <button type="button" class="btn btn-primary" onclick="closeViewTaskModal(); configureTask('${taskId}')">
                            <i class="bi bi-pencil"></i> 修改配置
                        </button>
                    </div>
                </div>
            </div>
        </div>
        `;
        
        document.body.insertAdjacentHTML('beforeend', modalHtml);
        
    } catch (error) {
        showToast('加载任务失败: ' + error.message, 'error');
    }
}

// 关闭查看任务模态框
window.closeViewTaskModal = function() {
    const modal = document.getElementById('viewTaskModal');
    if (modal) {
        modal.remove();
    }
};

// 为查看任务模态框添加 ESC 键监听
document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape' || e.keyCode === 27) {
        const viewTaskModal = document.getElementById('viewTaskModal');
        if (viewTaskModal && viewTaskModal.style.display !== 'none') {
            closeViewTaskModal();
        }
    }
});

// 切换查看页面数据库展开/折叠
window.toggleViewDatabase = function(database) {
    const tablesDiv = document.getElementById(`view-tables-${database}`);
    const chevron = document.getElementById(`view-chevron-${database}`);
    
    if (tablesDiv && chevron) {
        if (tablesDiv.style.display === 'none') {
            tablesDiv.style.display = 'block';
            chevron.style.transform = 'rotate(90deg)';
        } else {
            tablesDiv.style.display = 'none';
            chevron.style.transform = 'rotate(0deg)';
        }
    }
};

// 删除任务
function deleteTask(id, name) {
    showConfirm(`确定要删除任务"${name}"吗？<br><span class="text-danger">此操作不可恢复！</span>`, async function() {
        try {
            const response = await fetch(`/api/v1/tasks/${id}`, {method: 'DELETE'});
            const result = await response.json();
            
            if (result.code === 200) {
                showToast('删除成功！', 'success');
                loadTaskConfigs();
            } else {
                showToast('删除失败: ' + result.message, 'error');
            }
        } catch (error) {
            showToast('删除失败: ' + error.message, 'error');
        }
    });
}

// 格式化日期（如果 app.js 中没有定义）
if (typeof formatDate === 'undefined') {
    function formatDate(dateString) {
        if (!dateString) return '-';
        const date = new Date(dateString);
        return date.toLocaleString('zh-CN', {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit'
        });
    }
}
