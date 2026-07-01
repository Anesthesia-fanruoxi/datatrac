// 任务监控 - 任务详情 SSE 连接
(function() {
    'use strict';
    
    window.TaskDetailSSE = {
        eventSource: null,
        
        // 启动任务详情 SSE 连接
        start: function(taskId) {
            this.close();
            
            this.eventSource = new EventSource(`/api/v1/tasks/${taskId}/stream/detail`);
            
            // 监听任务详情事件
            this.eventSource.addEventListener('task_detail', (e) => {
                try {
                    const taskDetail = JSON.parse(e.data);
                    
                    // 更新任务详情
                    if (window.TaskMonitorDetail) {
                        window.TaskMonitorDetail.updateTaskDetail(taskDetail);
                    }
                } catch (error) {
                    console.error('解析任务详情数据失败:', error);
                }
            });
            
            // 监听错误
            this.eventSource.onerror = (e) => {
                console.error('任务详情 SSE 连接错误:', e);
                // SSE 会自动重连
            };
        },
        
        // 关闭 SSE 连接
        close: function() {
            if (this.eventSource) {
                this.eventSource.close();
                this.eventSource = null;
            }
        }
    };
})();
