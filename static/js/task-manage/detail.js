// 任务管理详情模块 - 任务详情、进度、日志加载
(function() {
    'use strict';
    
    window.TaskManageDetail = {};

    // 加载任务详情
    async function loadTaskDetail(taskId) {
        try {
            const response = await fetch(`/api/v1/tasks/${taskId}`);
            const result = await response.json();
            
            if (result.code !== 200) {
                showToast('加载任务详情失败: ' + result.message, 'error');
                return;
            }
            
            window.TaskManageCore.currentTask = result.data;
            
            document.getElementById('taskDetailTitle').innerHTML = `
                <i class="bi bi-activity me-2"></i>${window.TaskManageCore.currentTask.name}
                ${window.TaskManageCore.getStatusBadge(window.TaskManageCore.currentTask)}
            `;
            
            document.getElementById('taskActions').style.display = 'block';
            document.getElementById('taskStats').style.display = 'flex';
            
            await loadTaskProgress(taskId);
            await loadTaskLogs(taskId);
            
        } catch (error) {
            console.error('加载任务详情失败:', error);
        }
    }
    window.TaskManageDetail.loadTaskDetail = loadTaskDetail;

    // 加载任务进度
    async function loadTaskProgress(taskId) {
        try {
            const response = await fetch(`/api/v1/tasks/${taskId}/progress`);
            const result = await response.json();
            
            if (result.code !== 200) {
                return;
            }
            
            const progress = result.data;
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
            
            const tableUnitsHtml = window.TaskManageUI.buildTableUnitsHtml(progress);
            
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
            
            window.TaskManageUI.updateStatsBar(progress);
            
        } catch (error) {
            console.error('加载任务进度失败:', error);
        }
    }
    window.TaskManageDetail.loadTaskProgress = loadTaskProgress;

    // 加载任务日志
    async function loadTaskLogs(taskId) {
        try {
            const response = await fetch(`/api/v1/tasks/${taskId}/logs?limit=50`);
            const result = await response.json();
            
            if (result.code !== 200) {
                return;
            }
            
            const logs = result.data || [];
            const logContent = document.getElementById('logContent');
            
            if (logs.length === 0) {
                logContent.innerHTML = `
                    <div class="empty-state">
                        <i class="bi bi-inbox"></i>
                        <div>暂无日志</div>
                    </div>
                `;
                return;
            }
            
            logContent.innerHTML = logs.map(log => `
                <div class="log-entry ${log.level}">
                    <span class="log-time">${log.time}</span>
                    ${log.message}
                </div>
            `).join('');
            
            logContent.scrollTop = logContent.scrollHeight;
            
        } catch (error) {
            console.error('加载任务日志失败:', error);
        }
    }
    window.TaskManageDetail.loadTaskLogs = loadTaskLogs;

})();
