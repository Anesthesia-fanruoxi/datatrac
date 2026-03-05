// 任务监控 - 任务列表管理
(function() {
    'use strict';
    
    window.TaskMonitorList = {
        currentTaskId: null,
        
        // 加载任务列表
        load: async function() {
            try {
                const result = await HttpUtils.get('/api/v1/tasks');
                
                if (result.code !== 200) {
                    Toast.error('加载任务失败: ' + result.message);
                    return;
                }
                
                const tasks = result.data || [];
                const configuredTasks = tasks.filter(t => t.status === 'configured' || t.is_running);
                const fullTasks = configuredTasks.filter(t => t.sync_mode === 'full');
                const incrementalTasks = configuredTasks.filter(t => t.sync_mode === 'incremental');
                
                this.render('fullTaskList', fullTasks);
                this.render('incrementalTaskList', incrementalTasks);
                
            } catch (error) {
                console.error('加载任务失败:', error);
                Toast.error('加载任务失败: ' + error.message);
            }
        },
        
        // 渲染任务列表
        render: function(containerId, tasks) {
            const container = document.getElementById(containerId);
            
            if (!tasks || tasks.length === 0) {
                container.innerHTML = '<div class="text-center text-muted p-3" style="font-size: 13px;">暂无任务</div>';
                return;
            }
            
            container.innerHTML = tasks.map(task => `
                <div class="task-item ${this.currentTaskId === task.id ? 'active' : ''}" onclick="TaskMonitorList.select('${task.id}')">
                    <div class="task-item-name">${task.name}</div>
                    <div class="task-item-info">
                        ${this.getStatusBadge(task)}
                        <span class="ms-2">${task.source_type} → ${task.target_type}</span>
                    </div>
                </div>
            `).join('');
        },
        
        // 获取状态徽章
        getStatusBadge: function(task) {
            // 1. 运行中：is_running=true
            if (task.is_running) {
                return '<span class="badge bg-primary" style="font-size: 11px;">运行中</span>';
            }
            
            // 2. 已完成：is_running=false 且 current_step 不为空
            if (!task.is_running && task.current_step) {
                return '<span class="badge bg-success" style="font-size: 11px;">已完成</span>';
            }
            
            // 3. 未运行：is_running=false 且 current_step 为空
            return '<span class="badge bg-secondary" style="font-size: 11px;">未运行</span>';
        },
        
        // 选择任务
        select: async function(taskId) {
            this.currentTaskId = taskId;
            
            // 更新选中状态
            document.querySelectorAll('.task-item').forEach(item => {
                item.classList.remove('active');
            });
            event.currentTarget.classList.add('active');
            
            // 关闭旧的 SSE 连接
            if (window.TaskMonitorSSE) {
                window.TaskMonitorSSE.closeAll();
            }
            
            // 加载任务详情
            if (window.TaskMonitorDetail) {
                await window.TaskMonitorDetail.load(taskId);
            }
        },
        
        // 切换分类展开/折叠
        toggleCategory: function() {
            const headerElement = event.currentTarget;
            const list = headerElement.nextElementSibling;
            headerElement.classList.toggle('expanded');
            list.classList.toggle('expanded');
        }
    };
    
    // 全局函数（供 HTML onclick 调用）
    window.toggleCategory = function() {
        TaskMonitorList.toggleCategory();
    };
})();
