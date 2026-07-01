// 任务监控 - 任务列表管理
(function() {
    'use strict';
    
    window.TaskMonitorList = {
        currentTaskId: null,
        sseConnections: [], // 保存所有的 SSE 连接
        
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
                const structureTasks = configuredTasks.filter(t => t.sync_mode === 'structure');
                const fullTasks = configuredTasks.filter(t => t.sync_mode === 'full');
                const incrementalTasks = configuredTasks.filter(t => t.sync_mode === 'incremental');
                
                this.render('structureTaskList', structureTasks);
                this.render('fullTaskList', fullTasks);
                this.render('incrementalTaskList', incrementalTasks);
                
                // 渲染完成后，初始化所有任务的 SSE 状态监听
                this.initSSE(configuredTasks);
                
            } catch (error) {
                console.error('加载任务失败:', error);
                Toast.error('加载任务失败: ' + error.message);
            }
        },
        
        // 清理所有任务的 SSE 连接
        clearSSE: function() {
            if (this.sseConnections && this.sseConnections.length > 0) {
                this.sseConnections.forEach(sse => sse.close());
            }
            this.sseConnections = [];
        },
        
        // 批量初始化 SSE 监听
        initSSE: function(tasks) {
            this.clearSSE();
            
            tasks.forEach(task => {
                const sse = new EventSource(`/api/v1/tasks/${task.id}/stream/detail`);
                
                sse.addEventListener('task_detail', (e) => {
                    try {
                        const detail = JSON.parse(e.data);
                        // 合并最新状态
                        const updatedTask = { ...task, ...detail };
                        
                        // 找到对应的 DOM 并更新状态
                        const taskElement = document.querySelector(`.task-item[data-task-id="${task.id}"]`);
                        if (taskElement) {
                            // 更新信息条（涵盖状态徽章）
                            const infoDiv = taskElement.querySelector('.task-item-info');
                            if (infoDiv) {
                                infoDiv.innerHTML = `
                                    ${this.getStatusBadge(updatedTask)}
                                    <span class="ms-2">${updatedTask.source_type} → ${updatedTask.target_type}</span>
                                `;
                            }
                            
                            // 更新操作按钮（由于只需要刷新启动/停止，可以直接替换原按钮元素）
                            const actionsDiv = taskElement.querySelector('.task-item-actions');
                            if (actionsDiv) {
                                const oldControlBtn = actionsDiv.querySelector('.task-control-btn');
                                if (oldControlBtn) {
                                    oldControlBtn.outerHTML = this.getControlButton(updatedTask);
                                }
                            }
                        }
                    } catch (err) {
                        console.error('解析任务列表 SSE 数据失败:', err);
                    }
                });
                
                sse.onerror = (e) => {
                    console.error(`任务 ${task.id} SSE 连接错误:`, e);
                };
                
                this.sseConnections.push(sse);
            });
        },
        
        // 渲染任务列表
        render: function(containerId, tasks) {
            const container = document.getElementById(containerId);
            
            if (!tasks || tasks.length === 0) {
                container.innerHTML = '<div class="text-center text-muted p-3" style="font-size: 13px;">暂无任务</div>';
                return;
            }
            
            container.innerHTML = tasks.map(task => `
                <div class="task-item ${this.currentTaskId === task.id ? 'active' : ''}" data-task-id="${task.id}">
                    <div class="task-item-name">${task.name}</div>
                    <div class="task-item-info">
                        ${this.getStatusBadge(task)}
                        <span class="ms-2">${task.source_type} → ${task.target_type}</span>
                    </div>
                    <div class="task-item-actions">
                        ${this.getControlButton(task)}
                        <button class="task-action-btn" onclick="TaskMonitorDrawer.openProgress('${task.id}', '${task.name}')"><i class="bi bi-graph-up"></i>详情</button>
                        <button class="task-action-btn" onclick="TaskMonitorDrawer.openLogs('${task.id}', '${task.name}')"><i class="bi bi-terminal"></i>日志</button>
                    </div>
                </div>
            `).join('');
        },
        
        // 获取控制按钮
        getControlButton: function(task) {
            // 根据任务状态显示不同的控制按钮
            if (task.is_running) {
                // 运行中：只显示停止按钮
                return `<button class="task-action-btn task-control-btn task-stop-btn" onclick="TaskMonitorList.stopTask('${task.id}')" title="停止任务"><i class="bi bi-stop-fill"></i>停止</button>`;
            } else {
                // 未运行：显示启动按钮
                return `<button class="task-action-btn task-control-btn task-start-btn" onclick="TaskMonitorList.startTask('${task.id}')" title="启动任务"><i class="bi bi-play-fill"></i>启动</button>`;
            }
        },

        // 获取状态徽章
        getStatusBadge: function(task) {
            // 适配 SSE 推送的数据结构：优先使用 is_running，同时也结合 status 字段判断
            const isRunning = task.is_running === true;
            const currentStep = task.current_step || '';
            
            // 1. 运行中：显示具体步骤
            if (isRunning) {
                if (currentStep === 'initialize') {
                    return '<span class="badge bg-info" style="font-size: 11px;"><i class="bi bi-gear-fill me-1"></i>初始化</span>';
                } else if (currentStep === 'sync_data') {
                    return '<span class="badge bg-primary" style="font-size: 11px;"><i class="bi bi-arrow-clockwise me-1"></i>全量同步</span>';
                } else if (currentStep === 'incremental') {
                    return '<span class="badge bg-warning text-dark" style="font-size: 11px;"><i class="bi bi-activity me-1"></i>增量同步</span>';
                }
                return '<span class="badge bg-primary" style="font-size: 11px;"><i class="bi bi-play-fill me-1"></i>运行中</span>';
            }
            
            // 2. 已完成：非运行状态，且有 current_step（或者是 completed 状态）
            if (!isRunning && (task.current_step === 'completed' || task.status === 'completed')) {
                return '<span class="badge bg-success" style="font-size: 11px;"><i class="bi bi-check-circle-fill me-1"></i>已完成</span>';
            }
            
            // 3. 未运行
            return '<span class="badge bg-secondary" style="font-size: 11px;">未运行</span>';
        },
        
        // 选择任务（现在只用于高亮显示）
        select: function(taskId) {
            this.currentTaskId = taskId;
            
            // 更新选中状态
            document.querySelectorAll('.task-item').forEach(item => {
                item.classList.remove('active');
            });
            
            const selectedItem = document.querySelector(`[data-task-id="${taskId}"]`);
            if (selectedItem) {
                selectedItem.classList.add('active');
            }
        },
        
        // 切换标签页
        switchTab: function(e, categoryType) {
            // 设置Tab active
            document.querySelectorAll('.task-tab').forEach(tab => {
                tab.classList.remove('active');
            });
            if (e && e.currentTarget) {
                e.currentTarget.classList.add('active');
            }
            
            // 设置List active
            document.querySelectorAll('.task-tab-content .task-list').forEach(list => {
                list.classList.remove('active');
            });
            const targetList = document.getElementById(categoryType + 'TaskList');
            if (targetList) {
                targetList.classList.add('active');
            }
        },
        
        // 启动任务
        startTask: async function(taskId) {
            try {
                const result = await HttpUtils.post(`/api/v1/tasks/${taskId}/start`, {});
                
                if (result.code === 200) {
                    Toast.success('任务启动成功！');
                    this.load(); // 重新加载任务列表
                } else {
                    Toast.error('启动失败: ' + result.message);
                }
            } catch (error) {
                Toast.error('启动失败: ' + error.message);
            }
        },
        
        // 停止任务
        stopTask: async function(taskId) {
            Modal.confirm('确定要停止任务吗？', async () => {
                try {
                    const result = await HttpUtils.post(`/api/v1/tasks/${taskId}/stop`, {});
                    
                    if (result.code === 200) {
                        Toast.success('任务已停止！');
                        this.load(); // 重新加载任务列表
                    } else {
                        Toast.error('停止失败: ' + result.message);
                    }
                } catch (error) {
                    Toast.error('停止失败: ' + error.message);
                }
            });
        }
    };
    
    // 全局函数（供 HTML onclick 调用）
    window.switchTab = function(e, categoryType) {
        TaskMonitorList.switchTab(e, categoryType);
    };
})();
