// 任务监控 - 进度 SSE 连接（统一接口）
(function() {
    'use strict';
    
    window.TaskMonitorProgressSSE = {
        eventSource: null,
        currentTaskId: null,
        currentDatabase: null,
        currentTargetId: null,
        
        // 启动进度 SSE 连接
        start: function(taskId, targetId, database) {
            this.close();
            
            this.currentTaskId = taskId;
            this.currentDatabase = database || null;
            this.currentTargetId = targetId || null;
            
            // 构建URL，如果有参数则添加
            let url = `/api/v1/tasks/${taskId}/stream/progress`;
            let params = [];
            if (targetId) {
                params.push(`target_id=${encodeURIComponent(targetId)}`);
            }
            if (database) {
                params.push(`database=${encodeURIComponent(database)}`);
            }
            
            if (params.length > 0) {
                url += `?${params.join('&')}`;
            }
            
            this.eventSource = new EventSource(url);
            
            // 监听进度事件
            this.eventSource.addEventListener('progress', (e) => {
                try {
                    const progress = JSON.parse(e.data);
                    // 优先调用抽屉组件的更新方法
                    if (window.TaskMonitorDrawer && typeof window.TaskMonitorDrawer.updateProgressUI === 'function') {
                        window.TaskMonitorDrawer.updateProgressUI(progress);
                    } else if (window.TaskMonitorDetail && typeof window.TaskMonitorDetail.updateProgressUI === 'function') {
                        window.TaskMonitorDetail.updateProgressUI(progress);
                    }
                } catch (error) {
                    console.error('解析进度数据失败:', error);
                }
            });
            
            // 监听错误
            this.eventSource.onerror = (e) => {
                console.error('进度 SSE 连接错误:', e);
                // SSE 会自动重连
            };
        },
        
        // 切换数据库（重新建立SSE连接）
        switchDatabase: function(taskId, database) {
            // 保持当前的 targetId，只切换 database
            this.start(taskId, this.currentTargetId, database);
        },
        
        // 关闭 SSE 连接
        close: function() {
            if (this.eventSource) {
                this.eventSource.close();
                this.eventSource = null;
            }
            this.currentTaskId = null;
            this.currentDatabase = null;
            this.currentTargetId = null;
        }
    };
})();
