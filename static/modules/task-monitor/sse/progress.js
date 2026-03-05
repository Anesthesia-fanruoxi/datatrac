// 任务监控 - 进度 SSE 连接（统一接口）
(function() {
    'use strict';
    
    window.TaskMonitorProgressSSE = {
        eventSource: null,
        
        // 启动进度 SSE 连接
        start: function(taskId) {
            this.close();
            
            console.log('启动进度 SSE 连接，任务ID:', taskId);
            this.eventSource = new EventSource(`/api/v1/tasks/${taskId}/stream/progress`);
            
            // 监听进度事件
            this.eventSource.addEventListener('progress', (e) => {
                try {
                    const progress = JSON.parse(e.data);
                    console.log('收到进度更新:', progress);
                    if (window.TaskMonitorDetail) {
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
            
            // 监听连接打开
            this.eventSource.onopen = () => {
                console.log('进度 SSE 连接已建立');
            };
        },
        
        // 关闭 SSE 连接
        close: function() {
            if (this.eventSource) {
                console.log('关闭进度 SSE 连接');
                this.eventSource.close();
                this.eventSource = null;
            }
        }
    };
})();
