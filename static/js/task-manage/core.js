// 任务管理核心模块 - SSE连接和任务列表
(function() {
    'use strict';
    
    // 导出到全局作用域
    window.TaskManageCore = {
        currentTaskId: null,
        currentTask: null,
        eventSource: null
    };

    // 初始化任务监控页面
    window.initTaskManage = function() {
        console.log('initTaskManage 被调用');
        loadMonitorTasks();
    };

    // 清理任务监控页面
    window.cleanupTaskManage = function() {
        closeSSE();
    };

    // 关闭SSE连接
    function closeSSE() {
        if (window.TaskManageCore.eventSource) {
            console.log('关闭SSE连接');
            window.TaskManageCore.eventSource.close();
            window.TaskManageCore.eventSource = null;
        }
    }
    window.TaskManageCore.closeSSE = closeSSE;

    // 启动SSE连接
    function startSSE(taskId) {
        closeSSE();
        
        console.log('启动SSE连接:', taskId);
        const url = `/api/v1/tasks/${taskId}/stream`;
        window.TaskManageCore.eventSource = new EventSource(url);
        
        // 监听进度事件
        window.TaskManageCore.eventSource.addEventListener('progress', function(e) {
            try {
                const progress = JSON.parse(e.data);
                console.log('收到进度更新:', progress);
                if (window.TaskManageUI && window.TaskManageUI.updateProgress) {
                    window.TaskManageUI.updateProgress(progress);
                }
            } catch (error) {
                console.error('解析进度数据失败:', error);
            }
        });
        
        // 监听日志事件
        window.TaskManageCore.eventSource.addEventListener('log', function(e) {
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
        window.TaskManageCore.eventSource.onerror = function(e) {
            console.error('SSE连接错误:', e);
        };
        
        // 监听连接打开
        window.TaskManageCore.eventSource.onopen = function() {
            console.log('SSE连接已建立');
        };
    }
    window.TaskManageCore.startSSE = startSSE;

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
            const fullTasks = tasks.filter(t => t.sync_mode === 'full');
            const incrementalTasks = tasks.filter(t => t.sync_mode === 'incremental');
            
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
        window.event.currentTarget.classList.add('active');
        
        closeSSE();
        
        if (window.TaskManageDetail && window.TaskManageDetail.loadTaskDetail) {
            await window.TaskManageDetail.loadTaskDetail(taskId);
        }
        
        if (window.TaskManageCore.currentTask && window.TaskManageCore.currentTask.is_running) {
            console.log('任务运行中，启动SSE连接');
            startSSE(taskId);
        } else {
            console.log('任务未运行，不启动SSE连接');
            if (window.TaskManageDetail) {
                await window.TaskManageDetail.loadTaskProgress(taskId);
                await window.TaskManageDetail.loadTaskLogs(taskId);
            }
        }
    };

})();
