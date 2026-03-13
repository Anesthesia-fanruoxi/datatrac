// 任务监控 - 进度 SSE 连接（统一接口）
(function() {
    'use strict';
    
    window.TaskMonitorProgressSSE = {
        eventSource: null,
        currentTaskId: null,
        currentDatabase: null,
        
        // 启动进度 SSE 连接
        start: function(taskId, database) {
            this.close();
            
            this.currentTaskId = taskId;
            this.currentDatabase = database || null;
            
            // 构建URL，如果有database参数则添加
            let url = `/api/v1/tasks/${taskId}/stream/progress`;
            if (database) {
                url += `?database=${encodeURIComponent(database)}`;
            }
            
            this.eventSource = new EventSource(url);
            
            // 监听进度事件
            this.eventSource.addEventListener('progress', (e) => {
                try {
                    const progress = JSON.parse(e.data);
                    if (window.TaskMonitorDetail && typeof window.TaskMonitorDetail.updateProgressUI === 'function') {
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
            this.start(taskId, database);
        },
        
        // 关闭 SSE 连接
        close: function() {
            if (this.eventSource) {
                this.eventSource.close();
                this.eventSource = null;
            }
            this.currentTaskId = null;
            this.currentDatabase = null;
        }
    };
})();
