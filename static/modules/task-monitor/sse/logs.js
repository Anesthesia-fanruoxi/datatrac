// 任务监控 - 日志 SSE 连接
(function() {
    'use strict';
    
    window.TaskMonitorLogsSSE = {
        eventSource: null,
        
        // 启动日志 SSE 连接
        start: function(taskId) {
            this.close();
            
            console.log('启动日志 SSE 连接，任务ID:', taskId);
            this.eventSource = new EventSource(`/api/v1/tasks/${taskId}/stream/logs`);
            
            // 监听日志事件
            this.eventSource.addEventListener('log', (e) => {
                try {
                    const logs = JSON.parse(e.data);
                    console.log('收到日志（数组）:', logs);
                    
                    // 后端推送的是日志数组，需要遍历追加
                    if (window.TaskMonitorDetail && Array.isArray(logs)) {
                        logs.forEach(log => {
                            window.TaskMonitorDetail.appendLog(log);
                        });
                    }
                } catch (error) {
                    console.error('解析日志数据失败:', error);
                }
            });
            
            // 监听错误
            this.eventSource.onerror = (e) => {
                console.error('日志 SSE 连接错误:', e);
                // SSE 会自动重连，这里只记录错误
            };
            
            // 监听连接打开
            this.eventSource.onopen = () => {
                console.log('日志 SSE 连接已建立');
            };
        },
        
        // 关闭 SSE 连接
        close: function() {
            if (this.eventSource) {
                console.log('关闭日志 SSE 连接');
                this.eventSource.close();
                this.eventSource = null;
            }
        }
    };
    
    // SSE 管理器
    window.TaskMonitorSSE = {
        closeAll: function() {
            console.log('关闭所有 SSE 连接');
            if (window.TaskDetailSSE) {
                window.TaskDetailSSE.close();
            }
            if (window.TaskMonitorProgressSSE) {
                window.TaskMonitorProgressSSE.close();
            }
            if (window.TaskMonitorLogsSSE) {
                window.TaskMonitorLogsSSE.close();
            }
        }
    };
})();
