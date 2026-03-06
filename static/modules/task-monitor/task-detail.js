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
            console.log('启动 SSE 连接，任务ID:', taskId);
            
            // 关闭旧的 SSE 连接
            if (window.TaskMonitorSSE) {
                window.TaskMonitorSSE.closeAll();
            }
            
            // 启动任务详情 SSE（始终连接）
            if (window.TaskDetailSSE) {
                console.log('启动任务详情 SSE');
                window.TaskDetailSSE.start(taskId);
            }
            
            // 启动进度 SSE（始终连接）
            if (window.TaskMonitorProgressSSE) {
                console.log('启动进度 SSE');
                window.TaskMonitorProgressSSE.start(taskId);
            }
            
            // 启动日志 SSE（始终连接，默认all分类）
            if (window.TaskMonitorLogsSSE) {
                console.log('启动日志 SSE');
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
            const isRunning = this.currentTask?.is_running || false;
            
            console.log('更新进度UI - syncMode:', syncMode, 'currentStep:', currentStep, 'progress:', progress);
            
            // 如果是增量模式且有表统计数据,直接更新表格
            if (syncMode === 'incremental' && progress.table_stats) {
                this.updateIncrementalTable(progress.table_stats);
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
                        ${this.renderProgressDetail('总体进度', (progress.overall_progress ? progress.overall_progress.toFixed(2) : 0) + '%', false, true, progress.overall_progress || 0)}
                        ${this.renderProgressDetail('表进度', `${progress.completed_tables || 0} / ${progress.total_tables || 0}`, currentStep === 'initialize', false)}
                        ${progress.processed_records !== undefined ? this.renderProgressDetail('已处理记录', `${this.formatNumber(progress.processed_records)} / ${this.formatNumber(progress.total_records || 0)}`, currentStep === 'sync_data', false) : ''}
                        ${progress.sync_speed ? this.renderProgressDetail('同步速度', `${this.formatNumber(progress.sync_speed)} 条/秒`, currentStep === 'sync_data', false) : ''}
                        ${progress.elapsed_time ? this.renderProgressDetail('已用时间', progress.elapsed_time, false, false) : ''}
                        ${progress.estimated_time ? this.renderProgressDetail('预计剩余', progress.estimated_time, false, false) : ''}
                    </div>
                `;
            }
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
