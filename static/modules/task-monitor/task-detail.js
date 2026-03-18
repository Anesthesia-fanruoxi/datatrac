// 任务监控 - 任务详情管理
(function() {
    'use strict';
    
    window.TaskMonitorDetail = {
        currentTask: null,
        
        // 加载任务详情
        load: async function(taskId) {
            // 设置当前任务 ID（用于 SSE 连接）
            if (!this.currentTask) {
                this.currentTask = { id: taskId };
            } else {
                this.currentTask.id = taskId;
            }
            
            // 更新头部
            this.updateHeader();
            
            // 直接启动 SSE 连接，detail SSE 会推送任务详情
            this.startSSE(taskId);
        },
        
        // 启动 SSE 连接
        startSSE: function(taskId) {
            // 关闭旧的 SSE 连接
            if (window.TaskMonitorSSE) {
                window.TaskMonitorSSE.closeAll();
            }
            
            // 启动任务详情 SSE（始终连接）
            if (window.TaskDetailSSE) {
                window.TaskDetailSSE.start(taskId);
            }
            
            // 启动进度 SSE（始终连接）
            if (window.TaskMonitorProgressSSE) {
                window.TaskMonitorProgressSSE.start(taskId);
            }
            
            // 启动日志 SSE（始终连接，默认all分类）
            if (window.TaskMonitorLogsSSE) {
                window.TaskMonitorLogsSSE.start(taskId, 'all');
            }
        },
        
        // 更新任务详情（由任务详情SSE调用）
        updateTaskDetail: function(taskDetail) {
            if (!taskDetail) return;
            
            // 更新当前任务信息
            if (this.currentTask) {
                this.currentTask.id = taskDetail.id || this.currentTask.id;
                this.currentTask.name = taskDetail.name;
                this.currentTask.status = taskDetail.status;
                this.currentTask.is_running = taskDetail.is_running;
                this.currentTask.current_step = taskDetail.current_step;
                this.currentTask.sync_mode = taskDetail.sync_mode;
            }
            
            // 更新头部显示任务名称
            this.updateHeader();
            
            // 如果是增量模式,初始化表格容器
            if (taskDetail.sync_mode === 'incremental') {
                const container = document.getElementById('progressContent');
                if (container && !container.querySelector('#incrementalTablePlaceholder')) {
                    container.innerHTML = '<div id="incrementalTablePlaceholder"></div>';
                }
            }
            
            // 不重新渲染进度 UI，因为 progress SSE 会单独推送进度数据
            // 只刷新左侧任务列表（更新任务状态显示）
            if (window.TaskMonitorList) {
                window.TaskMonitorList.load();
            }
        },
        
        // 更新任务头部
        updateHeader: function() {
            const title = document.getElementById('taskDetailTitle');
            const actions = document.getElementById('taskActions');
            
            if (this.currentTask && this.currentTask.id) {
                // 显示任务名称（如果有），否则显示 "加载中..."
                const taskName = this.currentTask.name || '加载中...';
                title.innerHTML = `<i class="bi bi-info-circle me-2"></i>${taskName}`;
                actions.style.display = 'block';
            } else {
                title.innerHTML = '<i class="bi bi-info-circle me-2"></i>请选择任务';
                actions.style.display = 'none';
            }
        },
        

        
        // 更新进度 UI
        updateProgressUI: async function(progress) {
            const container = document.getElementById('progressContent');
            if (!container) return;
            
            // 如果没有进度数据,直接返回
            if (!progress) return;
            
            // 从 progress 数据中获取同步模式和步骤信息
            const syncMode = progress.sync_mode || 'full';
            const currentStep = progress.current_step || '';
            
            // 同时更新 currentTask 中的信息(用于其他地方使用)
            if (this.currentTask) {
                this.currentTask.sync_mode = syncMode;
                this.currentTask.current_step = currentStep;
            }
            
            // 从 currentTask 获取任务状态信息
            const taskStatus = this.currentTask?.status || 'configured';
            const isRunning = this.currentTask?.is_running === true || taskStatus === 'running';
            // 判断是否已完成
            const isCompleted = taskStatus === 'completed' || 
                               (progress.completed_tables === progress.total_tables && progress.total_tables > 0);
            
            // 渲染步骤进度条
            if (syncMode === 'incremental') {
                if (progress.database_stats) {
                    // 显示数据库级别统计（第一级）
                    this.updateDatabaseStats(progress.database_stats, progress.table_stats);
                } else if (progress.table_stats) {
                    // 兼容旧版本：直接显示表明细
                    this.updateIncrementalTable(progress.table_stats);
                }
                return;
            }
            
            // 全量模式:渲染进度条和详情
            if (syncMode === 'full') {
                // 定义步骤
                const steps = [
                    { key: 'initialize', name: '初始化', icon: 'bi-gear' },
                    { key: 'sync_data', name: '数据同步', icon: 'bi-arrow-repeat' },
                    { key: 'completed', name: '完成', icon: 'bi-check-circle' }
                ];
                
                // 判断是否已完成
                const isCompleted = taskStatus === 'completed' || 
                                   (progress.completed_tables === progress.total_tables && progress.total_tables > 0);
                
                // 渲染步骤进度条
                container.innerHTML = `
                    <div class="sync-steps">
                        ${steps.map((step, index) => {
                            let isActive = false;
                            let isStepCompleted = false;
                            let stepStatus = '';
                            
                            if (isCompleted) {
                                isStepCompleted = true;
                                isActive = step.key === 'completed';
                                stepStatus = '已完成';
                            } else if (!isRunning || !currentStep) {
                                if (index === 0) {
                                    stepStatus = '待开始';
                                }
                            } else {
                                isActive = step.key === currentStep;
                                isStepCompleted = this.isStepCompleted(steps, currentStep, step.key);
                                if (isActive) {
                                    stepStatus = '进行中...';
                                }
                            }
                            
                            const statusClass = isActive ? 'active' : (isStepCompleted ? 'completed' : '');
                            
                            return `
                                <div class="sync-step ${statusClass}" data-step="${step.key}">
                                    <div class="step-indicator">
                                        <div class="step-number">
                                            ${isStepCompleted && !isActive ? '<i class="bi bi-check"></i>' : (index + 1)}
                                        </div>
                                        <div class="step-line"></div>
                                    </div>
                                    <div class="step-content">
                                        <div class="step-icon"><i class="${step.icon}"></i></div>
                                        <div class="step-name">${step.name}</div>
                                        ${stepStatus ? `<div class="step-status" style="color: ${isCompleted ? '#10b981' : (isActive ? '#667eea' : '#64748b')};">${stepStatus}</div>` : ''}
                                    </div>
                                </div>
                            `;
                        }).join('')}
                    </div>
                    
                    <div class="progress-details mt-3">
                        ${this.renderProgressDetail('任务状态', this.getStatusText(taskStatus), false, false)}
                        ${this.renderProgressDetail('总体进度', (progress.overall_progress != null ? progress.overall_progress.toFixed(2) : 0) + '%', false, true, progress.overall_progress != null ? progress.overall_progress : 0)}
                        ${progress.init_tables != null ? this.renderProgressDetail('表初始化', `${progress.init_tables} / ${progress.total_tables || 0}`, currentStep === 'initialize', false) : ''}
                        ${this.renderProgressDetail('表同步完成', `${progress.completed_tables || 0} / ${progress.total_tables || 0}`, currentStep === 'sync_data', false)}
                        ${progress.processed_records !== undefined ? this.renderProgressDetail('已处理记录', `${this.formatNumber(progress.processed_records)} / ${this.formatNumber(progress.total_records || 0)}`, currentStep === 'sync_data', false) : ''}
                        ${progress.sync_speed ? this.renderProgressDetail('同步速度', `${this.formatNumber(progress.sync_speed)} 条/秒`, currentStep === 'sync_data', false) : ''}
                        ${progress.elapsed_time ? this.renderProgressDetail('已用时间', progress.elapsed_time, false, false) : ''}
                        ${progress.estimated_time ? this.renderProgressDetail('预计剩余', progress.estimated_time, false, false) : ''}
                    </div>
                    ${(progress.target_stats && progress.target_stats.length > 0) ? this.renderTargetStats(progress.target_stats) : ''}
                `;
            }
        },
        
        // 更新数据库级别统计（第一级展示）
        updateDatabaseStats: function(databaseStats, tableStats) {
            const placeholder = document.getElementById('incrementalTablePlaceholder');
            if (!placeholder) return;
            
            if (!databaseStats || databaseStats.length === 0) {
                placeholder.innerHTML = '<div class="text-muted text-center mt-3">暂无统计数据</div>';
                return;
            }
            
            // 检查是否已经存在表格，如果存在则只更新数据，不重新渲染
            const existingTable = placeholder.querySelector('.incremental-database-container');
            if (existingTable) {
                // 只更新数据，保持展开状态
                this.updateDatabaseStatsData(databaseStats, tableStats);
                return;
            }
            
            // 首次渲染：创建完整的表格结构
            placeholder.innerHTML = `
                <div class="incremental-database-container mt-3">
                    <h6 class="mb-2">同步统计</h6>
                    <div class="table-responsive">
                        <table class="table table-sm table-hover">
                            <thead>
                                <tr>
                                    <th width="40"></th>
                                    <th>数据库</th>
                                    <th>表数量</th>
                                    <th>全量同步进度</th>
                                    <th>历史增量</th>
                                    <th>今日增量</th>
                                    <th>INSERT</th>
                                    <th>UPDATE</th>
                                    <th>DELETE</th>
                                </tr>
                            </thead>
                            <tbody id="databaseStatsBody">
                                ${databaseStats.map(db => `
                                    <tr class="database-row" data-database="${db.database}">
                                        <td>
                                            <i class="bi bi-chevron-right expand-icon" style="cursor: pointer;"></i>
                                        </td>
                                        <td><strong>${db.database}</strong></td>
                                        <td>${db.table_count || 0}</td>
                                        <td>
                                            <div class="d-flex align-items-center">
                                                <span class="me-2">${db.full_sync_progress ? db.full_sync_progress.toFixed(2) + '%' : '0%'}</span>
                                                <div class="progress flex-grow-1" style="height: 8px; max-width: 100px;">
                                                    <div class="progress-bar" role="progressbar" 
                                                         style="width: ${db.full_sync_progress || 0}%"
                                                         aria-valuenow="${db.full_sync_progress || 0}" 
                                                         aria-valuemin="0" aria-valuemax="100"></div>
                                                </div>
                                            </div>
                                        </td>
                                        <td>${this.formatNumber(db.total_count || 0)}</td>
                                        <td>${this.formatNumber(db.today_count || 0)}</td>
                                        <td class="text-success">${this.formatNumber(db.total_insert_count || 0)}</td>
                                        <td class="text-primary">${this.formatNumber(db.total_update_count || 0)}</td>
                                        <td class="text-danger">${this.formatNumber(db.total_delete_count || 0)}</td>
                                    </tr>
                                    <tr class="table-detail-row" data-database="${db.database}" style="display: none;">
                                        <td colspan="9">
                                            <div class="table-detail-container p-3 bg-light">
                                                <div class="text-muted text-center">加载中...</div>
                                            </div>
                                        </td>
                                    </tr>
                                `).join('')}
                            </tbody>
                        </table>
                    </div>
                </div>
            `;
            
            // 绑定展开/收起事件
            this.bindDatabaseExpandEvents();
            
            // 如果有表明细数据，更新对应的展开行
            if (tableStats && tableStats.length > 0) {
                this.updateTableDetails(tableStats);
            }
        },
        
        // 更新数据库统计数据（不重新渲染，保持展开状态）
        updateDatabaseStatsData: function(databaseStats, tableStats) {
            databaseStats.forEach(db => {
                const row = document.querySelector(`.database-row[data-database="${db.database}"]`);
                if (!row) return;
                
                // 更新数据（跳过第一列的展开图标）
                const cells = row.querySelectorAll('td');
                if (cells.length >= 9) {
                    cells[2].textContent = db.table_count || 0;
                    
                    // 更新全量同步进度
                    const progressText = cells[3].querySelector('span');
                    const progressBar = cells[3].querySelector('.progress-bar');
                    if (progressText) {
                        progressText.textContent = db.full_sync_progress ? db.full_sync_progress.toFixed(2) + '%' : '0%';
                    }
                    if (progressBar) {
                        const progress = db.full_sync_progress || 0;
                        progressBar.style.width = progress + '%';
                        progressBar.setAttribute('aria-valuenow', progress);
                    }
                    
                    cells[4].textContent = this.formatNumber(db.total_count || 0);
                    cells[5].textContent = this.formatNumber(db.today_count || 0);
                    cells[6].textContent = this.formatNumber(db.total_insert_count || 0);
                    cells[7].textContent = this.formatNumber(db.total_update_count || 0);
                    cells[8].textContent = this.formatNumber(db.total_delete_count || 0);
                }
            });
            
            // 如果有表明细数据，更新对应的展开行
            if (tableStats && tableStats.length > 0) {
                this.updateTableDetails(tableStats);
            }
        },
        
        // 绑定数据库展开/收起事件
        bindDatabaseExpandEvents: function() {
            const rows = document.querySelectorAll('.database-row');
            rows.forEach(row => {
                const expandIcon = row.querySelector('.expand-icon');
                if (!expandIcon) return;
                
                expandIcon.addEventListener('click', (e) => {
                    e.stopPropagation();
                    const database = row.dataset.database;
                    const detailRow = document.querySelector(`.table-detail-row[data-database="${database}"]`);
                    
                    if (!detailRow) return;
                    
                    const isExpanded = detailRow.style.display !== 'none';
                    
                    if (isExpanded) {
                        // 收起
                        detailRow.style.display = 'none';
                        expandIcon.classList.remove('bi-chevron-down');
                        expandIcon.classList.add('bi-chevron-right');
                        
                        // 断开旧的SSE连接，重新连接不带database参数的
                        if (window.TaskMonitorProgressSSE && this.currentTask) {
                            window.TaskMonitorProgressSSE.switchDatabase(this.currentTask.id, null);
                        }
                    } else {
                        // 先收起其他所有展开的行
                        document.querySelectorAll('.table-detail-row').forEach(r => {
                            r.style.display = 'none';
                        });
                        document.querySelectorAll('.expand-icon').forEach(icon => {
                            icon.classList.remove('bi-chevron-down');
                            icon.classList.add('bi-chevron-right');
                        });
                        
                        // 展开当前行
                        detailRow.style.display = '';
                        expandIcon.classList.remove('bi-chevron-right');
                        expandIcon.classList.add('bi-chevron-down');
                        
                        // 切换SSE连接，带上database参数
                        if (window.TaskMonitorProgressSSE && this.currentTask) {
                            window.TaskMonitorProgressSSE.switchDatabase(this.currentTask.id, database);
                        }
                    }
                });
            });
        },
        
        // 更新表明细（第二级展示）
        updateTableDetails: function(tableStats) {
            if (!tableStats || tableStats.length === 0) return;
            
            // 按数据库分组
            const groupedByDatabase = {};
            tableStats.forEach(table => {
                if (!groupedByDatabase[table.database]) {
                    groupedByDatabase[table.database] = [];
                }
                groupedByDatabase[table.database].push(table);
            });
            
            // 更新每个数据库的表明细
            Object.keys(groupedByDatabase).forEach(database => {
                const detailRow = document.querySelector(`.table-detail-row[data-database="${database}"]`);
                if (!detailRow) return;
                
                const container = detailRow.querySelector('.table-detail-container');
                if (!container) return;
                
                const tables = groupedByDatabase[database];
                container.innerHTML = `
                    <table class="table table-sm table-bordered mb-0" style="font-size: 0.875rem;">
                        <thead class="table-light">
                            <tr>
                                <th style="min-width: 150px;">表名</th>
                                <th style="min-width: 100px;">全量总数</th>
                                <th style="min-width: 120px;">全量进度</th>
                                <th style="min-width: 80px;">历史增量</th>
                                <th style="min-width: 80px;">今日增量</th>
                                <th style="min-width: 70px;" class="text-success">INSERT</th>
                                <th style="min-width: 70px;" class="text-primary">UPDATE</th>
                                <th style="min-width: 70px;" class="text-danger">DELETE</th>
                                <th style="min-width: 80px;">复制延迟</th>
                                <th style="min-width: 140px;">最后增量时间</th>
                            </tr>
                        </thead>
                        <tbody>
                            ${tables.map(t => `
                                <tr>
                                    <td><strong>${t.table}</strong></td>
                                    <td>${this.formatNumber(t.full_sync_total_records || 0)}</td>
                                    <td>
                                        <div class="d-flex align-items-center">
                                            <span class="me-2" style="min-width: 45px;">${t.full_sync_progress ? t.full_sync_progress.toFixed(1) + '%' : '0%'}</span>
                                            <div class="progress flex-grow-1" style="height: 6px; max-width: 60px;">
                                                <div class="progress-bar bg-success" role="progressbar" 
                                                     style="width: ${t.full_sync_progress || 0}%"></div>
                                            </div>
                                        </div>
                                    </td>
                                    <td>${this.formatNumber(t.total_count || 0)}</td>
                                    <td>${this.formatNumber(t.today_count || 0)}</td>
                                    <td class="text-success"><strong>${this.formatNumber(t.insert_count || 0)}</strong></td>
                                    <td class="text-primary"><strong>${this.formatNumber(t.update_count || 0)}</strong></td>
                                    <td class="text-danger"><strong>${this.formatNumber(t.delete_count || 0)}</strong></td>
                                    <td>${t.replication_lag_seconds || 0} ms</td>
                                    <td style="font-size: 0.8rem;">${t.last_event_time && t.last_event_time !== '0001-01-01T00:00:00Z' ? this.formatDateTime(t.last_event_time) : '-'}</td>
                                </tr>
                            `).join('')}
                        </tbody>
                    </table>
                `;
            });
        },
        
        // 更新增量表格（增量更新，不重新渲染整个表格）
        updateIncrementalTable: function(tableStats) {
            const placeholder = document.getElementById('incrementalTablePlaceholder');
            if (!placeholder) return;
            
            // 检查表格是否已存在
            let tableContainer = placeholder.querySelector('.incremental-table-container');
            
            if (!tableContainer) {
                // 表格不存在，创建表格
                if (!tableStats || tableStats.length === 0) {
                    placeholder.innerHTML = '<div class="text-muted text-center mt-3">暂无增量统计数据</div>';
                    return;
                }
                
                placeholder.innerHTML = `
                    <div class="incremental-table-container mt-3">
                        <h6 class="mb-2">同步统计</h6>
                        <div class="table-responsive">
                            <table class="table table-sm table-hover">
                                <thead>
                                    <tr>
                                        <th>表名</th>
                                        <th>同步数据总数</th>
                                        <th>同步进度</th>
                                        <th>历史增量</th>
                                        <th>今日增量</th>
                                        <th>复制延迟</th>
                                        <th>最后增量时间</th>
                                    </tr>
                                </thead>
                                <tbody id="incrementalTableBody">
                                </tbody>
                            </table>
                        </div>
                    </div>
                `;
            }
            
            // 更新表格数据
            const tbody = document.getElementById('incrementalTableBody');
            if (!tbody || !tableStats) return;
            
            // 清空并重新填充（简单方案，后续可优化为只更新变化的行）
            tbody.innerHTML = tableStats.map(t => `
                <tr data-table="${t.database}.${t.table}">
                    <td><strong>${t.database}.${t.table}</strong></td>
                    <td>${this.formatNumber(t.full_sync_total_records || 0)}</td>
                    <td>${t.full_sync_progress ? t.full_sync_progress.toFixed(2) + '%' : '0%'}</td>
                    <td>${this.formatNumber(t.total_count || 0)}</td>
                    <td>${this.formatNumber(t.today_count || 0)}</td>
                    <td>${t.replication_lag_seconds ? (t.replication_lag_seconds * 1000) : 0} ms</td>
                    <td>${t.last_event_time ? this.formatDateTime(t.last_event_time) : '-'}</td>
                </tr>
            `).join('');
        },
        

        
        // 获取状态文本
        getStatusText: function(status) {
            const statusMap = {
                'pending': '待执行',
                'configured': '已配置',
                'running': '运行中',
                'paused': '已暂停',
                'completed': '已完成',
                'failed': '失败'
            };
            return statusMap[status] || status;
        },
        
        // 格式化数字（添加千分位）
        formatNumber: function(num) {
            if (num === undefined || num === null) return '0';
            return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',');
        },
        
        // 判断步骤是否已完成
        isStepCompleted: function(steps, currentStep, checkStep) {
            const currentIndex = steps.findIndex(s => s.key === currentStep);
            const checkIndex = steps.findIndex(s => s.key === checkStep);
            return checkIndex < currentIndex;
        },
        
        // 渲染进度详情项（带高亮和进度条）
        renderProgressDetail: function(label, value, highlight, showProgressBar, progressValue) {
            const highlightClass = highlight ? 'highlight' : '';
            
            // 如果需要显示进度条
            if (showProgressBar && progressValue !== undefined) {
                return `
                    <div class="detail-item ${highlightClass}">
                        <div class="detail-row">
                            <span class="detail-label">${label}：</span>
                            <span class="detail-value">${value}</span>
                        </div>
                        <div class="progress-bar-container">
                            <div class="progress-bar-fill" style="width: ${progressValue}%"></div>
                        </div>
                    </div>
                `;
            }
            
            // 普通显示
            return `
                <div class="detail-item ${highlightClass}">
                    <span class="detail-label">${label}：</span>
                    <span class="detail-value">${value}</span>
                </div>
            `;
        },

        // 渲染各目标数据源进度（紧凑表格，多目标不占满屏）
        renderTargetStats: function(targetStats) {
            if (!targetStats || targetStats.length === 0) return '';
            const self = this;
            return `
                <div class="target-stats-section mt-3 pt-3 border-top">
                    <h6 class="mb-2 text-secondary">各目标数据源进度</h6>
                    <div class="table-responsive">
                        <table class="table table-sm table-hover mb-0 target-stats-table">
                            <thead>
                                <tr>
                                    <th style="width:22%">目标</th>
                                    <th style="width:10%">状态</th>
                                    <th style="width:14%">初始化表</th>
                                    <th style="width:14%">已同步</th>
                                    <th style="width:20%">已处理记录</th>
                                    <th style="width:20%">进度</th>
                                </tr>
                            </thead>
                            <tbody>
                                ${targetStats.map(t => `
                                    <tr>
                                        <td class="text-truncate" title="${(t.target_name || t.target_id || '').replace(/"/g, '&quot;')}">${(t.target_name || t.target_id || '未知目标')}</td>
                                        <td><span class="badge ${t.status === 'completed' ? 'bg-success' : (t.status === 'pending' ? 'bg-secondary' : 'bg-primary')}" style="font-size:0.7rem">${t.status === 'completed' ? '已完成' : (t.status === 'pending' ? '待开始' : '同步中')}</span></td>
                                        <td>${t.init_tables != null ? t.init_tables : 0} / ${t.total_tables != null ? t.total_tables : 0}</td>
                                        <td>${t.completed_tables != null ? t.completed_tables : 0} / ${t.total_tables != null ? t.total_tables : 0}</td>
                                        <td>${self.formatNumber(t.processed_records || 0)} / ${self.formatNumber(t.total_records || 0)}</td>
                                        <td>
                                            <div class="progress flex-grow-1" style="height:6px;min-width:60px">
                                                <div class="progress-bar" role="progressbar" style="width:${(t.progress != null ? t.progress : 0)}%"></div>
                                            </div>
                                        </td>
                                    </tr>
                                `).join('')}
                            </tbody>
                        </table>
                    </div>
                </div>
            `;
        },

        // 更新日志 UI（替换所有日志）
        updateLogsUI: function(logs) {
            const container = document.getElementById('logContent');
            if (!container) return;
            
            if (!logs || logs.length === 0) {
                container.innerHTML = '<div class="empty-state"><i class="bi bi-inbox"></i><div>暂无日志</div></div>';
                return;
            }
            
            container.innerHTML = logs.map(log => this.formatLogEntry(log)).join('');
            
            // 滚动到底部
            container.scrollTop = container.scrollHeight;
        },
        
        // 追加单条日志（SSE 推送）
        appendLog: function(log) {
            const container = document.getElementById('logContent');
            if (!container) return;
            
            // 如果是空状态，先清空
            const emptyState = container.querySelector('.empty-state');
            if (emptyState) {
                container.innerHTML = '';
            }
            
            // 追加日志
            container.insertAdjacentHTML('beforeend', this.formatLogEntry(log));
            
            // 滚动到底部
            container.scrollTop = container.scrollHeight;
        },
        
        // 清空日志显示
        clearLogs: function() {
            const container = document.getElementById('logContent');
            if (!container) return;
            
            container.innerHTML = '<div class="empty-state"><i class="bi bi-inbox"></i><div>加载中...</div></div>';
        },
        
        // 切换日志分类
        switchLogCategory: function(category) {
            if (!this.currentTask) return;
            
            // 更新分类标签激活状态
            document.querySelectorAll('.log-category-tab').forEach(tab => {
                tab.classList.remove('active');
            });
            event.currentTarget.classList.add('active');
            
            // 切换SSE连接
            if (window.TaskMonitorLogsSSE) {
                window.TaskMonitorLogsSSE.switchCategory(this.currentTask.id, category);
            }
        },
        
        // 渲染增量统计信息
        renderIncrementalStats: function(progress, currentStep) {
            if (!progress || !progress.incremental_stats) {
                return '';
            }
            
            const stats = progress.incremental_stats;
            const isActive = currentStep === 'incremental';
            
            // 基本统计信息
            const basicStats = `
                ${this.renderProgressDetail('历史同步总数', this.formatNumber(stats.total_events || 0), isActive, false)}
                ${this.renderProgressDetail('今日处理数量', this.formatNumber(stats.today_events || 0), isActive, false)}
                ${stats.last_event_time ? this.renderProgressDetail('最新数据时间', this.formatDateTime(stats.last_event_time), isActive, false) : ''}
                ${this.renderProgressDetail('复制延迟', `${stats.replication_lag_seconds || 0} 秒`, isActive, false)}
                ${stats.current_binlog_file ? this.renderProgressDetail('Binlog 位置', `${stats.current_binlog_file}:${stats.current_binlog_pos || 0}`, false, false) : ''}
            `;
            
            return basicStats;
        },
        

        
        // 获取事件类型徽章颜色
        getEventTypeBadge: function(eventType) {
            const badgeMap = {
                'INSERT': 'success',
                'UPDATE': 'primary',
                'DELETE': 'danger'
            };
            return badgeMap[eventType] || 'secondary';
        },
        
        // 格式化日期时间
        formatDateTime: function(dateStr) {
            if (!dateStr) return '-';
            const date = new Date(dateStr);
            return date.toLocaleString('zh-CN', { 
                year: 'numeric', 
                month: '2-digit', 
                day: '2-digit',
                hour: '2-digit', 
                minute: '2-digit', 
                second: '2-digit' 
            });
        },
        
        // 格式化日志条目
        formatLogEntry: function(log) {
            // 后端Go结构体字段：Time, Message, Level, Category（首字母大写）
            const time = log.Time || log.time || '-';
            const message = log.Message || log.message || '(空消息)';
            const level = (log.Level || log.level || 'info').toLowerCase();
            
            return `
                <div class="log-entry log-${level}">
                    <span class="log-time">${time}</span>
                    <span class="log-message">${message}</span>
                </div>
            `;
        }
    };
})();
