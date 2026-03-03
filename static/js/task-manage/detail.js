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
            
            // 如果任务正在运行，自动启动SSE连接
            if (window.TaskManageCore.currentTask.is_running) {
                // 启动日志SSE（一直保持连接）
                console.log('任务运行中，启动日志SSE');
                if (window.TaskManageCore.startLogSSE) {
                    window.TaskManageCore.startLogSSE(taskId);
                }
                
                // 启动当前步骤的进度SSE
                const currentStep = getCurrentProgressStep();
                console.log('任务运行中，启动步骤进度SSE:', currentStep);
                if (window.TaskManageCore.startProgressSSE) {
                    window.TaskManageCore.startProgressSSE(taskId, currentStep);
                }
            }
            
        } catch (error) {
            console.error('加载任务详情失败:', error);
        }
    }
    window.TaskManageDetail.loadTaskDetail = loadTaskDetail;
    
    // 获取当前激活的进度步骤
    function getCurrentProgressStep() {
        const activeTab = document.querySelector('.progress-step-tab.active');
        if (activeTab) {
            return activeTab.getAttribute('data-step');
        }
        return 'initialize'; // 默认返回初始化步骤
    }

    // 加载任务进度
    async function loadTaskProgress(taskId) {
        try {
            const response = await fetch(`/api/v1/tasks/${taskId}/progress`);
            const result = await response.json();
            
            if (result.code !== 200) {
                return;
            }
            
            const progress = result.data;
            
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
            
            // 根据当前步骤切换标签和更新图标
            if (progress.current_step === 'initialize') {
                window.TaskManageUI.switchProgressStep('initialize');
                window.TaskManageUI.updateStepIcon('initialize', 'running');
                window.TaskManageUI.updateStepIcon('sync_data', 'pending');
            } else if (progress.current_step === 'sync_data') {
                window.TaskManageUI.switchProgressStep('sync_data');
                window.TaskManageUI.updateStepIcon('initialize', 'completed');
                window.TaskManageUI.updateStepIcon('sync_data', 'running');
            } else {
                window.TaskManageUI.switchProgressStep('initialize');
                window.TaskManageUI.updateStepIcon('initialize', 'running');
                window.TaskManageUI.updateStepIcon('sync_data', 'pending');
            }
            
            // 更新进度显示
            window.TaskManageUI.updateProgress(progress);
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
