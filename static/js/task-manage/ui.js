// 任务管理UI模块 - 进度更新、日志更新、统计信息栏
(function() {
    'use strict';
    
    window.TaskManageUI = {};
    
    // 构建任务单元列表HTML
    function buildTableUnitsHtml(progress) {
        if (!progress.table_units || progress.table_units.length === 0) {
            return '';
        }
        
        // 排序：running → pending → completed/failed/paused
        const sortedUnits = [...progress.table_units].sort((a, b) => {
            const order = { 'running': 1, 'pending': 2, 'completed': 3, 'failed': 3, 'paused': 3 };
            return (order[a.status] || 99) - (order[b.status] || 99);
        });
        
        // 为不同的数据库分配颜色
        const dbColors = ['#667eea', '#f093fb', '#4facfe', '#43e97b', '#fa709a', '#30cfd0', '#a8edea', '#fbc2eb'];
        const dbColorMap = {};
        let colorIndex = 0;
        
        sortedUnits.forEach(unit => {
            const dbName = unit.name.split('.')[0];
            if (!dbColorMap[dbName]) {
                dbColorMap[dbName] = dbColors[colorIndex % dbColors.length];
                colorIndex++;
            }
        });
        
        return `
            <div style="margin-top: 20px;">
                <h6 style="color: #2d3748; font-size: 14px; font-weight: 600; margin-bottom: 12px;">
                    <i class="bi bi-table me-2"></i>同步表列表
                </h6>
                <div>
                    ${sortedUnits.map(unit => {
                        // 使用任务单元的实际状态
                        const displayStatus = unit.status;
                        
                        const statusIcon = {
                            'pending': '<i class="bi bi-clock text-secondary"></i>',
                            'running': '<i class="bi bi-arrow-repeat text-primary"></i>',
                            'completed': '<i class="bi bi-check-circle text-success"></i>',
                            'failed': '<i class="bi bi-x-circle text-danger"></i>',
                            'paused': '<i class="bi bi-pause-circle text-warning"></i>'
                        }[displayStatus] || '<i class="bi bi-circle text-muted"></i>';
                        
                        const statusText = {
                            'pending': '等待中',
                            'running': '同步中',
                            'completed': '已完成',
                            'failed': '失败',
                            'paused': '已暂停'
                        }[displayStatus] || '未知';
                        
                        const parts = unit.name.split('.');
                        const dbName = parts[0];
                        const tableName = parts.slice(1).join('.');
                        const dbColor = dbColorMap[dbName];
                        
                        const bgColor = displayStatus === 'completed' ? '#f0fdf4' : '#f8f9fa';
                        const borderColor = displayStatus === 'completed' ? '#22c55e' : 
                            displayStatus === 'running' ? '#1890ff' : 
                            displayStatus === 'failed' ? '#ff4d4f' : '#d9d9d9';
                        
                        // 计算进度百分比
                        const progressPercent = unit.total_records > 0 ? unit.progress : (displayStatus === 'completed' ? 100 : 0);
                        
                        return `
                            <div style="padding: 10px; margin-bottom: 8px; background: ${bgColor}; border-radius: 6px; border-left: 3px solid ${borderColor};">
                                <div style="display: flex; align-items: center; gap: 12px;">
                                    <div style="flex-shrink: 0;">
                                        ${statusIcon}
                                    </div>
                                    <div style="flex: 1; min-width: 0;">
                                        <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 4px;">
                                            <div style="font-size: 13px; font-weight: 500; color: #2d3748;">
                                                <span style="color: ${dbColor}; font-weight: 600;">${dbName}</span>
                                                <span style="color: #718096;">.</span>
                                                <span>${tableName}</span>
                                                <span style="font-size: 12px; color: #718096; margin-left: 8px;">${statusText}</span>
                                            </div>
                                            <span style="font-size: 11px; color: #718096; white-space: nowrap;">
                                                ${unit.processed_records.toLocaleString()} / ${unit.total_records.toLocaleString()} (${progressPercent.toFixed(1)}%)
                                            </span>
                                        </div>
                                        <div style="display: flex; align-items: center; gap: 8px;">
                                            <div class="progress" style="flex: 1; height: 4px;">
                                                <div class="progress-bar ${
                                                    displayStatus === 'completed' ? 'bg-success' : 
                                                    displayStatus === 'running' ? 'bg-primary' : 
                                                    displayStatus === 'failed' ? 'bg-danger' : 'bg-secondary'
                                                }" style="width: ${progressPercent}%;"></div>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        `;
                    }).join('')}
                </div>
            </div>
        `;
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
        
        const tableUnitsHtml = buildTableUnitsHtml(progress);
        
        progressContent.innerHTML = `
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
                    <span class="progress-label-text">已同步表</span>
                    <span class="progress-label-value">${progress.completed_tables} / ${progress.total_tables}</span>
                </div>
                <div class="progress" style="height: 8px;">
                    <div class="progress-bar bg-success" role="progressbar" 
                         style="width: ${(progress.completed_tables / progress.total_tables * 100).toFixed(1)}%;" 
                         aria-valuenow="${progress.completed_tables}" aria-valuemin="0" aria-valuemax="${progress.total_tables}"></div>
                </div>
            </div>
            
            ${tableUnitsHtml}
        `;
        
        updateStatsBar(progress);
    }
    window.TaskManageUI.updateProgress = updateProgress;

    // 更新统计信息栏
    function updateStatsBar(progress) {
        if (!progress) return;
        
        let totalProcessed = 0;
        if (progress.table_units && progress.table_units.length > 0) {
            totalProcessed = progress.table_units.reduce((sum, unit) => sum + unit.processed_records, 0);
        }
        
        document.getElementById('statSpeed').textContent = progress.sync_speed > 0 ? 
            progress.sync_speed.toLocaleString() : '0.00';
        document.getElementById('statRemaining').textContent = progress.estimated_time || '-';
        document.getElementById('statProcessed').textContent = totalProcessed.toLocaleString();
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
