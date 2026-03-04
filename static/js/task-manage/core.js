// 任务管理核心模块 - SSE连接和任务列表
(function() {
    'use strict';
    
    // 导出到全局作用域
    window.TaskManageCore = {
        currentTaskId: null,
        currentTask: null,
        progressEventSource: null,  // 进度SSE连接
        logEventSource: null,        // 日志SSE连接
        currentStep: null            // 当前连接的步骤
    };

    // 初始化任务监控页面
    window.initTaskManage = function(taskId) {
        console.log('initTaskManage 被调用, taskId:', taskId);
        loadMonitorTasks();
        
        // 如果提供了taskId，自动选择该任务
        if (taskId) {
            setTimeout(function() {
                console.log('自动选择任务:', taskId);
                autoSelectTask(taskId);
            }, 500);
        }
    };

    // 清理任务监控页面
    window.cleanupTaskManage = function() {
        closeProgressSSE();
        closeLogSSE();
    };

    // 关闭进度SSE连接
    function closeProgressSSE() {
        if (window.TaskManageCore.progressEventSource) {
            console.log('关闭进度SSE连接');
            window.TaskManageCore.progressEventSource.close();
            window.TaskManageCore.progressEventSource = null;
        }
    }
    window.TaskManageCore.closeProgressSSE = closeProgressSSE;

    // 关闭日志SSE连接
    function closeLogSSE() {
        if (window.TaskManageCore.logEventSource) {
            console.log('关闭日志SSE连接');
            window.TaskManageCore.logEventSource.close();
            window.TaskManageCore.logEventSource = null;
        }
    }
    window.TaskManageCore.closeLogSSE = closeLogSSE;

    // 启动进度SSE连接（按步骤）
    function startProgressSSE(taskId, step) {
        closeProgressSSE();
        
        if (!step) {
            console.warn('未指定步骤，不启动进度SSE连接');
            return;
        }
        
        console.log('启动进度SSE连接:', taskId, '步骤:', step);
        window.TaskManageCore.currentStep = step;
        
        // 根据步骤选择不同的SSE接口
        let url;
        if (step === 'initialize') {
            url = `/api/v1/tasks/${taskId}/stream/initialize`;
        } else if (step === 'sync_data') {
            url = `/api/v1/tasks/${taskId}/stream/sync`;
        } else {
            console.warn('未知步骤:', step);
            return;
        }
        
        window.TaskManageCore.progressEventSource = new EventSource(url);
        
        // 监听进度事件
        window.TaskManageCore.progressEventSource.addEventListener('progress', function(e) {
            try {
                const progress = JSON.parse(e.data);
                console.log('收到进度更新 [' + step + ']:', progress);
                if (window.TaskManageUI && window.TaskManageUI.updateProgress) {
                    window.TaskManageUI.updateProgress(progress);
                }
            } catch (error) {
                console.error('解析进度数据失败:', error);
            }
        });
        
        // 监听错误
        window.TaskManageCore.progressEventSource.onerror = function(e) {
            console.error('进度SSE连接错误 [' + step + ']:', e);
        };
        
        // 监听连接打开
        window.TaskManageCore.progressEventSource.onopen = function() {
            console.log('进度SSE连接已建立 [' + step + ']');
        };
    }
    window.TaskManageCore.startProgressSSE = startProgressSSE;

    // 启动日志SSE连接
    function startLogSSE(taskId) {
        closeLogSSE();
        
        console.log('启动日志SSE连接:', taskId);
        const url = `/api/v1/tasks/${taskId}/stream/logs`;
        window.TaskManageCore.logEventSource = new EventSource(url);
        
        // 监听日志事件
        window.TaskManageCore.logEventSource.addEventListener('log', function(e) {
            try {
                const logs = JSON.parse(e.data);
                console.log('收到日志更新:', logs.length, '条');
                if (window.TaskManageUI && window.TaskManageUI.updateLogs) {
                    window.TaskManageUI.updateLogs(logs);
                }
            } catch (error) {
                console.error('解析日志数据失败:', error);
            }
        });
        
        // 监听错误
        window.TaskManageCore.logEventSource.onerror = function(e) {
            console.error('日志SSE连接错误:', e);
        };
        
        // 监听连接打开
        window.TaskManageCore.logEventSource.onopen = function() {
            console.log('日志SSE连接已建立');
        };
    }
    window.TaskManageCore.startLogSSE = startLogSSE;

    // 加载任务列表
    async function loadMonitorTasks() {
        console.log('loadMonitorTasks 开始执行');
        try {
            const response = await fetch('/api/v1/tasks');
            const result = await response.json();
            
            if (result.code !== 200) {
                console.error('API返回错误:', result.message);
                showToast('加载任务失败: ' + result.message, 'error');
                document.getElementById('fullTaskList').innerHTML = '<div class="text-center text-danger p-3" style="font-size: 13px;">加载失败</div>';
                document.getElementById('incrementalTaskList').innerHTML = '<div class="text-center text-danger p-3" style="font-size: 13px;">加载失败</div>';
                return;
            }
            
            const tasks = result.data || [];
            // 只显示已配置的任务
            const configuredTasks = tasks.filter(t => t.status === 'configured');
            const fullTasks = configuredTasks.filter(t => t.sync_mode === 'full');
            const incrementalTasks = configuredTasks.filter(t => t.sync_mode === 'incremental');
            
            renderTaskList('fullTaskList', fullTasks);
            renderTaskList('incrementalTaskList', incrementalTasks);
            
        } catch (error) {
            console.error('加载任务失败:', error);
            document.getElementById('fullTaskList').innerHTML = '<div class="text-center text-danger p-3" style="font-size: 13px;">加载失败: ' + error.message + '</div>';
            document.getElementById('incrementalTaskList').innerHTML = '<div class="text-center text-danger p-3" style="font-size: 13px;">加载失败: ' + error.message + '</div>';
        }
    }
    window.TaskManageCore.loadMonitorTasks = loadMonitorTasks;

    // 渲染任务列表
    function renderTaskList(containerId, tasks) {
        const container = document.getElementById(containerId);
        
        if (!tasks || tasks.length === 0) {
            container.innerHTML = '<div class="text-center text-muted p-3" style="font-size: 13px;">暂无任务</div>';
            return;
        }
        
        container.innerHTML = tasks.map(task => `
            <div class="task-item ${window.TaskManageCore.currentTaskId === task.id ? 'active' : ''}" onclick="selectTask('${task.id}')">
                <div class="task-item-name">${task.name}</div>
                <div class="task-item-info">
                    ${getStatusBadge(task)}
                    <span class="ms-2">${task.source_type} → ${task.target_type}</span>
                </div>
            </div>
        `).join('');
    }

    // 获取状态徽章
    function getStatusBadge(task) {
        if (task.is_running) {
            return '<span class="badge bg-primary" style="font-size: 11px;">运行中</span>';
        }
        if (task.status === 'idle') {
            return '<span class="badge bg-secondary" style="font-size: 11px;">未配置</span>';
        }
        return '<span class="badge bg-info" style="font-size: 11px;">已配置</span>';
    }
    window.TaskManageCore.getStatusBadge = getStatusBadge;

    // 切换分类展开/折叠
    window.toggleCategory = function(category) {
        const headerElement = window.event.currentTarget;
        const list = headerElement.nextElementSibling;
        headerElement.classList.toggle('expanded');
        list.classList.toggle('expanded');
    };

    // 选择任务
    window.selectTask = async function(taskId) {
        window.TaskManageCore.currentTaskId = taskId;
        
        document.querySelectorAll('.task-item').forEach(item => {
            item.classList.remove('active');
        });
        if (window.event && window.event.currentTarget) {
            window.event.currentTarget.classList.add('active');
        }
        
        // 断开旧的SSE连接
        closeProgressSSE();
        closeLogSSE();
        
        // 加载任务详情
        if (window.TaskManageDetail && window.TaskManageDetail.loadTaskDetail) {
            await window.TaskManageDetail.loadTaskDetail(taskId);
        }
        
        // 加载静态数据
        console.log('任务已选择，等待用户点击步骤标签启动进度SSE');
        if (window.TaskManageDetail) {
            await window.TaskManageDetail.loadTaskProgress(taskId);
            await window.TaskManageDetail.loadTaskLogs(taskId);
        }
    };
    
    // 自动选择任务（不依赖点击事件）
    async function autoSelectTask(taskId) {
        window.TaskManageCore.currentTaskId = taskId;
        
        // 高亮选中的任务
        document.querySelectorAll('.task-item').forEach(item => {
            item.classList.remove('active');
            if (item.getAttribute('onclick') && item.getAttribute('onclick').includes(taskId)) {
                item.classList.add('active');
            }
        });
        
        // 断开旧的SSE连接
        closeProgressSSE();
        closeLogSSE();
        
        // 加载任务详情
        if (window.TaskManageDetail && window.TaskManageDetail.loadTaskDetail) {
            await window.TaskManageDetail.loadTaskDetail(taskId);
        }
        
        // 加载静态数据
        console.log('任务已选择，等待用户点击步骤标签启动进度SSE');
        if (window.TaskManageDetail) {
            await window.TaskManageDetail.loadTaskProgress(taskId);
            await window.TaskManageDetail.loadTaskLogs(taskId);
        }
    }
    window.TaskManageCore.autoSelectTask = autoSelectTask;

})();
