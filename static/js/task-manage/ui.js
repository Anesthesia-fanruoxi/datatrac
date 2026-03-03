// 任务管理UI模块 - 进度更新、日志更新、统计信息栏
(function() {
    'use strict';
    
    window.TaskManageUI = {};
    
    // 构建任务单元列表HTML（已废弃，不再使用）
    // 保留此函数以保持向后兼容，但不再调用
    function buildTableUnitsHtml(progress) {
        return '';
    }
    window.TaskManageUI.buildTableUnitsHtml = buildTableUnitsHtml;

    // 更新进度显示（由SSE调用）
    function updateProgress(progress) {
        const progressContent = document.getElementById('progressContent');
        
        if (!progress || progress.total_tables === 0) {
            progressContent.innerHTML = `
                <div class="empty-state">
                    <i class="bi bi-inbox"></i>
                    <div>任务尚未开始</div>
                </div>
            `;
            return;
        }
        
        // 步骤信息
        let stepInfo = '';
        if (progress.current_step) {
            const stepNames = {
                'initialize': '初始化（创建数据库和表结构）',
                'sync_data': '数据同步',
                'validate': '数据校验'
            };
            const stepName = stepNames[progress.current_step] || progress.current_step;
            stepInfo = `
            <div class="alert alert-primary mb-3" style="font-size: 14px;">
                <i class="bi bi-arrow-right-circle me-2"></i>
                <strong>当前步骤:</strong> ${stepName}
            </div>
            `;
        }
        
        // 简化的进度展示，只显示汇总信息
        progressContent.innerHTML = `
            ${stepInfo}
            
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
        
        updateStatsBar(progress);
    }
    window.TaskManageUI.updateProgress = updateProgress;

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
        if (tab === 'create') {
            filteredLogs = allLogs.filter(log => log.category === 'create');
        } else if (tab === 'complete') {
            filteredLogs = allLogs.filter(log => log.category === 'complete');
        }
        
        if (filteredLogs.length === 0) {
            logContent.innerHTML = `
                <div class="empty-state">
                    <i class="bi bi-inbox"></i>
                    <div>暂无${tab === 'create' ? '创建' : tab === 'complete' ? '完成' : ''}日志</div>
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
