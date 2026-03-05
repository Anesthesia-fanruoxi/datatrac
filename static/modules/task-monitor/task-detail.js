// 任务监控 - 任务详情管理
(function() {
    'use strict';
    
    window.TaskMonitorDetail = {
        currentTask: null,
        
        // 加载任务详情
        load: async function(taskId) {
            try {
                const result = await HttpUtils.get(`/api/v1/tasks/${taskId}`);
                
                if (result.code === 200) {
                    this.currentTask = result.data;
                    this.updateHeader();
                    
                    // 首次加载进度和日志（HTTP 请求）
                    await this.loadProgress(taskId);
                    await this.loadLogs(taskId);
                    
                    // 启动 SSE 实时推送
                    this.startSSE(taskId);
                } else {
                    Toast.error('加载任务详情失败: ' + result.message);
                }
            } catch (error) {
                console.error('加载任务详情失败:', error);
                Toast.error('加载任务详情失败: ' + error.message);
            }
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
            
            // 启动日志 SSE（始终连接）
            if (window.TaskMonitorLogsSSE) {
                console.log('启动日志 SSE');
                window.TaskMonitorLogsSSE.start(taskId);
            }
        },
        
        // 更新任务详情（由任务详情SSE调用）
        updateTaskDetail: function(taskDetail) {
            if (!taskDetail) return;
            
            // 更新当前任务信息
            if (this.currentTask) {
                this.currentTask.status = taskDetail.status;
                this.currentTask.is_running = taskDetail.is_running;
                this.currentTask.current_step = taskDetail.current_step;
                this.currentTask.sync_mode = taskDetail.sync_mode;
            }
            
            // 直接重新渲染进度UI（不发起HTTP请求）
            this.updateProgressUI({});
        },
        
        // 更新任务头部
        updateHeader: function() {
            const title = document.getElementById('taskDetailTitle');
            const actions = document.getElementById('taskActions');
            
            if (this.currentTask) {
                title.innerHTML = `<i class="bi bi-info-circle me-2"></i>${this.currentTask.name}`;
                actions.style.display = 'block';
            } else {
                title.innerHTML = '<i class="bi bi-info-circle me-2"></i>请选择任务';
                actions.style.display = 'none';
            }
        },
        
        // 加载任务进度
        loadProgress: async function(taskId) {
            try {
                const result = await HttpUtils.get(`/api/v1/tasks/${taskId}/progress`);
                
                if (result.code === 200) {
                    console.log('进度数据:', result.data); // 调试日志
                    this.updateProgressUI(result.data);
                }
            } catch (error) {
                console.error('加载进度失败:', error);
            }
        },
        
        // 更新进度 UI
        updateProgressUI: function(progress) {
            const container = document.getElementById('progressContent');
            if (!container) return;
            
            // 优先使用当前任务的信息
            const taskStatus = this.currentTask?.status || progress?.status || 'configured';
            const isRunning = this.currentTask?.is_running || false;
            const currentStep = this.currentTask?.current_step || progress?.current_step || '';
            const syncMode = this.currentTask?.sync_mode || progress?.sync_mode || 'full';
            
            // 如果没有进度数据，使用任务的基本信息
            if (!progress || Object.keys(progress).length === 0) {
                progress = {
                    status: taskStatus,
                    sync_mode: syncMode,
                    current_step: currentStep,
                    overall_progress: 0,
                    completed_tables: 0,
                    total_tables: 0
                };
            }
            
            // 定义步骤（根据后端实际使用的步骤值）
            const steps = syncMode === 'full' ? [
                { key: 'initialize', name: '初始化', icon: 'bi-gear' },
                { key: 'sync_data', name: '数据同步', icon: 'bi-arrow-repeat' },
                { key: 'completed', name: '完成', icon: 'bi-check-circle' }
            ] : [
                { key: 'initialize', name: '初始化', icon: 'bi-gear' },
                { key: 'incremental', name: 'Binlog监听', icon: 'bi-broadcast' },
                { key: 'running', name: '运行中', icon: 'bi-play-circle' }
            ];
            
            // 判断是否已完成
            const isCompleted = taskStatus === 'completed' || 
                               (syncMode === 'full' && progress.completed_tables === progress.total_tables && progress.total_tables > 0);
            
            // 渲染步骤进度条（不可点击）
            container.innerHTML = `
                <div class="sync-steps">
                    ${steps.map((step, index) => {
                        let isActive = false;
                        let isStepCompleted = false;
                        let stepStatus = '';
                        
                        if (isCompleted) {
                            // 任务已完成，所有步骤都标记为完成
                            isStepCompleted = true;
                            isActive = step.key === 'completed' || step.key === 'running';
                            stepStatus = '已完成';
                        } else if (!isRunning || !currentStep) {
                            // 任务未运行或没有当前步骤，显示待开始状态
                            if (index === 0) {
                                stepStatus = '待开始';
                            }
                        } else {
                            // 任务进行中，根据 current_step 高亮
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
                    ${this.renderProgressDetail('任务状态', this.getStatusText(taskStatus), false)}
                    ${this.renderProgressDetail('总体进度', (progress.overall_progress ? progress.overall_progress.toFixed(2) : 0) + '%', false)}
                    ${this.renderProgressDetail('表进度', `${progress.completed_tables || 0} / ${progress.total_tables || 0}`, currentStep === 'initialize')}
                    ${progress.processed_records !== undefined ? this.renderProgressDetail('已处理记录', `${this.formatNumber(progress.processed_records)} / ${this.formatNumber(progress.total_records || 0)}`, currentStep === 'sync_data') : ''}
                    ${progress.sync_speed ? this.renderProgressDetail('同步速度', `${this.formatNumber(progress.sync_speed)} 条/秒`, currentStep === 'sync_data') : ''}
                    ${progress.elapsed_time ? this.renderProgressDetail('已用时间', progress.elapsed_time, false) : ''}
                    ${progress.estimated_time ? this.renderProgressDetail('预计剩余', progress.estimated_time, false) : ''}
                    ${syncMode === 'incremental' && progress.incremental_events_applied !== undefined ? this.renderProgressDetail('已应用事件', this.formatNumber(progress.incremental_events_applied), currentStep === 'incremental') : ''}
                </div>
            `;
            
            // 不再需要自动启动步骤进度SSE，因为已经在startSSE中启动了统一的进度SSE
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
        
        // 渲染进度详情项（带高亮）
        renderProgressDetail: function(label, value, highlight) {
            const highlightClass = highlight ? 'highlight' : '';
            return `
                <div class="detail-item ${highlightClass}">
                    <span class="detail-label">${label}：</span>
                    <span class="detail-value">${value}</span>
                </div>
            `;
        },
        
        // 加载任务日志
        loadLogs: async function(taskId) {
            try {
                const result = await HttpUtils.get(`/api/v1/tasks/${taskId}/logs?limit=100`);
                
                if (result.code === 200) {
                    this.updateLogsUI(result.data);
                }
            } catch (error) {
                console.error('加载日志失败:', error);
            }
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
