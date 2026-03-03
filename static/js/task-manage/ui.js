// 任务管理UI模块 - 进度更新、日志更新、统计信息栏
(function() {
    'use strict';
    
    window.TaskManageUI = {};
    
    // 当前激活的进度步骤
    let currentProgressStep = 'initialize';
    
    // 步骤状态：pending, running, completed
    let stepStatus = {
        initialize: 'running',
        sync_data: 'pending'
    };
    
    // 更新步骤图标
    function updateStepIcon(step, status) {
        const iconElement = document.getElementById(`stepIcon-${step}`);
        if (!iconElement) return;
        
        stepStatus[step] = status;
        
        if (status === 'completed') {
            iconElement.innerHTML = '<i class="bi bi-check-circle-fill"></i>';
            iconElement.classList.add('completed');
            iconElement.classList.remove('running');
        } else if (status === 'running') {
            iconElement.innerHTML = '<i class="bi bi-arrow-repeat spin"></i>';
            iconElement.classList.add('running');
            iconElement.classList.remove('completed');
        } else {
            iconElement.innerHTML = '<i class="bi bi-circle"></i>';
            iconElement.classList.remove('completed', 'running');
        }
    }
    window.TaskManageUI.updateStepIcon = updateStepIcon;
    
    // 切换进度步骤标签
    function switchProgressStep(step) {
        currentProgressStep = step;
        
        // 更新标签样式
        document.querySelectorAll('.progress-step-tab').forEach(t => t.classList.remove('active'));
        document.querySelector(`.progress-step-tab[data-step="${step}"]`).classList.add('active');
        
        // 切换内容显示
        document.querySelectorAll('.progress-step-content').forEach(el => el.style.display = 'none');
        document.getElementById(`progressStep-${step}`).style.display = 'block';
        
        // 断开旧的进度SSE连接，启动新的进度SSE连接
        if (window.TaskManageCore && window.TaskManageCore.currentTaskId) {
            const taskId = window.TaskManageCore.currentTaskId;
            const task = window.TaskManageCore.currentTask;
            
            // 如果任务正在运行，启动对应步骤的进度SSE连接
            if (task && task.is_running) {
                console.log('切换到步骤:', step, '启动进度SSE连接');
                if (window.TaskManageCore.startProgressSSE) {
                    window.TaskManageCore.startProgressSSE(taskId, step);
                }
            } else {
                console.log('任务未运行，不启动进度SSE连接');
                // 断开进度SSE连接
                if (window.TaskManageCore.closeProgressSSE) {
                    window.TaskManageCore.closeProgressSSE();
                }
            }
        }
    }
    window.TaskManageUI.switchProgressStep = switchProgressStep;
    
    // 构建任务单元列表HTML（已废弃，不再使用）
    // 保留此函数以保持向后兼容，但不再调用
    function buildTableUnitsHtml(progress) {
        return '';
    }
    window.TaskManageUI.buildTableUnitsHtml = buildTableUnitsHtml;

    // 更新进度显示（由SSE调用）
    function updateProgress(progress) {
        if (!progress || progress.total_tables === 0) {
            document.getElementById('progressStep-initialize').innerHTML = `
                <div class="empty-state">
                    <i class="bi bi-inbox"></i>
                    <div>任务尚未开始</div>
                </div>
            `;
            document.getElementById('progressStep-sync_data').innerHTML = `
                <div class="empty-state">
                    <i class="bi bi-inbox"></i>
                    <div>等待初始化完成</div>
                </div>
            `;
            return;
        }
        
        // 根据当前步骤更新图标状态（不自动切换标签）
        if (progress.current_step === 'initialize') {
            updateStepIcon('initialize', 'running');
            updateStepIcon('sync_data', 'pending');
        } else if (progress.current_step === 'sync_data') {
            updateStepIcon('initialize', 'completed');
            updateStepIcon('sync_data', 'running');
        } else if (progress.current_step === '' || progress.status === 'completed') {
            // 任务已完成，所有步骤都标记为完成
            updateStepIcon('initialize', 'completed');
            updateStepIcon('sync_data', 'completed');
        }
        
        // 更新对应步骤的进度内容
        if (progress.current_step === 'initialize') {
            updateInitializeProgress(progress);
        } else if (progress.current_step === 'sync_data') {
            updateSyncDataProgress(progress);
        }
        
        updateStatsBar(progress);
    }
    window.TaskManageUI.updateProgress = updateProgress;
    
    // 更新初始化步骤的进度
    function updateInitializeProgress(progress) {
        console.log('更新初始化进度:', progress);
        const container = document.getElementById('progressStep-initialize');
        const initProgress = progress.total_tables > 0 ? (progress.completed_tables / progress.total_tables * 100).toFixed(1) : 0;
        
        container.innerHTML = `
            <div class="alert alert-primary mb-3" style="font-size: 15px; font-weight: 500; border-left: 4px solid;">
                <i class="bi bi-gear me-2"></i>
                <strong>正在初始化</strong>
                <span class="ms-2 text-muted" style="font-weight: normal;">- 创建数据库和表结构</span>
            </div>
            
            <div class="progress-item">
                <div class="progress-label">
                    <span class="progress-label-text">初始化进度</span>
                    <span class="progress-label-value">${initProgress}%</span>
                </div>
                <div class="progress" style="height: 8px;">
                    <div class="progress-bar bg-primary" role="progressbar" 
                         style="width: ${initProgress}%;" 
                         aria-valuenow="${progress.completed_tables}" aria-valuemin="0" aria-valuemax="${progress.total_tables}"></div>
                </div>
            </div>
            
            <div class="progress-item">
                <div class="progress-label">
                    <span class="progress-label-text">已完成表</span>
                    <span class="progress-label-value">${progress.completed_tables} / ${progress.total_tables}</span>
                </div>
                <div class="progress" style="height: 8px;">
                    <div class="progress-bar bg-success" role="progressbar" 
                         style="width: ${initProgress}%;" 
                         aria-valuenow="${progress.completed_tables}" aria-valuemin="0" aria-valuemax="${progress.total_tables}"></div>
                </div>
            </div>
            
            <div class="alert alert-info mt-3 mb-0" style="font-size: 14px;">
                <i class="bi bi-info-circle me-2"></i>
                已完成 ${progress.completed_tables} 个表的结构创建
            </div>
        `;
    }
    
    // 更新数据同步步骤的进度
    function updateSyncDataProgress(progress) {
        const container = document.getElementById('progressStep-sync_data');
        
        container.innerHTML = `
            <div class="alert alert-success mb-3" style="font-size: 15px; font-weight: 500; border-left: 4px solid;">
                <i class="bi bi-arrow-repeat me-2"></i>
                <strong>正在同步数据</strong>
                <span class="ms-2 text-muted" style="font-weight: normal;">- 批量同步表数据</span>
            </div>
            
            <div class="progress-item">
                <div class="progress-label">
                    <span class="progress-label-text">总体进度</span>
                    <span class="progress-label-value">${progress.overall_progress.toFixed(1)}%</span>
                </div>
                <div class="progress" style="height: 8px;">
                    <div class="progress-bar" role="progressbar" 
                         style="width: ${progress.overall_progress}%; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);" 
                         aria-valuenow="${progress.overall_progress}" aria-valuemin="0" aria-valuemax="100"></div>
                </div>
            </div>
            
            <div class="progress-item">
                <div class="progress-label">
                    <span class="progress-label-text">已完成表</span>
                    <span class="progress-label-value">${progress.completed_tables} / ${progress.total_tables}</span>
                </div>
                <div class="progress" style="height: 8px;">
                    <div class="progress-bar bg-success" role="progressbar" 
                         style="width: ${(progress.completed_tables / progress.total_tables * 100).toFixed(1)}%;" 
                         aria-valuenow="${progress.completed_tables}" aria-valuemin="0" aria-valuemax="${progress.total_tables}"></div>
                </div>
            </div>
            
            <div class="progress-item">
                <div class="progress-label">
                    <span class="progress-label-text">已处理记录</span>
                    <span class="progress-label-value">${progress.processed_records.toLocaleString()} / ${progress.total_records.toLocaleString()}</span>
                </div>
                <div class="progress" style="height: 8px;">
                    <div class="progress-bar bg-info" role="progressbar" 
                         style="width: ${progress.total_records > 0 ? (progress.processed_records / progress.total_records * 100).toFixed(1) : 0}%;" 
                         aria-valuenow="${progress.processed_records}" aria-valuemin="0" aria-valuemax="${progress.total_records}"></div>
                </div>
            </div>
            
            ${progress.running_tables > 0 ? `
            <div class="alert alert-info mt-3 mb-0" style="font-size: 14px;">
                <i class="bi bi-info-circle me-2"></i>
                正在同步 ${progress.running_tables} 个表
                ${progress.sync_speed > 0 ? `，速度: ${progress.sync_speed.toLocaleString()} 条/秒` : ''}
                ${progress.estimated_time ? `，预计剩余: ${progress.estimated_time}` : ''}
            </div>
            ` : ''}
            
            ${progress.failed_tables > 0 ? `
            <div class="alert alert-danger mt-3 mb-0" style="font-size: 14px;">
                <i class="bi bi-exclamation-triangle me-2"></i>
                ${progress.failed_tables} 个表同步失败
            </div>
            ` : ''}
        `;
    }

    // 更新统计信息栏
    function updateStatsBar(progress) {
        if (!progress) return;
        
        // 使用新的简化数据结构
        document.getElementById('statSpeed').textContent = progress.sync_speed > 0 ? 
            progress.sync_speed.toLocaleString() : '0';
        document.getElementById('statRemaining').textContent = progress.estimated_time || '-';
        document.getElementById('statProcessed').textContent = progress.processed_records.toLocaleString();
        document.getElementById('statProgress').textContent = progress.overall_progress.toFixed(1) + '%';
    }
    window.TaskManageUI.updateStatsBar = updateStatsBar;

    // 当前选中的日志标签
    let currentLogTab = 'all';
    
    // 保存所有日志
    let allLogs = [];
    
    // 更新日志显示（由SSE调用）
    function updateLogs(logs) {
        // 如果是单条日志，添加到现有日志中
        if (logs && logs.length === 1) {
            // 检查是否已存在相同的日志
            const logExists = allLogs.some(existingLog => 
                existingLog.time === logs[0].time && 
                existingLog.message === logs[0].message
            );
            if (!logExists) {
                allLogs.push(logs[0]);
                // 限制日志数量（所有日志最多1000条）
                if (allLogs.length > 1000) {
                    allLogs = allLogs.slice(allLogs.length - 1000);
                }
            }
        } else if (logs && logs.length > 1) {
            // 如果是多条日志，替换现有日志
            allLogs = logs;
            // 限制日志数量（所有日志最多1000条）
            if (allLogs.length > 1000) {
                allLogs = allLogs.slice(allLogs.length - 1000);
            }
        }
        
        // 根据当前标签过滤日志
        filterLogsByTab(currentLogTab);
    }
    window.TaskManageUI.updateLogs = updateLogs;
    
    // 根据标签过滤日志
    function filterLogsByTab(tab) {
        currentLogTab = tab;
        
        // 更新标签样式
        document.querySelectorAll('.log-tab').forEach(t => t.classList.remove('active'));
        document.querySelector(`.log-tab[data-tab="${tab}"]`).classList.add('active');
        
        const logContent = document.getElementById('logContent');
        
        if (!allLogs || allLogs.length === 0) {
            logContent.innerHTML = `
                <div class="empty-state">
                    <i class="bi bi-inbox"></i>
                    <div>暂无日志</div>
                </div>
            `;
            return;
        }
        
        // 根据标签过滤日志
        let filteredLogs = allLogs;
        if (tab === 'initialize') {
            filteredLogs = allLogs.filter(log => log.category === 'initialize');
        } else if (tab === 'sync') {
            filteredLogs = allLogs.filter(log => log.category === 'sync');
        } else if (tab === 'complete') {
            filteredLogs = allLogs.filter(log => log.category === 'complete');
        }
        
        if (filteredLogs.length === 0) {
            const tabNames = {
                'initialize': '初始化',
                'sync': '数据同步',
                'complete': '完成'
            };
            logContent.innerHTML = `
                <div class="empty-state">
                    <i class="bi bi-inbox"></i>
                    <div>暂无${tabNames[tab] || ''}日志</div>
                </div>
            `;
            return;
        }
        
        logContent.innerHTML = filteredLogs.map(log => `
            <div class="log-entry ${log.level}">
                <span class="log-time">${log.time}</span>
                ${log.message}
            </div>
        `).join('');
        
        logContent.scrollTop = logContent.scrollHeight;
    }
    window.TaskManageUI.filterLogsByTab = filterLogsByTab;

})();
